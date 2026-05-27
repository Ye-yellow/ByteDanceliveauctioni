package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	auctionbiz "live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
	"live-auction-bid/backend/app/auction/service/internal/pkg/requestctx"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
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

type Hub struct {
	mu       sync.RWMutex
	rooms    map[string]map[*connection]struct{}
	snapshot SnapshotProvider
	auth     *auth.Manager
	upgrader websocket.Upgrader
}

type connection struct {
	hub       *Hub
	roomID    string
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

func NewHub(snapshot SnapshotProvider) *Hub {
	return &Hub{
		rooms:    make(map[string]map[*connection]struct{}),
		snapshot: snapshot,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (h *Hub) BindSnapshotProvider(snapshot SnapshotProvider) {
	h.snapshot = snapshot
}

func (h *Hub) BindAuthManager(authManager *auth.Manager) {
	h.auth = authManager
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
	ctx, authCtx := h.authContextFromUpgrade(r)
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &connection{
		hub:     h,
		roomID:  roomID,
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

func (h *Hub) authContextFromUpgrade(r *http.Request) (context.Context, auth.AuthContext) {
	ctx := r.Context()
	if h.auth == nil {
		return ctx, auth.AuthContext{TokenStatus: auth.TokenStatusNone}
	}
	authorization := websocketAuthorization(r)
	authCtx := h.auth.AuthContextFromBearer(authorization)
	if authCtx.TokenStatus == auth.TokenStatusValid {
		ctx = auth.WithAuthContext(ctx, authCtx)
	}
	return ctx, authCtx
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
	authCtx := c.hub.auth.AuthContextFromBearer(authorization)
	if authCtx.TokenStatus != auth.TokenStatusValid {
		_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid auth"), time.Now().Add(writeTimeout))
		c.close()
		return
	}
	c.mu.Lock()
	c.authCtx = authCtx
	c.ctx = auth.WithAuthContext(c.ctx, authCtx)
	c.client = requestctx.Snapshot(c.ctx)
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
	if c.authCtx.TokenStatus != auth.TokenStatusValid || c.authCtx.Claims == nil {
		return false
	}
	switch c.authCtx.Claims.Role {
	case v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR, v1.UserRole_USER_ROLE_ADMIN:
		return true
	default:
		return false
	}
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
	viewer := c.lotResultViewer()
	if viewer.CanViewPrivateAuctionData() {
		return event
	}
	return auctionbiz.EventForViewer(event, viewer)
}

func (c *connection) lotResultViewer() auctionbiz.LotResultViewer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.authCtx.TokenStatus != auth.TokenStatusValid || c.authCtx.Claims == nil {
		return auctionbiz.LotResultViewer{}
	}
	return auctionbiz.LotResultViewer{
		UserID: c.authCtx.Claims.UserID,
		Role:   c.authCtx.Claims.Role,
	}
}

func (c *connection) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}
