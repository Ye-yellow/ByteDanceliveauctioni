package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"live-auction-bid/backend/app/auction/service/internal/cluster"
)

func registerAdminHTTP(mux *http.ServeMux, gateway *cluster.Gateway, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	admin := gatewayAdmin{gateway: gateway, token: token}
	mux.HandleFunc("/cluster/admin/shards", admin.handleShards)
	mux.HandleFunc("/cluster/admin/shards/status", admin.handleShardStatus)
	mux.HandleFunc("/cluster/admin/shards/failover", admin.handleShardFailover)
	mux.HandleFunc("/cluster/admin/rooms/assign", admin.handleRoomAssignment)
	mux.HandleFunc("/cluster/admin/autoscale/evaluate", admin.handleAutoscaleEvaluate)
}

type gatewayAdmin struct {
	gateway *cluster.Gateway
	token   string
}

func (a gatewayAdmin) authorized(w http.ResponseWriter, r *http.Request) bool {
	if constantTimeToken(r.Header.Get("X-Auction-Admin-Token"), a.token) {
		return true
	}
	if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		return constantTimeToken(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), a.token)
	}
	writeAdminJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "admin token is required"})
	return false
}

func (a gatewayAdmin) handleShards(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut:
		var shard cluster.Shard
		if err := json.NewDecoder(r.Body).Decode(&shard); err != nil {
			writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid shard payload"})
			return
		}
		if err := a.gateway.UpsertShard(shard); err != nil {
			writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"ok": true, "shard": shard})
	case http.MethodDelete:
		id, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("id")))
		if err != nil {
			writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "id is required"})
			return
		}
		if err := a.gateway.RemoveShard(id); err != nil {
			writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"ok": true, "removed": id})
	default:
		writeAdminJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method not allowed"})
	}
}

func (a gatewayAdmin) handleShardStatus(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		writeAdminJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method not allowed"})
		return
	}
	var req struct {
		ShardID int                 `json:"shardId"`
		Status  cluster.ShardStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid status payload"})
		return
	}
	if err := a.gateway.SetShardStatus(req.ShardID, req.Status); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeAdminJSON(w, http.StatusOK, map[string]any{"ok": true, "shardId": req.ShardID, "status": req.Status})
}

func (a gatewayAdmin) handleShardFailover(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeAdminJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method not allowed"})
		return
	}
	var req struct {
		SourceShardID             int      `json:"sourceShardId"`
		TargetShardID             *int     `json:"targetShardId,omitempty"`
		RoomIDs                   []string `json:"roomIds,omitempty"`
		MarkSourceOffline         bool     `json:"markSourceOffline,omitempty"`
		IncludeHotDedicatedTarget bool     `json:"includeHotDedicatedTarget,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid failover payload"})
		return
	}
	result, err := a.gateway.FailoverRooms(r.Context(), cluster.FailoverRequest{
		SourceShardID:             req.SourceShardID,
		TargetShardID:             req.TargetShardID,
		RoomIDs:                   req.RoomIDs,
		MarkSourceOffline:         req.MarkSourceOffline,
		IncludeHotDedicatedTarget: req.IncludeHotDedicatedTarget,
	})
	if err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeAdminJSON(w, http.StatusOK, map[string]any{"ok": true, "failover": result})
}

func (a gatewayAdmin) handleRoomAssignment(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut:
		var req struct {
			RoomID  string `json:"roomId"`
			ShardID int    `json:"shardId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid assignment payload"})
			return
		}
		if err := a.gateway.AssignRoomToShard(req.RoomID, req.ShardID); err != nil {
			writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"ok": true, "roomId": strings.TrimSpace(req.RoomID), "shardId": req.ShardID})
	case http.MethodDelete:
		roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
		if err := a.gateway.ClearRoomAssignment(roomID); err != nil {
			writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{"ok": true, "roomId": roomID})
	default:
		writeAdminJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method not allowed"})
	}
}

func (a gatewayAdmin) handleAutoscaleEvaluate(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeAdminJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method not allowed"})
		return
	}
	var req struct {
		Policy  cluster.ScalePolicy  `json:"policy"`
		Signals cluster.ScaleSignals `json:"signals"`
		Apply   bool                 `json:"apply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid autoscale payload"})
		return
	}
	rec := a.gateway.EvaluateScale(req.Policy, req.Signals)
	if req.Apply && rec.Safe {
		if err := a.gateway.ApplyScaleRecommendation(rec); err != nil {
			writeAdminJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "recommendation": rec, "error": err.Error()})
			return
		}
	}
	writeAdminJSON(w, http.StatusOK, map[string]any{"ok": true, "recommendation": rec, "applied": req.Apply && rec.Safe})
}

func constantTimeToken(got, want string) bool {
	got = strings.TrimSpace(got)
	want = strings.TrimSpace(want)
	if got == "" || want == "" || len(got) != len(want) {
		return false
	}
	var diff byte
	for i := range got {
		diff |= got[i] ^ want[i]
	}
	return diff == 0
}

func writeAdminJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
