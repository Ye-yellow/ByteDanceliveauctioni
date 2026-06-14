package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"live-auction-bid/backend/app/auction/service/internal/observability"
)

const (
	HeaderShardID = "X-Auction-Shard-ID"
	HeaderRoomID  = "X-Auction-Room-ID"
)

var (
	lotPathPattern   = regexp.MustCompile(`^/api/lots/([^/]+)(?:/.*)?$`)
	orderPathPattern = regexp.MustCompile(`^/api/orders/([^/]+)(?:/.*)?$`)
	roomPathPattern  = regexp.MustCompile(`^/(?:api/)?(?:ws/)?rooms/([^/]+)(?:/.*)?$`)
)

type Gateway struct {
	registry     *StaticRegistry
	routes       RouteTable
	defaultShard Shard
	proxyMu      sync.RWMutex
	proxies      map[int]*httputil.ReverseProxy
	client       *http.Client
}

func NewGateway(registry *StaticRegistry, routes RouteTable) (*Gateway, error) {
	if registry == nil {
		return nil, errors.New("registry is required")
	}
	if routes == nil {
		var err error
		routes, err = NewMemoryRouteTable(registry)
		if err != nil {
			return nil, err
		}
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Shards) == 0 {
		return nil, errors.New("registry has no shards")
	}
	proxies := make(map[int]*httputil.ReverseProxy, len(snapshot.Shards))
	var defaultShard Shard
	for i, shard := range snapshot.Shards {
		target, err := url.Parse(shard.BackendURL)
		if err != nil {
			return nil, fmt.Errorf("parse shard %d backend url: %w", shard.ID, err)
		}
		proxies[shard.ID] = newShardProxy(shard, target, routes)
		if i == 0 || shard.ID == 0 {
			defaultShard = shard
		}
	}
	gateway := &Gateway{
		registry:     registry,
		routes:       routes,
		defaultShard: defaultShard,
		proxies:      proxies,
		client:       &http.Client{Timeout: 5 * time.Second},
	}
	observeGatewaySnapshot(gateway.registry.Snapshot())
	return gateway, nil
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if field, ok := aggregateCollectionField(r); ok {
		g.serveAggregateCollection(w, r, field)
		return
	}
	shard, route, err := g.routeRequest(r)
	if err != nil {
		observability.RecordGatewayRoute(route, -1, "error")
		writeGatewayError(w, http.StatusBadGateway, err)
		return
	}
	proxy, err := g.proxyForShard(shard)
	if err != nil {
		observability.RecordGatewayRoute(route, shard.ID, "error")
		writeGatewayError(w, http.StatusBadGateway, err)
		return
	}
	observability.RecordGatewayRoute(route, shard.ID, "ok")
	r.Header.Set(HeaderShardID, fmt.Sprintf("%d", shard.ID))
	if roomID := roomIDFromRequest(r); roomID != "" {
		r.Header.Set(HeaderRoomID, roomID)
	}
	proxy.ServeHTTP(w, r)
}

func (g *Gateway) routeRequest(r *http.Request) (Shard, string, error) {
	if shardID, ok := ShardIDFromHeader(r.Header.Get(HeaderShardID)); ok {
		shard, exists := g.registry.LookupShard(shardID)
		if exists && shard.ServesExistingRooms() {
			return shard, "header", nil
		}
		return Shard{}, "header", fmt.Errorf("header references unavailable shard %d", shardID)
	}
	if roomID := roomIDFromRequest(r); roomID != "" {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			shard, err := g.routes.AssignRoom(r.Context(), roomID)
			return shard, "room", err
		}
		if shard, ok, err := g.routes.ResolveRoom(r.Context(), roomID); err != nil {
			return Shard{}, "room", err
		} else if ok {
			return shard, "room", nil
		}
		shard, err := g.routes.AssignRoom(r.Context(), roomID)
		return shard, "room", err
	}
	if lotID := lotIDFromPath(r.URL.Path); lotID != "" {
		if shard, ok, err := g.routes.ResolveLot(r.Context(), lotID); err != nil {
			return Shard{}, "lot", err
		} else if ok {
			return shard, "lot", nil
		}
		slog.Warn("lot route missing, using default shard", "lot_id", lotID, "default_shard", g.defaultShard.ID)
		return g.defaultShard, "lot_default", nil
	}
	if orderID := orderIDFromPath(r.URL.Path); orderID != "" {
		if shard, ok, err := g.routes.ResolveOrder(r.Context(), orderID); err != nil {
			return Shard{}, "order", err
		} else if ok {
			return shard, "order", nil
		}
		slog.Warn("order route missing, using default shard", "order_id", orderID, "default_shard", g.defaultShard.ID)
		return g.defaultShard, "order_default", nil
	}
	return g.defaultShard, "default", nil
}

func (g *Gateway) proxyForShard(shard Shard) (*httputil.ReverseProxy, error) {
	g.proxyMu.RLock()
	proxy := g.proxies[shard.ID]
	g.proxyMu.RUnlock()
	if proxy != nil {
		return proxy, nil
	}
	target, err := url.Parse(shard.BackendURL)
	if err != nil {
		return nil, fmt.Errorf("parse shard %d backend url: %w", shard.ID, err)
	}
	g.proxyMu.Lock()
	defer g.proxyMu.Unlock()
	if proxy = g.proxies[shard.ID]; proxy != nil {
		return proxy, nil
	}
	proxy = newShardProxy(shard, target, g.routes)
	g.proxies[shard.ID] = proxy
	return proxy, nil
}

func (g *Gateway) UpsertShard(shard Shard) error {
	if err := g.registry.UpsertShard(shard); err != nil {
		return err
	}
	g.proxyMu.Lock()
	delete(g.proxies, shard.ID)
	g.proxyMu.Unlock()
	observeGatewaySnapshot(g.registry.Snapshot())
	return nil
}

func (g *Gateway) SetShardStatus(id int, status ShardStatus) error {
	if err := g.registry.SetShardStatus(id, status); err != nil {
		return err
	}
	observeGatewaySnapshot(g.registry.Snapshot())
	return nil
}

func (g *Gateway) RemoveShard(id int) error {
	if g.defaultShard.ID == id {
		return fmt.Errorf("default shard %d cannot be removed", id)
	}
	if err := g.registry.RemoveShard(id); err != nil {
		return err
	}
	g.proxyMu.Lock()
	delete(g.proxies, id)
	g.proxyMu.Unlock()
	observability.SetGatewayShardStatus(id, string(ShardStatusOffline))
	observeGatewaySnapshot(g.registry.Snapshot())
	return nil
}

func (g *Gateway) AssignRoomToShard(roomID string, shardID int) error {
	if err := g.routes.BindRoom(context.Background(), roomID, shardID); err != nil {
		return err
	}
	observeGatewaySnapshot(g.registry.Snapshot())
	return nil
}

func (g *Gateway) ClearRoomAssignment(roomID string) error {
	if err := g.routes.ClearRoom(context.Background(), roomID); err != nil {
		return err
	}
	observeGatewaySnapshot(g.registry.Snapshot())
	return nil
}

type FailoverRequest struct {
	SourceShardID             int
	TargetShardID             *int
	RoomIDs                   []string
	MarkSourceOffline         bool
	IncludeHotDedicatedTarget bool
}

type FailoverResult struct {
	SourceShardID int              `json:"sourceShardId"`
	TargetShardID int              `json:"targetShardId"`
	Rooms         []RoomAssignment `json:"rooms"`
}

func (g *Gateway) FailoverRooms(ctx context.Context, req FailoverRequest) (FailoverResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := g.registry.LookupShard(req.SourceShardID); !ok {
		return FailoverResult{}, fmt.Errorf("unknown source shard: %d", req.SourceShardID)
	}
	targetID, err := g.failoverTargetID(req)
	if err != nil {
		return FailoverResult{}, err
	}
	target, ok := g.registry.LookupShard(targetID)
	if !ok {
		return FailoverResult{}, fmt.Errorf("unknown target shard: %d", targetID)
	}
	if target.Status != ShardStatusActive {
		return FailoverResult{}, fmt.Errorf("target shard %d must be active", targetID)
	}
	if target.HotDedicated && !req.IncludeHotDedicatedTarget {
		return FailoverResult{}, fmt.Errorf("target shard %d is hot-dedicated; set includeHotDedicatedTarget to true to use it", targetID)
	}
	rooms := normalizedRoomList(req.RoomIDs)
	if len(rooms) == 0 {
		rooms = g.registry.RoomsOnShard(req.SourceShardID)
	}
	moved := make([]RoomAssignment, 0, len(rooms))
	for _, roomID := range rooms {
		if err := g.routes.BindRoom(ctx, roomID, targetID); err != nil {
			return FailoverResult{}, err
		}
		moved = append(moved, RoomAssignment{RoomID: roomID, ShardID: targetID})
	}
	if req.MarkSourceOffline {
		if err := g.registry.SetShardStatus(req.SourceShardID, ShardStatusOffline); err != nil {
			return FailoverResult{}, err
		}
	}
	observeGatewaySnapshot(g.registry.Snapshot())
	return FailoverResult{SourceShardID: req.SourceShardID, TargetShardID: targetID, Rooms: moved}, nil
}

func (g *Gateway) failoverTargetID(req FailoverRequest) (int, error) {
	if req.TargetShardID != nil {
		if *req.TargetShardID == req.SourceShardID {
			return 0, errors.New("target shard must be different from source shard")
		}
		return *req.TargetShardID, nil
	}
	shard, ok := g.registry.PickFailoverShard(req.SourceShardID, req.IncludeHotDedicatedTarget)
	if !ok {
		return 0, errors.New("no active failover target shard is available")
	}
	return shard.ID, nil
}

func normalizedRoomList(roomIDs []string) []string {
	rooms := make([]string, 0, len(roomIDs))
	seen := make(map[string]struct{}, len(roomIDs))
	for _, roomID := range roomIDs {
		roomID = strings.TrimSpace(roomID)
		if roomID == "" {
			continue
		}
		if _, ok := seen[roomID]; ok {
			continue
		}
		seen[roomID] = struct{}{}
		rooms = append(rooms, roomID)
	}
	return rooms
}

func (g *Gateway) EvaluateScale(policy ScalePolicy, signals ScaleSignals) ScaleRecommendation {
	return EvaluateScale(policy, signals, g.registry.Snapshot())
}

func (g *Gateway) ApplyScaleRecommendation(rec ScaleRecommendation) error {
	if err := ApplyScaleRecommendation(g.registry, rec); err != nil {
		return err
	}
	observeGatewaySnapshot(g.registry.Snapshot())
	return nil
}

func newShardProxy(shard Shard, target *url.URL, routes RouteTable) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)
		r.Host = target.Host
		r.Header.Set(HeaderShardID, fmt.Sprintf("%d", shard.ID))
		if roomID := roomIDFromRequest(r); roomID != "" {
			r.Header.Set(HeaderRoomID, roomID)
		}
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set(HeaderShardID, fmt.Sprintf("%d", shard.ID))
		if roomID := strings.TrimSpace(resp.Request.Header.Get(HeaderRoomID)); roomID != "" {
			resp.Header.Set(HeaderRoomID, roomID)
		}
		return bindRoutesFromResponse(resp.Request.Context(), routes, shard.ID, resp)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Warn("shard gateway proxy error", "shard", shard.ID, "target", target.String(), "error", err)
		writeGatewayError(w, http.StatusBadGateway, err)
	}
	return proxy
}

func bindRoutesFromResponse(ctx context.Context, routes RouteTable, shardID int, resp *http.Response) error {
	if routes == nil || resp.Body == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(data))
	resp.ContentLength = int64(len(data))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	if len(data) == 0 {
		return nil
	}
	lotID, orderID := lotAndOrderIDsFromPayload(data)
	if lotID != "" {
		if err := routes.BindLot(ctx, lotID, shardID); err != nil {
			slog.Warn("bind lot route failed", "lot_id", lotID, "shard", shardID, "error", err)
		}
	}
	if orderID != "" {
		if err := routes.BindOrder(ctx, orderID, shardID); err != nil {
			slog.Warn("bind order route failed", "order_id", orderID, "shard", shardID, "error", err)
		}
	}
	return nil
}

func roomIDFromRequest(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get(HeaderRoomID)); value != "" {
		return value
	}
	if value := strings.TrimSpace(r.URL.Query().Get("roomId")); value != "" {
		return value
	}
	if value := strings.TrimSpace(r.URL.Query().Get("room_id")); value != "" {
		return value
	}
	if matches := roomPathPattern.FindStringSubmatch(r.URL.Path); len(matches) == 2 {
		return matches[1]
	}
	if r.Body == nil || (r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch) {
		return ""
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return ""
	}
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(data))
	return roomIDFromPayload(data)
}

func roomIDFromPayload(data []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	return firstString(payload, "roomId", "room_id", "roomID")
}

func lotIDFromPath(path string) string {
	matches := lotPathPattern.FindStringSubmatch(path)
	if len(matches) != 2 {
		return ""
	}
	lotID := matches[1]
	if lotID == "drafts" {
		return ""
	}
	return lotID
}

func orderIDFromPath(path string) string {
	matches := orderPathPattern.FindStringSubmatch(path)
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}

func lotAndOrderIDsFromPayload(data []byte) (string, string) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", ""
	}
	lotID := nestedID(payload, "lot")
	if lotID == "" {
		lotID = firstString(payload, "lotId", "lot_id")
	}
	orderID := nestedID(payload, "order")
	if orderID == "" {
		orderID = firstString(payload, "orderId", "order_id")
	}
	return lotID, orderID
}

func nestedID(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	return firstString(m, "id", "ID")
}

func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		if value, ok := raw.(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func writeGatewayError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"result": map[string]any{
			"code":    status,
			"message": err.Error(),
			"ok":      false,
		},
	})
}
