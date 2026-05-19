package server

import (
	"encoding/json"
	"net/http"
	"strings"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	auctionsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

type Server struct {
	app *auctionsvc.Service
	hub *realtime.Hub
}

func NewServer(addr string, app *auctionsvc.Service, hub *realtime.Hub) *khttp.Server {
	s := &Server{app: app, hub: hub}
	h := http.NewServeMux()
	s.registerRoutes(h)
	server := khttp.NewServer(khttp.Address(addr))
	server.HandlePrefix("/", h)
	return server
}

func (s *Server) registerRoutes(h *http.ServeMux) {
	h.HandleFunc("/healthz", s.health)
	h.HandleFunc("/api/lots", s.lots)
	h.HandleFunc("/api/lots/", s.lotAction)
	h.HandleFunc("/ws/rooms/", s.roomWS)
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]string{"service": "live-auction-bid", "status": "ok"}) })
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]string{"ok": "true"}) }

func (s *Server) lots(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var cmd auctionsvc.CreateLotCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		lot, err := s.app.CreateLot(r.Context(), cmd)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, lot)
		return
	}
	lots, _ := s.app.ListLots(r.Context())
	writeJSON(w, lots)
}

func (s *Server) lotAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/lots/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	lotID, action := parts[0], parts[1]
	switch action {
	case "bid":
		var req struct {
			UserID   string
			Nickname string
			Amount   biz.Money
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		lot, err := s.app.PlaceBid(r.Context(), lotID, req.UserID, req.Nickname, req.Amount)
		if err != nil {
			http.Error(w, err.Error(), 409)
			return
		}
		writeJSON(w, lot)
	case "settle":
		lot, err := s.app.Settle(r.Context(), lotID)
		if err != nil {
			http.Error(w, err.Error(), 409)
			return
		}
		writeJSON(w, lot)
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
