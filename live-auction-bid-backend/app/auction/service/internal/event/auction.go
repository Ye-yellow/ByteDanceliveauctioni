package event

import (
	"live-auction-bid/backend/app/auction/service/internal/model"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func NewAuctionEvent(typ model.EventType, lot *model.Lot) model.AuctionEvent {
	if lot == nil {
		return model.AuctionEvent{ID: idgen.New("evt"), Type: typ, OccurredAtUnixMs: clock.NowMs()}
	}
	return model.AuctionEvent{
		ID:               idgen.New("evt"),
		Type:             typ,
		RoomID:           lot.RoomID,
		LotID:            lot.ID,
		OccurredAtUnixMs: clock.NowMs(),
		Lot:              model.CloneLot(lot),
	}
}
