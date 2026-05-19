package biz

import "context"

// LotRepository 管理拍品聚合持久化。
// biz 只依赖接口，不关心内存、MySQL 或其他存储实现。
type LotRepository interface {
	Create(ctx context.Context, lot *Lot) error
	Save(ctx context.Context, lot *Lot) error
	FindByID(ctx context.Context, lotID string) (*Lot, error)
	List(ctx context.Context, roomID string, status LotStatus) ([]*Lot, error)
}

// BidRepository 管理出价流水和幂等记录。
// V1 是内存实现；后续可替换为 Redis Stream / MySQL / Redis 幂等键。
type BidRepository interface {
	Append(ctx context.Context, bid Bid) error
	ListByLot(ctx context.Context, lotID string) ([]Bid, error)
	FindByIdempotencyKey(ctx context.Context, lotID, key string) (Bid, bool, error)
	SaveIdempotencyKey(ctx context.Context, lotID, key string, bid Bid) error
}

// EventPublisher 发布领域事件。
// realtime 模块实现该接口，biz 不直接依赖 WebSocket。
type EventPublisher interface {
	Publish(ctx context.Context, event AuctionEvent)
}
