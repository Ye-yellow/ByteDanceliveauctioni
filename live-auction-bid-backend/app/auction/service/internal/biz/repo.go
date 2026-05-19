package biz

import "context"

type LotRepository interface {
	Create(ctx context.Context, lot *Lot) error
	Save(ctx context.Context, lot *Lot) error
	FindByID(ctx context.Context, lotID string) (*Lot, error)
	List(ctx context.Context, roomID string, status LotStatus) ([]*Lot, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event AuctionEvent)
}
