package biz

import "context"

type LotRepository interface {
	Save(ctx context.Context, lot *Lot) error
	FindByID(ctx context.Context, id string) (*Lot, error)
	FindLiveByRoom(ctx context.Context, roomID string) (*Lot, error)
	List(ctx context.Context) ([]*Lot, error)
}

type EventPublisher interface {
	PublishLotUpdated(ctx context.Context, lot *Lot) error
	PublishBidAccepted(ctx context.Context, lot *Lot, bid Bid) error
	PublishLotSettled(ctx context.Context, lot *Lot) error
}

type AtmosphereAI interface {
	OnBid(ctx context.Context, lot *Lot, bid Bid) string
	SuggestStartPrice(ctx context.Context, title, description string, referencePrice Money) Money
}
