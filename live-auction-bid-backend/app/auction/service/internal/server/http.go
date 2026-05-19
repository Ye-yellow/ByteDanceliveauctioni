package server

import (
	"encoding/json"
	"net/http"
	"strings"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
	appsvc "live-auction-bid/backend/app/auction/service/internal/service"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(addr string, auction *appsvc.AuctionService, hub *realtime.Hub) *httptransport.Server {
	srv := httptransport.NewServer(httptransport.Address(addr))

	// Kratos generated HTTP routes from api/auction/service/v1/auction.proto.
	v1.RegisterAuctionServiceHTTPServer(srv, auction)

	// Non-proto realtime endpoint: WebSocket room broadcasting.
	srv.HandlePrefix("/ws/rooms/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roomID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/ws/rooms/"), "/")
		if roomID == "" {
			roomID = "demo"
		}
		hub.ServeRoom(w, r, roomID)
	}))

	// Health and root are operational endpoints, not business RPC.
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	srv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"service": "auction-backend", "transport": "kratos-http", "status": "ok"})
	})

	return srv
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
