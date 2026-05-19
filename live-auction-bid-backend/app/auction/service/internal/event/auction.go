package event

import (
	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func NewAuctionEvent(typ v1.AuctionEventType, lot *v1.Lot) v1.AuctionEvent {
	if lot == nil {
		return v1.AuctionEvent{Id: idgen.New("evt"), Type: typ, OccurredAtUnixMs: clock.NowMs()}
	}
	return v1.AuctionEvent{
		Id:               idgen.New("evt"),
		Type:             typ,
		RoomId:           lot.RoomId,
		LotId:            lot.Id,
		OccurredAtUnixMs: clock.NowMs(),
		Lot:              cloneLot(lot),
	}
}

func cloneLot(lot *v1.Lot) *v1.Lot {
	if lot == nil {
		return nil
	}
	return proto.Clone(lot).(*v1.Lot)
}
