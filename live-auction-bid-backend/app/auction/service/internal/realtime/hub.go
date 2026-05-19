package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"live-auction-bid/backend/app/auction/service/internal/biz"
)

type SnapshotProvider interface { Snapshot(ctx context.Context, roomID string) (*biz.RoomSnapshot, error) }

type Hub struct { mu sync.RWMutex; rooms map[string]map[*websocket.Conn]bool; snapshot SnapshotProvider; upgrader websocket.Upgrader }
func NewHub(snapshot SnapshotProvider) *Hub { return &Hub{rooms: map[string]map[*websocket.Conn]bool{}, snapshot: snapshot, upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}} }
func (h *Hub) Publish(ctx context.Context, event biz.AuctionEvent) { h.mu.RLock(); conns := h.rooms[event.RoomID]; for c := range conns { _ = c.SetWriteDeadline(time.Now().Add(3*time.Second)); _ = c.WriteJSON(event) }; h.mu.RUnlock() }
func (h *Hub) ServeRoom(w http.ResponseWriter, r *http.Request, roomID string) { c, err := h.upgrader.Upgrade(w,r,nil); if err != nil { return }; h.mu.Lock(); if h.rooms[roomID] == nil { h.rooms[roomID] = map[*websocket.Conn]bool{} }; h.rooms[roomID][c] = true; h.mu.Unlock(); defer func(){ h.mu.Lock(); delete(h.rooms[roomID], c); h.mu.Unlock(); _ = c.Close() }(); if h.snapshot != nil { if snap, err := h.snapshot.Snapshot(r.Context(), roomID); err == nil { _ = c.WriteJSON(biz.AuctionEvent{ID:"evt_snapshot", Type: biz.EventRoomSnapshot, RoomID: roomID, OccurredAtUnixMs: biz.NowMs(), Snapshot: snap}) } }; for { _, data, err := c.ReadMessage(); if err != nil { return }; var raw map[string]any; _ = json.Unmarshal(data, &raw) } }
