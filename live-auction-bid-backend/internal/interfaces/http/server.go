package httpiface

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/websocket"

	auctionapp "live-auction-bid/backend/internal/application/auction"
	domain "live-auction-bid/backend/internal/domain/auction"
	"live-auction-bid/backend/internal/interfaces/ws"
)

type Server struct {
	app *auctionapp.Service
	hub *ws.Hub
}

func NewServer(addr string, app *auctionapp.Service, hub *ws.Hub) *khttp.Server {
	s := &Server{app: app, hub: hub}
	h := http.NewServeMux()
	h.HandleFunc("/healthz", s.health)
	h.HandleFunc("/api/lots", s.lots)
	h.HandleFunc("/api/lots/", s.lotAction)
	h.HandleFunc("/ws/rooms/", s.roomWS)
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]string{"service":"live-auction-bid","status":"ok"}) })
	server := khttp.NewServer(khttp.Address(addr))
	server.HandlePrefix("/", h)
	return server
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]string{"ok":"true"}) }

func (s *Server) lots(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var cmd auctionapp.CreateLotCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil { http.Error(w, err.Error(), 400); return }
		lot, err := s.app.CreateLot(r.Context(), cmd)
		if err != nil { http.Error(w, err.Error(), 500); return }
		writeJSON(w, lot); return
	}
	lots, _ := s.app.ListLots(r.Context())
	writeJSON(w, lots)
}

func (s *Server) lotAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/lots/"), "/")
	if len(parts) < 2 { http.NotFound(w, r); return }
	lotID, action := parts[0], parts[1]
	switch action {
	case "bid":
		var req struct { UserID, Nickname string; Amount domain.Money }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, err.Error(), 400); return }
		lot, err := s.app.PlaceBid(r.Context(), lotID, req.UserID, req.Nickname, req.Amount)
		if err != nil { http.Error(w, err.Error(), 409); return }
		writeJSON(w, lot)
	case "settle":
		lot, err := s.app.Settle(r.Context(), lotID)
		if err != nil { http.Error(w, err.Error(), 409); return }
		writeJSON(w, lot)
	default:
		http.NotFound(w, r)
	}
}

type socketClient struct{ conn *websocket.Conn }
func (c socketClient) SendJSON(v interface{}) error { return c.conn.WriteJSON(v) }

func (s *Server) roomWS(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/rooms/")
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { return }
	client := socketClient{conn: conn}
	leave := s.hub.Join(roomID, client)
	defer leave(); defer conn.Close()
	if lot, err := s.app.LiveLot(context.Background(), roomID); err == nil {
		_ = client.SendJSON(ws.Envelope{Type: "lot.updated", Data: lot})
	}
	for {
		var msg struct { Type string `json:"type"`; LotID string `json:"lotId"`; UserID string `json:"userId"`; Nickname string `json:"nickname"`; Amount domain.Money `json:"amount"` }
		if err := conn.ReadJSON(&msg); err != nil { return }
		if msg.Type == "bid.place" { _, _ = s.app.PlaceBid(r.Context(), msg.LotID, msg.UserID, msg.Nickname, msg.Amount) }
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) { w.Header().Set("Content-Type", "application/json"); _ = json.NewEncoder(w).Encode(v) }
