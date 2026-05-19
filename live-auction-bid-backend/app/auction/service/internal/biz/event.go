package biz

func NewAuctionEvent(typ EventType, lot *Lot) AuctionEvent {
	if lot == nil {
		return AuctionEvent{ID: NewID("evt"), Type: typ, OccurredAtUnixMs: NowMs()}
	}
	return AuctionEvent{
		ID:               NewID("evt"),
		Type:             typ,
		RoomID:           lot.RoomID,
		LotID:            lot.ID,
		OccurredAtUnixMs: NowMs(),
		Lot:              CloneLot(lot),
	}
}
