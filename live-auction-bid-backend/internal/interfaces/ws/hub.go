package ws

import (
	"context"
	"encoding/json"
	"sync"

	domain "live-auction-bid/backend/internal/domain/auction"
)

type Envelope struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Client interface{ SendJSON(v interface{}) error }

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[Client]bool
}

func NewHub() *Hub { return &Hub{clients: map[string]map[Client]bool{}} }

func (h *Hub) Join(roomID string, c Client) func() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[roomID] == nil { h.clients[roomID] = map[Client]bool{} }
	h.clients[roomID][c] = true
	return func() {
		h.mu.Lock(); defer h.mu.Unlock()
		delete(h.clients[roomID], c)
	}
}

func (h *Hub) Broadcast(roomID string, msg Envelope) {
	h.mu.RLock()
	clients := make([]Client, 0, len(h.clients[roomID]))
	for c := range h.clients[roomID] { clients = append(clients, c) }
	h.mu.RUnlock()
	for _, c := range clients { _ = c.SendJSON(msg) }
}

func (h *Hub) PublishLotUpdated(ctx context.Context, lot *domain.Lot) error {
	h.Broadcast(lot.RoomID, Envelope{Type: "lot.updated", Data: lotSnapshot(lot)})
	return nil
}
func (h *Hub) PublishBidAccepted(ctx context.Context, lot *domain.Lot, bid domain.Bid) error {
	h.Broadcast(lot.RoomID, Envelope{Type: "bid.accepted", Data: map[string]interface{}{"bid": bid, "lot": lotSnapshot(lot)}})
	return nil
}
func (h *Hub) PublishLotSettled(ctx context.Context, lot *domain.Lot) error {
	h.Broadcast(lot.RoomID, Envelope{Type: "lot.settled", Data: lotSnapshot(lot)})
	return nil
}

func lotSnapshot(lot *domain.Lot) map[string]interface{} {
	b, _ := json.Marshal(lot)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	m["ranking"] = lot.Ranking(10)
	return m
}
