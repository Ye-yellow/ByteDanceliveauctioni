package auction

import (
	"context"

	"live-auction-bid/backend/app/auction/service/internal/model"
)

// LotRepository 管理拍品聚合持久化。
// biz 只依赖接口，不关心内存、MySQL 或其他存储实现。
type LotRepository interface {
	Create(ctx context.Context, lot *model.Lot) error
	Save(ctx context.Context, lot *model.Lot) error
	FindByID(ctx context.Context, lotID string) (*model.Lot, error)
	List(ctx context.Context, roomID string, status model.LotStatus) ([]*model.Lot, error)
}

// BidRepository 管理出价流水和幂等记录。
type BidRepository interface {
	Append(ctx context.Context, bid model.Bid) error
	ListByLot(ctx context.Context, lotID string) ([]model.Bid, error)
	FindByIdempotencyKey(ctx context.Context, lotID, key string) (model.Bid, bool, error)
	SaveIdempotencyKey(ctx context.Context, lotID, key string, bid model.Bid) error
}

// EventPublisher 发布领域事件。
type EventPublisher interface {
	Publish(ctx context.Context, event model.AuctionEvent)
}
