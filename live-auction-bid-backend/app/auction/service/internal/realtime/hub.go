package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	auctionbiz "live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
	"live-auction-bid/backend/app/auction/service/internal/pkg/requestctx"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

const (
	writeTimeout         = 3 * time.Second
	pingInterval         = 20 * time.Second
	pongTimeout          = 60 * time.Second
	connectionSendBuffer = 32
)

var eventJSONMarshal = protojson.MarshalOptions{
	UseEnumNumbers: false,
	UseProtoNames:  false,
}

type SnapshotProvider interface {
	Snapshot(ctx context.Context, roomID string) (*v1.RoomSnapshot, error)
}

type RoomAccessValidator interface {
	ValidateRoomInMainAccount(ctx context.Context, roomID, mainAccountID string) error
}

type Hub struct {
	mu             sync.RWMutex
	rooms          map[string]map[*connection]struct{}
	snapshot       SnapshotProvider
	roomAccess     RoomAccessValidator
	auth           *auth.Manager
	config         Config
	allowedOrigins map[string]struct{}
	tickets        wsTicketCodec
	upgrader       websocket.Upgrader
}

type connection struct {
	hub       *Hub
	roomID    string
	scope     string
	mu        sync.RWMutex
	ctx       context.Context
	authCtx   auth.AuthContext
	client    requestctx.RequestContext
	conn      *websocket.Conn
	send      chan v1.AuctionEvent
	done      chan struct{}
	closeOnce sync.Once
}

type clientMessage struct {
	Type          string `json:"type"`
	AccessToken   string `json:"accessToken"`
	Authorization string `json:"authorization"`
}

func NewHub(snapshot SnapshotProvider, configs ...Config) *Hub {
	cfg := DefaultConfig()
	if len(configs) > 0 {
		cfg = configs[0]
	}
	normalized, err := NormalizeConfig(cfg)
	if err != nil {
		panic(err)
	}
	allowedOrigins := make(map[string]struct{}, len(normalized.AllowedOrigins))
	for _, origin := range normalized.AllowedOrigins {
		allowedOrigins[origin] = struct{}{}
	}
	h := &Hub{
		rooms:          make(map[string]map[*connection]struct{}),
		snapshot:       snapshot,
		config:         normalized,
		allowedOrigins: allowedOrigins,
		tickets:        newWSTicketCodec(normalized),
	}
	h.upgrader = websocket.Upgrader{CheckOrigin: h.checkOrigin}
	return h
}

func (h *Hub) BindSnapshotProvider(snapshot SnapshotProvider) {
	h.snapshot = snapshot
}

func (h *Hub) BindAuthManager(authManager *auth.Manager) {
	h.auth = authManager
}

func (h *Hub) BindRoomAccessValidator(validator RoomAccessValidator) {
	h.roomAccess = validator
}

func (h *Hub) Publish(ctx context.Context, event v1.AuctionEvent) error {
	for _, conn := range h.roomConnections(event.RoomId) {
		select {
		case conn.send <- event:
		case <-conn.done:
		default:
			h.leave(conn)
			conn.close()
		}
	}
	return nil
}

func (h *Hub) ServeRoom(w http.ResponseWriter, r *http.Request, roomID string) {
	ctx, authCtx, scope, ok := h.authenticateUpgrade(w, r, roomID)
	if !ok {
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &connection{
		hub:     h,
		roomID:  roomID,
		scope:   scope,
		ctx:     ctx,
		authCtx: authCtx,
		client:  requestctx.Snapshot(ctx),
		conn:    conn,
		send:    make(chan v1.AuctionEvent, connectionSendBuffer),
		done:    make(chan struct{}),
	}
	h.join(client)
	defer func() {
		h.leave(client)
		client.close()
	}()

	h.enqueueSnapshot(client.ctx, client)
	go client.writePump()
	client.readPump()
}

func (h *Hub) checkOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return h.config.AllowMissingOrigin
	}
	normalized, ok := normalizeOrigin(origin)
	if !ok {
		return false
	}
	if _, ok := h.allowedOrigins[normalized]; ok {
		return true
	}
	if !isProdEnv(h.config.Environment) && isLocalhostOrigin(normalized) {
		return true
	}
	return false
}

func (h *Hub) authenticateUpgrade(w http.ResponseWriter, r *http.Request, roomID string) (context.Context, auth.AuthContext, string, bool) {
	scope, ok := normalizeScope(r.URL.Query().Get("scope"))
	if !ok {
		http.Error(w, "invalid websocket scope", http.StatusBadRequest)
		return r.Context(), auth.AuthContext{}, "", false
	}
	ticket := strings.TrimSpace(r.URL.Query().Get("ticket"))
	if scope == ScopePublic {
		if ticket == "" {
			return r.Context(), auth.AuthContext{TokenStatus: auth.TokenStatusNone}, scope, true
		}
		ctx, authCtx, err := h.authContextFromTicket(r.Context(), ticket, roomID, scope)
		if err != nil {
			http.Error(w, "invalid websocket ticket", http.StatusUnauthorized)
			return r.Context(), auth.AuthContext{}, "", false
		}
		return ctx, authCtx, scope, true
	}
	if ticket == "" {
		http.Error(w, "websocket ticket is required", http.StatusUnauthorized)
		return r.Context(), auth.AuthContext{}, "", false
	}
	ctx, authCtx, err := h.authContextFromTicket(r.Context(), ticket, roomID, scope)
	if err != nil {
		http.Error(w, "invalid websocket ticket", http.StatusUnauthorized)
		return r.Context(), auth.AuthContext{}, "", false
	}
	if !canOpenAdminScope(authCtx.Claims) {
		http.Error(w, "websocket admin scope is forbidden", http.StatusForbidden)
		return r.Context(), auth.AuthContext{}, "", false
	}
	mainAccountID := auth.EffectiveMainAccountID(authCtx.Claims)
	if mainAccountID == "" {
		http.Error(w, "main account id is required", http.StatusForbidden)
		return r.Context(), auth.AuthContext{}, "", false
	}
	if h.roomAccess == nil {
		http.Error(w, "room access validator is not configured", http.StatusServiceUnavailable)
		return r.Context(), auth.AuthContext{}, "", false
	}
	if err := h.roomAccess.ValidateRoomInMainAccount(ctx, roomID, mainAccountID); err != nil {
		http.Error(w, "room access denied", http.StatusForbidden)
		return r.Context(), auth.AuthContext{}, "", false
	}
	return ctx, authCtx, scope, true
}

func (h *Hub) authContextFromAuthorization(ctx context.Context, authorization string) (context.Context, auth.AuthContext) {
	if h.auth == nil {
		return ctx, auth.AuthContext{TokenStatus: auth.TokenStatusNone}
	}
	authCtx := h.auth.AuthContextFromBearer(authorization)
	if authCtx.TokenStatus == auth.TokenStatusValid {
		ctx = auth.WithClaims(ctx, authCtx.Claims)
	}
	return ctx, authCtx
}

func (h *Hub) authContextFromTicket(ctx context.Context, ticket, roomID, scope string) (context.Context, auth.AuthContext, error) {
	claims, err := h.tickets.parse(ticket, roomID, scope)
	if err != nil {
		return ctx, auth.AuthContext{}, err
	}
	authCtx := authContextFromTicketClaims(ticket, claims)
	return auth.WithClaims(ctx, authCtx.Claims), authCtx, nil
}

func websocketAuthorization(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("Authorization")); value != "" {
		return value
	}
	if value := strings.TrimSpace(r.Header.Get("authorization")); value != "" {
		return value
	}
	return ""
}

func (h *Hub) join(conn *connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[conn.roomID] == nil {
		h.rooms[conn.roomID] = make(map[*connection]struct{})
	}
	h.rooms[conn.roomID][conn] = struct{}{}
}

func (h *Hub) leave(conn *connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.rooms[conn.roomID], conn)
	if len(h.rooms[conn.roomID]) == 0 {
		delete(h.rooms, conn.roomID)
	}
}

func (h *Hub) roomConnections(roomID string) []*connection {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connections := make([]*connection, 0, len(h.rooms[roomID]))
	for conn := range h.rooms[roomID] {
		connections = append(connections, conn)
	}
	return connections
}

func (h *Hub) RoomPresence(roomID string) *v1.RoomPresence {
	connections := h.roomConnections(roomID)
	presence := &v1.RoomPresence{
		RoomId:           roomID,
		TotalConnections: int32(len(connections)),
		ServerTimeUnixMs: clock.NowMs(),
	}
	for _, conn := range connections {
		if conn.canReceivePrivateEvents() {
			presence.OperatorConnections++
			continue
		}
		presence.ViewerConnections++
	}
	return presence
}

func (h *Hub) enqueueSnapshot(ctx context.Context, conn *connection) {
	if h.snapshot == nil {
		return
	}
	snapshot, err := h.snapshot.Snapshot(ctx, conn.roomID)
	if err != nil {
		return
	}
	event := v1.AuctionEvent{
		Id:               idgen.New("evt"),
		Type:             v1.AuctionEventType_AUCTION_EVENT_TYPE_ROOM_SNAPSHOT,
		RoomId:           conn.roomID,
		OccurredAtUnixMs: clock.NowMs(),
		Snapshot:         snapshot,
	}
	if mainAccountID := snapshotMainAccountID(snapshot); mainAccountID != "" {
		event.MainAccountId = mainAccountID
	}
	select {
	case conn.send <- event:
	case <-conn.done:
	default:
		h.leave(conn)
		conn.close()
	}
}

func (c *connection) readPump() {
	_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		c.handleClientMessage(payload)
	}
}

func (c *connection) handleClientMessage(payload []byte) {
	var msg clientMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	if !strings.EqualFold(msg.Type, "AUTH") {
		return
	}
	if c.scope != ScopePublic {
		return
	}
	if c.hub.auth == nil {
		return
	}
	token := strings.TrimSpace(msg.AccessToken)
	authorization := strings.TrimSpace(msg.Authorization)
	if authorization == "" && token != "" {
		authorization = "Bearer " + token
	}
	if authorization == "" {
		return
	}
	ctx, authCtx := c.hub.authContextFromAuthorization(c.context(), authorization)
	if authCtx.TokenStatus != auth.TokenStatusValid {
		_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid auth"), time.Now().Add(writeTimeout))
		c.close()
		return
	}
	c.mu.Lock()
	c.authCtx = authCtx
	c.ctx = ctx
	c.client = requestctx.Snapshot(ctx)
	c.mu.Unlock()
	c.hub.enqueueSnapshot(c.context(), c)
}

func (c *connection) context() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

func (c *connection) canReceivePrivateEvents() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.scope != ScopeAdmin {
		return false
	}
	if c.authCtx.TokenStatus != auth.TokenStatusValid || c.authCtx.Claims == nil {
		return false
	}
	return auth.HasAnyPermission(c.authCtx.Claims, userbiz.PermissionRealtimeView, userbiz.PermissionAuctionControl, userbiz.PermissionLotViewAdmin)
}

func (c *connection) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	defer func() {
		c.hub.leave(c)
		c.close()
	}()

	for {
		select {
		case event := <-c.send:
			select {
			case <-c.done:
				return
			default:
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			event = c.eventForDelivery(event)
			payload, err := eventJSONMarshal.Marshal(&event)
			if err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *connection) eventForDelivery(event v1.AuctionEvent) v1.AuctionEvent {
	viewer := c.deliveryViewer()
	if event.GetType() == v1.AuctionEventType_AUCTION_EVENT_TYPE_ROOM_SNAPSHOT && event.GetSnapshot() != nil {
		mainAccountID := event.GetMainAccountId()
		if mainAccountID == "" {
			mainAccountID = snapshotMainAccountID(event.GetSnapshot())
		}
		if viewer.CanViewMainAccountPrivate(mainAccountID) {
			if event.GetMainAccountId() == "" && mainAccountID != "" {
				cloned := proto.Clone(&event).(*v1.AuctionEvent)
				cloned.MainAccountId = mainAccountID
				return *cloned
			}
			return event
		}
		cloned := proto.Clone(&event).(*v1.AuctionEvent)
		cloned.Snapshot = auctionbiz.SnapshotForViewer(event.GetSnapshot(), viewer)
		cloned.MainAccountId = ""
		return *cloned
	}
	return auctionbiz.EventForViewer(event, viewer)
}

func (c *connection) deliveryViewer() auctionbiz.LotResultViewer {
	viewer := c.lotResultViewer()
	if c.scope == ScopeAdmin {
		return viewer
	}
	public := auctionbiz.LotResultViewer{UserID: viewer.UserID}
	for _, permission := range viewer.PermissionCodes {
		if userbiz.NormalizePermissionCode(permission) == userbiz.PermissionOrderViewOwn {
			public.PermissionCodes = append(public.PermissionCodes, userbiz.PermissionOrderViewOwn)
			break
		}
	}
	return public
}

func (c *connection) lotResultViewer() auctionbiz.LotResultViewer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.authCtx.TokenStatus != auth.TokenStatusValid || c.authCtx.Claims == nil {
		return auctionbiz.LotResultViewer{}
	}
	return auctionbiz.LotResultViewer{
		UserID:          c.authCtx.Claims.UserID,
		MainAccountID:   auth.EffectiveMainAccountID(c.authCtx.Claims),
		RoleCodes:       append([]string(nil), c.authCtx.Claims.RoleCodes...),
		PermissionCodes: append([]string(nil), c.authCtx.Claims.PermissionCodes...),
	}
}

func snapshotMainAccountID(snapshot *v1.RoomSnapshot) string {
	if snapshot == nil || snapshot.GetCurrentLot() == nil {
		return ""
	}
	return strings.TrimSpace(snapshot.GetCurrentLot().GetMainAccountId())
}

func canOpenAdminScope(claims *auth.Claims) bool {
	if claims == nil || !auth.HasPermission(claims, userbiz.PermissionRealtimeView) {
		return false
	}
	return auth.HasRoleCode(claims, userbiz.RoleMerchantOwner) || auth.HasRoleCode(claims, userbiz.RoleAnchor) || auth.HasRoleCode(claims, userbiz.RoleOperator)
}

func (c *connection) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}
