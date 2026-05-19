package auction

import (
	"context"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

// LotRepository 管理拍品聚合持久化。
// biz 只依赖接口，不关心内存、MySQL 或其他存储实现。
type LotRepository interface {
	Create(ctx context.Context, lot *v1.Lot) error
	Save(ctx context.Context, lot *v1.Lot) error
	FindByID(ctx context.Context, lotID string) (*v1.Lot, error)
	List(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error)
}

// BidRepository 管理出价流水和幂等记录。
type BidRepository interface {
	Append(ctx context.Context, bid v1.Bid) error
	ListByLot(ctx context.Context, lotID string) ([]v1.Bid, error)
	FindByIdempotencyKey(ctx context.Context, lotID, key string) (v1.Bid, bool, error)
	SaveIdempotencyKey(ctx context.Context, lotID, key string, bid v1.Bid) error
}

// EventPublisher 发布领域事件。
type EventPublisher interface {
	Publish(ctx context.Context, event v1.AuctionEvent)
}
