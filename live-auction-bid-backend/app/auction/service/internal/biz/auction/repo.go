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
	AttachAssets(ctx context.Context, ownerUserID string, lot *v1.Lot) error
	FindByID(ctx context.Context, lotID string) (*v1.Lot, error)
	List(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error)
	ListLots(ctx context.Context, query LotQuery) (LotList, error)
}

type ExpiredLotRepository interface {
	ListExpiredOpen(ctx context.Context, nowMs int64, limit int) ([]*v1.Lot, error)
}

// BidRepository 管理出价流水和幂等记录。
// 持久幂等键随 CommitAcceptedBid 进入 MySQL 事务；CacheIdempotencyKey 只预热 Redis 缓存，不能成为用户级成功/失败边界。
type BidRepository interface {
	CommitAcceptedBid(ctx context.Context, bid v1.Bid, lot *v1.Lot, expectedLotVersion int64, idempotencyKey string, order *Order, events []v1.AuctionEvent) error
	ListByLot(ctx context.Context, lotID string) ([]v1.Bid, error)
	ListBidRecordsByBuyer(ctx context.Context, buyerUserID string, query BidRecordQuery) (BidRecordList, error)
	FindByIdempotencyKey(ctx context.Context, lotID, userID, key string) (v1.Bid, bool, error)
	CacheIdempotencyKey(ctx context.Context, lotID, userID, key string, bid v1.Bid)
}

type OrderRepository interface {
	CreateOrderForSettledLot(ctx context.Context, order Order, lot *v1.Lot, expectedLotVersion int64, events []v1.AuctionEvent) error
	FindOrderByID(ctx context.Context, orderID string) (*Order, error)
	FindOrderByLot(ctx context.Context, lotID string) (*Order, bool, error)
	ListOrdersByBuyer(ctx context.Context, buyerUserID string) ([]Order, error)
	ListOrders(ctx context.Context, query OrderQuery) (OrderList, error)
}

type PaymentRepository interface {
	FindPaymentByIdempotencyKey(ctx context.Context, orderID, key string) (*Payment, bool, error)
	CommitPaymentSuccess(ctx context.Context, payment Payment, order Order, expectedOrderVersion int64, events []v1.AuctionEvent) error
}

// EventRepository 持久化不伴随聚合状态更新的领域事件。
// 伴随 lot/bid 状态变化的事件必须随对应 repository 方法进入同一个 MySQL 事务。
type EventRepository interface {
	PersistEvents(ctx context.Context, events []v1.AuctionEvent) error
	ListRoomEvents(ctx context.Context, query RoomEventQuery) (RoomEventList, error)
}

// EventPublisher 发布领域事件。
type EventPublisher interface {
	Publish(ctx context.Context, event v1.AuctionEvent) error
}
