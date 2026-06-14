package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"live-auction-bid/backend/app/auction/service/internal/observability"
)

type aggregateShardResult struct {
	shard   Shard
	status  int
	payload map[string]any
	err     error
}

func aggregateCollectionField(r *http.Request) (string, bool) {
	if r == nil || r.Method != http.MethodGet {
		return "", false
	}
	if _, ok := ShardIDFromHeader(r.Header.Get(HeaderShardID)); ok {
		return "", false
	}
	if roomIDFromRequest(r) != "" {
		return "", false
	}
	switch r.URL.Path {
	case "/api/admin/lots":
		return "lots", true
	case "/api/admin/orders", "/api/me/orders":
		return "orders", true
	case "/api/me/bids":
		return "bids", true
	case "/api/admin/rooms", "/api/rooms":
		return "rooms", true
	default:
		return "", false
	}
}

func (g *Gateway) serveAggregateCollection(w http.ResponseWriter, r *http.Request, field string) {
	results := g.fetchAggregateCollection(r)
	items := make([]any, 0)
	shards := make([]map[string]any, 0, len(results))
	var total int64
	var firstResult any
	var firstPage, firstPageSize any
	successes := 0
	for _, result := range results {
		aggregateResult := "ok"
		if result.err != nil || result.status < 200 || result.status >= 300 {
			aggregateResult = "error"
		}
		observability.RecordGatewayAggregate(r.URL.Path, result.shard.ID, aggregateResult)
		meta := map[string]any{
			"id":     result.shard.ID,
			"name":   result.shard.Name,
			"status": result.shard.Status,
		}
		if result.err != nil {
			meta["ok"] = false
			meta["error"] = result.err.Error()
			shards = append(shards, meta)
			continue
		}
		meta["ok"] = result.status >= 200 && result.status < 300
		meta["httpStatus"] = result.status
		shards = append(shards, meta)
		if result.status < 200 || result.status >= 300 {
			continue
		}
		successes++
		g.bindAggregateRoutes(r.Context(), result.shard.ID, field, result.payload)
		if firstResult == nil {
			firstResult = result.payload["result"]
			firstPage = result.payload["page"]
			firstPageSize = result.payload["pageSize"]
		}
		if values, ok := result.payload[field].([]any); ok {
			items = append(items, values...)
			if _, hasTotal := result.payload["total"]; !hasTotal {
				total += int64(len(values))
			}
		}
		if value, ok := numericInt64(result.payload["total"]); ok {
			total += value
		}
	}
	if successes == 0 {
		writeGatewayError(w, http.StatusBadGateway, fmt.Errorf("all shards failed for %s", r.URL.Path))
		return
	}
	if firstResult == nil {
		firstResult = map[string]any{"code": 0, "message": "ok", "ok": true}
	}
	payload := map[string]any{
		"result":  firstResult,
		field:     items,
		"shards":  shards,
		"partial": successes < len(results),
	}
	if field != "rooms" {
		payload["total"] = total
		if firstPage != nil {
			payload["page"] = firstPage
		} else if page := strings.TrimSpace(r.URL.Query().Get("page")); page != "" {
			payload["page"] = page
		}
		if firstPageSize != nil {
			payload["pageSize"] = firstPageSize
		} else if size := firstNonEmpty(r.URL.Query().Get("pageSize"), r.URL.Query().Get("size")); size != "" {
			payload["pageSize"] = size
		}
	}
	writeGatewayJSON(w, http.StatusOK, payload)
}

func (g *Gateway) bindAggregateRoutes(ctx context.Context, shardID int, field string, payload map[string]any) {
	if g.routes == nil || payload == nil {
		return
	}
	values, ok := payload[field].([]any)
	if !ok {
		return
	}
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		switch field {
		case "lots":
			if lotID := firstString(item, "id", "ID", "lotId", "lot_id"); lotID != "" {
				_ = g.routes.BindLot(ctx, lotID, shardID)
			}
		case "orders":
			if orderID := firstString(item, "id", "ID", "orderId", "order_id"); orderID != "" {
				_ = g.routes.BindOrder(ctx, orderID, shardID)
			}
			if lotID := firstString(item, "lotId", "lot_id"); lotID != "" {
				_ = g.routes.BindLot(ctx, lotID, shardID)
			}
		}
	}
}

func (g *Gateway) fetchAggregateCollection(r *http.Request) []aggregateShardResult {
	snapshot := g.registry.Snapshot()
	results := make([]aggregateShardResult, 0, len(snapshot.Shards))
	for _, shard := range snapshot.Shards {
		if !shard.ServesExistingRooms() {
			continue
		}
		payload, status, err := g.fetchShardJSON(r, shard)
		results = append(results, aggregateShardResult{shard: shard, status: status, payload: payload, err: err})
	}
	return results
}

func (g *Gateway) fetchShardJSON(r *http.Request, shard Shard) (map[string]any, int, error) {
	target, err := url.Parse(shard.BackendURL)
	if err != nil {
		return nil, 0, err
	}
	target.Path = singleJoiningSlash(target.Path, r.URL.Path)
	target.RawQuery = r.URL.RawQuery
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	copyHTTPHeaders(req.Header, r.Header)
	req.Header.Set(HeaderShardID, strconv.Itoa(shard.ID))
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if len(data) == 0 {
		return map[string]any{}, resp.StatusCode, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, resp.StatusCode, err
	}
	return payload, resp.StatusCode, nil
}

func copyHTTPHeaders(dst, src http.Header) {
	for key, values := range src {
		if strings.EqualFold(key, "Host") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}

func numericInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func writeGatewayJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (g *Gateway) ClusterSnapshot(ctx context.Context) map[string]any {
	snapshot := g.registry.Snapshot()
	observeGatewaySnapshot(snapshot)
	shards := make([]map[string]any, 0, len(snapshot.Shards))
	for _, shard := range snapshot.Shards {
		item := map[string]any{
			"id":            shard.ID,
			"name":          shard.Name,
			"status":        shard.Status,
			"backendUrl":    shard.BackendURL,
			"webSocketUrl":  shard.WebSocketURL,
			"hotDedicated":  shard.HotDedicated,
			"maxActiveRoom": shard.MaxActiveRoom,
			"readyz":        g.fetchShardStatus(ctx, shard, "/readyz"),
			"workers":       g.fetchShardStatus(ctx, shard, "/workerz"),
			"projection":    g.fetchShardStatus(ctx, shard, "/metrics/runtime-projection"),
		}
		shards = append(shards, item)
	}
	return map[string]any{
		"ok":          true,
		"mode":        "gateway",
		"registry":    snapshot,
		"shards":      shards,
		"assignments": snapshot.Assignments,
	}
}

func observeGatewaySnapshot(snapshot Snapshot) {
	assignmentCounts := make(map[int]int, len(snapshot.Shards))
	for _, shard := range snapshot.Shards {
		assignmentCounts[shard.ID] = 0
		observability.SetGatewayShardStatus(shard.ID, string(shard.Status))
	}
	for _, assignment := range snapshot.Assignments {
		assignmentCounts[assignment.ShardID]++
	}
	observability.SetGatewayRoomAssignments(assignmentCounts)
}

func (g *Gateway) fetchShardStatus(ctx context.Context, shard Shard, path string) map[string]any {
	target, err := url.Parse(shard.BackendURL)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	target.Path = singleJoiningSlash(target.Path, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	req.Header.Set(HeaderShardID, strconv.Itoa(shard.ID))
	resp, err := g.client.Do(req)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]any{"ok": false, "httpStatus": resp.StatusCode, "error": err.Error()}
	}
	var payload any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &payload); err != nil {
			return map[string]any{"ok": false, "httpStatus": resp.StatusCode, "error": err.Error()}
		}
	}
	return map[string]any{
		"ok":         resp.StatusCode >= 200 && resp.StatusCode < 300,
		"httpStatus": resp.StatusCode,
		"payload":    payload,
	}
}
