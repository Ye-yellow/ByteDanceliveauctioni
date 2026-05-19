package realtime

import (
	"context"
	"encoding/json"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
)

type Broadcaster struct {
	hub *Hub
}

func NewBroadcaster(hub *Hub) *Broadcaster { return &Broadcaster{hub: hub} }

func (b *Broadcaster) PublishLotUpdated(ctx context.Context, lot *biz.Lot) error {
	b.hub.Broadcast(lot.RoomID, Envelope{Type: MessageLotUpdated, Data: lotSnapshot(lot)})
	return nil
}

func (b *Broadcaster) PublishBidAccepted(ctx context.Context, lot *biz.Lot, bid biz.Bid) error {
	b.hub.Broadcast(lot.RoomID, Envelope{Type: MessageBidAccepted, Data: map[string]interface{}{"bid": bid, "lot": lotSnapshot(lot)}})
	return nil
}

func (b *Broadcaster) PublishLotSettled(ctx context.Context, lot *biz.Lot) error {
	b.hub.Broadcast(lot.RoomID, Envelope{Type: MessageLotSettled, Data: lotSnapshot(lot)})
	return nil
}

func lotSnapshot(lot *biz.Lot) map[string]interface{} {
	b, _ := json.Marshal(lot)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	m["ranking"] = lot.Ranking(10)
	return m
}
