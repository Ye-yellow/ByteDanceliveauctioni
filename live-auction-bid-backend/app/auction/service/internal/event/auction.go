package event

import (
	"live-auction-bid/backend/app/auction/service/internal/model"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func NewAuctionEvent(typ model.EventType, lot *model.Lot) model.AuctionEvent {
	if lot == nil {
		return model.AuctionEvent{Id: idgen.New("evt"), Type: typ, OccurredAtUnixMs: clock.NowMs()}
	}
	return model.AuctionEvent{
		Id:               idgen.New("evt"),
		Type:             typ,
		RoomId:           lot.RoomId,
		LotId:            lot.Id,
		OccurredAtUnixMs: clock.NowMs(),
		Lot:              model.CloneLot(lot),
	}
}
