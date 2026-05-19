package realtime

import "sync"

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[Client]bool
}

func NewHub() *Hub { return &Hub{clients: map[string]map[Client]bool{}} }

func (h *Hub) Join(roomID string, c Client) func() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[roomID] == nil {
		h.clients[roomID] = map[Client]bool{}
	}
	h.clients[roomID][c] = true
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.clients[roomID], c)
	}
}

func (h *Hub) Broadcast(roomID string, msg Envelope) {
	h.mu.RLock()
	clients := make([]Client, 0, len(h.clients[roomID]))
	for c := range h.clients[roomID] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		_ = c.SendJSON(msg)
	}
}

func (h *Hub) OnlineCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[roomID])
}
