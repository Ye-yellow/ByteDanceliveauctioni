package auction

import (
	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func newAuctionEvent(typ v1.AuctionEventType, lot *v1.Lot) v1.AuctionEvent {
	if lot == nil {
		return v1.AuctionEvent{Id: idgen.New("evt"), Type: typ, OccurredAtUnixMs: clock.NowMs()}
	}
	return v1.AuctionEvent{
		Id:               idgen.New("evt"),
		Type:             typ,
		RoomId:           lot.RoomId,
		LotId:            lot.Id,
		MainAccountId:    lot.GetMainAccountId(),
		OccurredAtUnixMs: clock.NowMs(),
		Lot:              proto.Clone(lot).(*v1.Lot),
	}
}

func NewAuctionEvent(typ v1.AuctionEventType, lot *v1.Lot) v1.AuctionEvent {
	return newAuctionEvent(typ, lot)
}

func newAuctionEventWithID(id string, typ v1.AuctionEventType, lot *v1.Lot, occurredAtUnixMs int64) v1.AuctionEvent {
	if occurredAtUnixMs <= 0 {
		occurredAtUnixMs = clock.NowMs()
	}
	if id == "" {
		id = idgen.New("evt")
	}
	if lot == nil {
		return v1.AuctionEvent{Id: id, Type: typ, OccurredAtUnixMs: occurredAtUnixMs}
	}
	return v1.AuctionEvent{
		Id:               id,
		Type:             typ,
		RoomId:           lot.RoomId,
		LotId:            lot.Id,
		MainAccountId:    lot.GetMainAccountId(),
		OccurredAtUnixMs: occurredAtUnixMs,
		Lot:              proto.Clone(lot).(*v1.Lot),
	}
}
