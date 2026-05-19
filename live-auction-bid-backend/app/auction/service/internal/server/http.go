package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"
)

type Server struct {
	app *appsvc.AuctionService
	hub *realtime.Hub
}

func New(app *appsvc.AuctionService, hub *realtime.Hub) *Server {
	return &Server{app: app, hub: hub}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/api/lots", s.handleLots)
	mux.HandleFunc("/api/lots/", s.handleLotAction)
	mux.HandleFunc("/api/rooms/", s.handleRoomAction)
	mux.HandleFunc("/ws/rooms/", s.handleRoomWS)
	mux.HandleFunc("/", s.index)
	return cors(mux)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"service": "auction-backend", "status": "ok"})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleLots(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		lots, err := s.app.ListLots(r.Context(), r.URL.Query().Get("roomId"), auction.LotStatus(r.URL.Query().Get("status")))
		writeResult(w, http.StatusOK, lots, err)
	case http.MethodPost:
		var cmd auction.CreateLotCommand
		if !decodeJSON(w, r, &cmd) {
			return
		}
		lot, err := s.app.CreateLot(r.Context(), cmd)
		writeResult(w, http.StatusCreated, map[string]any{"lot": lot}, err)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleLotAction(w http.ResponseWriter, r *http.Request) {
	lotID, action, rest, ok := parseLotPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		s.getLot(w, r, lotID)
	case r.Method == http.MethodPost && action == "start":
		s.startLot(w, r, lotID)
	case r.Method == http.MethodPost && action == "bid":
		s.placeBid(w, r, lotID)
	case r.Method == http.MethodPost && action == "trust-cards":
		s.revealTrustCard(w, r, lotID, rest)
	case r.Method == http.MethodPost && action == "duel":
		s.startDuel(w, r, lotID)
	case r.Method == http.MethodPost && action == "settle":
		s.settleLot(w, r, lotID)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) getLot(w http.ResponseWriter, r *http.Request, lotID string) {
	lot, err := s.app.GetLot(r.Context(), lotID)
	writeResult(w, http.StatusOK, map[string]any{"lot": lot}, err)
}

func (s *Server) startLot(w http.ResponseWriter, r *http.Request, lotID string) {
	lot, err := s.app.StartLot(r.Context(), lotID)
	writeResult(w, http.StatusOK, map[string]any{"lot": lot}, err)
}

func (s *Server) placeBid(w http.ResponseWriter, r *http.Request, lotID string) {
	var cmd auction.PlaceBidCommand
	if !decodeJSON(w, r, &cmd) {
		return
	}
	cmd.LotID = lotID

	lot, bid, ranking, err := s.app.PlaceBid(r.Context(), cmd)
	payload := map[string]any{
		"accepted":     err == nil,
		"lot":          lot,
		"bid":          bid,
		"ranking":      ranking,
		"rejectReason": errorText(err),
	}
	writeResult(w, http.StatusOK, payload, err)
}

func (s *Server) revealTrustCard(w http.ResponseWriter, r *http.Request, lotID string, rest []string) {
	if len(rest) != 2 || rest[1] != "reveal" {
		http.NotFound(w, r)
		return
	}
	lot, card, err := s.app.RevealTrustCard(r.Context(), lotID, rest[0], "")
	writeResult(w, http.StatusOK, map[string]any{"lot": lot, "trustCard": card}, err)
}

func (s *Server) startDuel(w http.ResponseWriter, r *http.Request, lotID string) {
	lot, duel, err := s.app.StartDuel(r.Context(), lotID, "", "", "")
	writeResult(w, http.StatusOK, map[string]any{"lot": lot, "duelState": duel}, err)
}

func (s *Server) settleLot(w http.ResponseWriter, r *http.Request, lotID string) {
	lot, err := s.app.SettleLot(r.Context(), lotID, "")
	writeResult(w, http.StatusOK, map[string]any{"lot": lot}, err)
}

func (s *Server) handleRoomAction(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/rooms/"))
	if len(parts) != 2 || parts[1] != "snapshot" {
		http.NotFound(w, r)
		return
	}
	snapshot, err := s.app.Snapshot(r.Context(), parts[0])
	writeResult(w, http.StatusOK, map[string]any{"snapshot": snapshot}, err)
}

func (s *Server) handleRoomWS(w http.ResponseWriter, r *http.Request) {
	roomID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/ws/rooms/"), "/")
	if roomID == "" {
		roomID = "demo"
	}
	s.hub.ServeRoom(w, r, roomID)
}

func parseLotPath(path string) (lotID string, action string, rest []string, ok bool) {
	parts := splitPath(strings.TrimPrefix(path, "/api/lots/"))
	if len(parts) == 0 || parts[0] == "" {
		return "", "", nil, false
	}
	lotID = parts[0]
	if len(parts) > 1 {
		action = parts[1]
		rest = parts[2:]
	}
	return lotID, action, rest, true
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeResult(w http.ResponseWriter, status int, payload any, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, status, payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}
