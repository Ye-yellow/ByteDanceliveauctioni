package auction

import (
	"context"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

// LotRepository 管理拍品聚合持久化。
// biz 只依赖接口，不关心内存、MySQL 或其他存储实现。
type LotRepository interface {
	Create(ctx context.Context, lot *v1.Lot, ownerUserID string, events []v1.AuctionEvent) error
	Save(ctx context.Context, lot *v1.Lot, expectedVersion int64, events []v1.AuctionEvent) error
	FindByID(ctx context.Context, lotID string) (*v1.Lot, error)
	List(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error)
}

// BidRepository 管理出价流水和幂等记录。
// 持久幂等键随 CommitAcceptedBid 进入 MySQL 事务；CacheIdempotencyKey 只预热 Redis 缓存，不能成为用户级成功/失败边界。
type BidRepository interface {
	CommitAcceptedBid(ctx context.Context, bid v1.Bid, lot *v1.Lot, expectedLotVersion int64, idempotencyKey string, events []v1.AuctionEvent) error
	ListByLot(ctx context.Context, lotID string) ([]v1.Bid, error)
	FindByIdempotencyKey(ctx context.Context, lotID, key string) (v1.Bid, bool, error)
	CacheIdempotencyKey(ctx context.Context, lotID, key string, bid v1.Bid)
}

// EventRepository 持久化不伴随聚合状态更新的领域事件。
// 伴随 lot/bid 状态变化的事件必须随对应 repository 方法进入同一个 MySQL 事务。
type EventRepository interface {
	PersistEvents(ctx context.Context, events []v1.AuctionEvent) error
}

// EventPublisher 发布领域事件。
type EventPublisher interface {
	Publish(ctx context.Context, event v1.AuctionEvent) error
}
