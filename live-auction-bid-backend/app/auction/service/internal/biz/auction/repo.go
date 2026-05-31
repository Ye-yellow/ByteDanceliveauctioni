package auction

import (
	"context"
	"errors"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

// LotRepository 管理拍品聚合持久化。
// biz 只依赖接口，不关心内存、MySQL 或其他存储实现。
type LotRepository interface {
	Create(ctx context.Context, lot *v1.Lot, ownerUserID string, events []v1.AuctionEvent) error
	Save(ctx context.Context, lot *v1.Lot, expectedVersion int64, events []v1.AuctionEvent) error
	QueueLotAsNext(ctx context.Context, lotID, mainAccountID, ownerUserID string, nowMs int64) (*v1.Lot, int32, []v1.AuctionEvent, error)
	StartLotAsOnlyActive(ctx context.Context, lot *v1.Lot, expectedVersion int64, events []v1.AuctionEvent) error
	AttachAssets(ctx context.Context, ownerUserID string, lot *v1.Lot) error
	FindByID(ctx context.Context, lotID string) (*v1.Lot, error)
	FindCoreByID(ctx context.Context, lotID string) (*v1.Lot, error)
	List(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error)
	ListLots(ctx context.Context, query LotQuery) (LotList, error)
	FindOrCreateRoomState(ctx context.Context, roomID, mainAccountID string, nowMs int64) (*RoomState, error)
	RepairRoomActiveLot(ctx context.Context, roomID, activeLotID string, nowMs int64) error
}

type RoomRepository interface {
	EnsureDefaultRoom(ctx context.Context, mainAccountID, createdByUserID string, nowMs int64) (*Room, error)
	ListRooms(ctx context.Context, query RoomQuery) ([]Room, error)
	FindRoomByID(ctx context.Context, roomID string) (*Room, bool, error)
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

type RuntimeBidResult struct {
	Lot                *v1.Lot
	Bid                *v1.Bid
	Ranking            []*v1.RankingItem
	RecentBids         []*v1.Bid
	PreviousLeaderID   string
	EndsBeforeBid      int64
	ExtendCountBefore  int32
	RuntimeEventID     string
	RuntimeStreamID    string
	PreviousLotVersion int64
	LotVersion         int64
	OrderID            string
	Replayed           bool
}

type RuntimeBidRejectError struct {
	Code               string
	CurrentAmount      int64
	CurrentCurrency    string
	MinIncrementAmount int64
	NextBidAmount      int64
	LeadingUserID      string
	LeadingNickname    string
	LotVersion         int64
	EndsAtUnixMs       int64
	Cause              error
}

func (e *RuntimeBidRejectError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return e.Code
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return string(apperr.CodeBidRejected)
}

func (e *RuntimeBidRejectError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func RuntimeBidRejectFromError(err error) (*RuntimeBidRejectError, bool) {
	var reject *RuntimeBidRejectError
	if errors.As(err, &reject) {
		return reject, true
	}
	return nil, false
}

func (e *RuntimeBidRejectError) Lot(lotID string, fallbackAmount *v1.Money) *v1.Lot {
	if e == nil {
		return &v1.Lot{Id: lotID}
	}
	currency := e.CurrentCurrency
	if currency == "" && fallbackAmount != nil {
		currency = fallbackAmount.GetCurrency()
	}
	lot := &v1.Lot{
		Id:              lotID,
		CurrentPrice:    &v1.Money{Amount: e.CurrentAmount, Currency: currency},
		LeadingUserId:   e.LeadingUserID,
		LeadingNickname: e.LeadingNickname,
		EndsAtUnixMs:    e.EndsAtUnixMs,
		Version:         e.LotVersion,
		Rule: &v1.BidRule{
			MinIncrement: &v1.Money{Amount: e.MinIncrementAmount, Currency: currency},
		},
	}
	switch apperr.BusinessCode(e.Code) {
	case apperr.CodeLotCancelled:
		lot.Status = v1.LotStatus_LOT_STATUS_CANCELLED
	case apperr.CodeBidEnded:
		lot.Status = v1.LotStatus_LOT_STATUS_SETTLED
	}
	return lot
}

type RuntimeProjectionEvent struct {
	RuntimeEventID     string
	RuntimeStreamID    string
	RoomID             string
	LotID              string
	EventType          string
	IdempotencyKey     string
	Bid                v1.Bid
	Lot                *v1.Lot
	Ranking            []*v1.RankingItem
	PreviousLeaderID   string
	EndsBeforeBid      int64
	ExtendCountBefore  int32
	PreviousLotVersion int64
	LotVersion         int64
	OccurredAtUnixMs   int64
	OrderID            string
}

type RuntimeProjectionOutcome struct {
	Projected        bool
	AlreadyProjected bool
	Gap              bool
	Conflict         bool
}

type AuctionRuntime interface {
	HydrateLotRuntime(ctx context.Context, lot *v1.Lot) error
	SyncLotRuntime(ctx context.Context, lot *v1.Lot) error
	CancelLotRuntime(ctx context.Context, lot *v1.Lot, reason, operatorID string, nowMs int64) (*v1.Lot, []*v1.RankingItem, error)
	PlaceBidRuntime(ctx context.Context, lot *v1.Lot, req *v1.PlaceBidRequest, bidderID, nickname, bidID string, nowMs int64) (RuntimeBidResult, error)
	SnapshotRuntime(ctx context.Context, current *v1.Lot) (*v1.RoomSnapshot, error)
	RankingRuntime(ctx context.Context, lotID string, limit int64) ([]*v1.RankingItem, error)
}

type RuntimeProjectionRepository interface {
	ProjectRuntimeBid(ctx context.Context, bid v1.Bid, lot *v1.Lot, idempotencyKey string, order *Order, events []v1.AuctionEvent) error
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
