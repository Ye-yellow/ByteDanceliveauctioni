package realtime

import (
	"context"
	"net/http"
	"sync"
	"time"

	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"

	"github.com/gorilla/websocket"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

const writeTimeout = 3 * time.Second

type SnapshotProvider interface {
	Snapshot(ctx context.Context, roomID string) (*v1.RoomSnapshot, error)
}

type Hub struct {
	mu       sync.RWMutex
	rooms    map[string]map[*websocket.Conn]struct{}
	snapshot SnapshotProvider
	upgrader websocket.Upgrader
}

func NewHub(snapshot SnapshotProvider) *Hub {
	return &Hub{
		rooms:    make(map[string]map[*websocket.Conn]struct{}),
		snapshot: snapshot,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (h *Hub) BindSnapshotProvider(snapshot SnapshotProvider) {
	h.snapshot = snapshot
}

func (h *Hub) Publish(ctx context.Context, event v1.AuctionEvent) error {
	for _, conn := range h.roomConnections(event.RoomId) {
		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		_ = conn.WriteJSON(event)
	}
	return nil
}

func (h *Hub) ServeRoom(w http.ResponseWriter, r *http.Request, roomID string) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	h.join(roomID, conn)
	defer h.leave(roomID, conn)

	h.sendSnapshot(r.Context(), roomID, conn)
	h.drain(conn)
}

func (h *Hub) join(roomID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*websocket.Conn]struct{})
	}
	h.rooms[roomID][conn] = struct{}{}
}

func (h *Hub) leave(roomID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.rooms[roomID], conn)
	if len(h.rooms[roomID]) == 0 {
		delete(h.rooms, roomID)
	}
}

func (h *Hub) roomConnections(roomID string) []*websocket.Conn {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connections := make([]*websocket.Conn, 0, len(h.rooms[roomID]))
	for conn := range h.rooms[roomID] {
		connections = append(connections, conn)
	}
	return connections
}

func (h *Hub) sendSnapshot(ctx context.Context, roomID string, conn *websocket.Conn) {
	if h.snapshot == nil {
		return
	}
	snapshot, err := h.snapshot.Snapshot(ctx, roomID)
	if err != nil {
		return
	}
	event := v1.AuctionEvent{
		Id:               idgen.New("evt"),
		Type:             v1.AuctionEventType_AUCTION_EVENT_TYPE_ROOM_SNAPSHOT,
		RoomId:           roomID,
		OccurredAtUnixMs: clock.NowMs(),
		Snapshot:         snapshot,
	}
	_ = conn.WriteJSON(event)
}

func (h *Hub) drain(conn *websocket.Conn) {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
