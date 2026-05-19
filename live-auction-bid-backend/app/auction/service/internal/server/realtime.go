package server

import (
	"net/http"
	"strings"

	"live-auction-bid/backend/app/auction/service/internal/realtime"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

func registerRealtimeHTTP(srv *httptransport.Server, hub *realtime.Hub) {
	srv.HandlePrefix("/ws/rooms/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roomID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/ws/rooms/"), "/")
		if roomID == "" {
			roomID = "demo"
		}
		hub.ServeRoom(w, r, roomID)
	}))
}
