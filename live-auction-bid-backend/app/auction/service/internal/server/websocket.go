package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
	"live-auction-bid/backend/app/auction/service/internal/realtime"
)

type socketClient struct{ conn *websocket.Conn }

func (c socketClient) SendJSON(v interface{}) error { return c.conn.WriteJSON(v) }

func (s *Server) roomWS(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/rooms/")
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := socketClient{conn: conn}
	leave := s.hub.Join(roomID, client)
	defer leave()
	defer conn.Close()
	if lot, err := s.app.LiveLot(context.Background(), roomID); err == nil {
		_ = client.SendJSON(realtime.Envelope{Type: realtime.MessageLotUpdated, Data: lot})
	}
	for {
		var msg struct {
			Type     string    `json:"type"`
			LotID    string    `json:"lotId"`
			UserID   string    `json:"userId"`
			Nickname string    `json:"nickname"`
			Amount   biz.Money `json:"amount"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		if msg.Type == "bid.place" {
			_, _ = s.app.PlaceBid(r.Context(), msg.LotID, msg.UserID, msg.Nickname, msg.Amount)
		}
	}
}
