package auction

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

// AuctionUsecase 编排直播竞拍业务流程。
//
// 分层约束：
// - 业务流程和规则放在 biz；
// - 存储细节通过 Repository 接口隔离；
// - 实时广播通过 EventPublisher 接口隔离；
// - HTTP/WS 适配不放在 biz 层。
type AuctionUsecase struct {
	mu          sync.Mutex
	lots        LotRepository
	expiredLots ExpiredLotRepository
	bids        BidRepository
	orders      OrderRepository
	payments    PaymentRepository
	eventsStore EventRepository
	events      EventPublisher
}

type CloseExpiredSummary struct {
	Scanned   int
	Closed    int
	Settled   int
	Failed    int
	Conflicts int
}

const (
	orderCreatedPublicReason   = "order_created"
	paymentSuccessPublicReason = "payment_success"
)

func NewAuctionUsecase(lots LotRepository, bids BidRepository, eventStore EventRepository, events EventPublisher) *AuctionUsecase {
	uc := &AuctionUsecase{lots: lots, bids: bids, eventsStore: eventStore, events: events}
	if repo, ok := lots.(ExpiredLotRepository); ok {
		uc.expiredLots = repo
	}
	if repo, ok := lots.(OrderRepository); ok {
		uc.orders = repo
	}
	if repo, ok := lots.(PaymentRepository); ok {
		uc.payments = repo
	}
	return uc
}

func (uc *AuctionUsecase) CloseExpiredLots(ctx context.Context, nowMs int64, limit int) (CloseExpiredSummary, error) {
	if nowMs <= 0 {
		return CloseExpiredSummary{}, fmt.Errorf("%w: now ms is required", apperr.ErrInvalidArgument)
	}
	if uc.expiredLots == nil {
		return CloseExpiredSummary{}, errors.New("expired lot repository is required")
	}
	lots, err := uc.expiredLots.ListExpiredOpen(ctx, nowMs, limit)
	if err != nil {
		return CloseExpiredSummary{}, err
	}
	summary := CloseExpiredSummary{Scanned: len(lots)}
	for _, lot := range lots {
		if lot == nil || lot.Id == "" {
			continue
		}
		closed, settled, err := uc.closeExpiredLot(ctx, lot.Id, nowMs)
		if err != nil {
			if apperr.IsLotVersionConflict(err) {
				summary.Conflicts++
				continue
			}
			return summary, err
		}
		if !closed {
			continue
		}
		if settled {
			summary.Settled++
		} else {
			summary.Failed++
		}
		summary.Closed++
	}
	return summary, nil
}

func (uc *AuctionUsecase) CreateLot(ctx context.Context, req *v1.CreateLotRequest, ownerUserID string) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := NewLotFromRequest(idgen.New("lot"), req)
	if err != nil {
		return nil, err
	}
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CREATED, lot)
	if err := uc.lots.Create(ctx, lot, ownerUserID, []v1.AuctionEvent{event}); err != nil {
		return nil, err
	}
	if err := uc.broadcast(ctx, event); err != nil {
		return nil, err
	}
	return proto.Clone(lot).(*v1.Lot), nil
}

func (uc *AuctionUsecase) CreateLotDraft(ctx context.Context, req *v1.CreateLotRequest, ownerUserID string) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := NewLotDraftFromRequest(idgen.New("lot"), req, false)
	if err != nil {
		return nil, err
	}
	if err := uc.lots.Create(ctx, lot, ownerUserID, nil); err != nil {
		return nil, err
	}
	return proto.Clone(lot).(*v1.Lot), nil
}

func (uc *AuctionUsecase) PatchLotDraft(ctx context.Context, req *v1.PatchLotDraftRequest, ownerUserID string) (*v1.Lot, error) {
	_ = ownerUserID
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if req == nil {
		return nil, errors.New("patch lot draft request is required")
	}
	if req.GetLotId() == "" {
		return nil, errors.New("lot id is required")
	}
	lot, err := uc.lots.FindByID(ctx, req.GetLotId())
	if err != nil {
		return nil, err
	}
	expectedVersion := lot.Version
	if err := ApplyDraftPatch(lot, req); err != nil {
		return nil, err
	}
	if err := uc.lots.Save(ctx, lot, expectedVersion, nil); err != nil {
		return nil, err
	}
	if err := uc.lots.AttachAssets(ctx, ownerUserID, lot); err != nil {
		return nil, err
	}
	return proto.Clone(lot).(*v1.Lot), nil
}

func (uc *AuctionUsecase) QueueLot(ctx context.Context, lotID, ownerUserID string) (*v1.Lot, int32, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if lotID == "" {
		return nil, 0, errors.New("lot id is required")
	}
	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, 0, err
	}
	if lot.GetRoomId() == "" {
		return nil, 0, errors.New("room id is required")
	}
	expectedVersion := lot.Version
	queuePosition := lot.GetQueuePosition()
	if lot.GetQueueStatus() != v1.LotQueueStatus_LOT_QUEUE_STATUS_QUEUED || queuePosition <= 0 {
		queuePosition, err = uc.nextQueuePosition(ctx, lot.GetRoomId())
		if err != nil {
			return nil, 0, err
		}
	}
	if err := QueueLot(lot, queuePosition); err != nil {
		return nil, 0, err
	}
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_QUEUED, lot)
	if err := uc.lots.Save(ctx, lot, expectedVersion, []v1.AuctionEvent{event}); err != nil {
		return nil, 0, err
	}
	if err := uc.lots.AttachAssets(ctx, ownerUserID, lot); err != nil {
		return nil, 0, err
	}
	if err := uc.broadcast(ctx, event); err != nil {
		return nil, 0, err
	}
	return proto.Clone(lot).(*v1.Lot), lot.GetQueuePosition(), nil
}

func (uc *AuctionUsecase) nextQueuePosition(ctx context.Context, roomID string) (int32, error) {
	lots, err := uc.lots.List(ctx, roomID, 0)
	if err != nil {
		return 0, err
	}
	maxPosition := int32(0)
	for _, lot := range lots {
		if lot.GetQueueStatus() == v1.LotQueueStatus_LOT_QUEUE_STATUS_QUEUED || lot.GetQueueStatus() == v1.LotQueueStatus_LOT_QUEUE_STATUS_NEXT {
			if lot.GetQueuePosition() > maxPosition {
				maxPosition = lot.GetQueuePosition()
			}
		}
	}
	return maxPosition + 1, nil
}

func (uc *AuctionUsecase) GetLot(ctx context.Context, lotID string) (*v1.Lot, error) {
	if lotID == "" {
		return nil, fmt.Errorf("%w: lot id is required", apperr.ErrInvalidArgument)
	}
	return uc.lots.FindByID(ctx, lotID)
}

func (uc *AuctionUsecase) ListLots(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error) {
	if roomID == "" {
		return nil, errors.New("room id is required")
	}
	return uc.lots.List(ctx, roomID, status)
}

func (uc *AuctionUsecase) ListLotsByQuery(ctx context.Context, query LotQuery) (LotList, error) {
	query.Page, query.PageSize = NormalizePagination(query.Page, query.PageSize)
	query.View = strings.ToLower(strings.TrimSpace(query.View))
	switch query.View {
	case "", "all", "current", "history", "library":
	default:
		return LotList{}, fmt.Errorf("%w: unsupported lot list view: %s", apperr.ErrInvalidArgument, query.View)
	}
	return uc.lots.ListLots(ctx, query)
}

func (uc *AuctionUsecase) StartLot(ctx context.Context, lotID string) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if lotID == "" {
		return nil, errors.New("lot id is required")
	}

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, err
	}
	expectedVersion := lot.Version
	if err := StartLot(lot, clock.NowMs()); err != nil {
		return nil, err
	}
	bids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return nil, err
	}
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_STARTED, lot)
	event.Ranking = BuildRanking(bids)
	if err := uc.lots.Save(ctx, lot, expectedVersion, []v1.AuctionEvent{event}); err != nil {
		return nil, err
	}
	if err := uc.broadcast(ctx, event); err != nil {
		return nil, err
	}
	return proto.Clone(lot).(*v1.Lot), nil
}

func (uc *AuctionUsecase) PlaceBid(ctx context.Context, req *v1.PlaceBidRequest, bidderID, nickname string) (*v1.Lot, *v1.Bid, []*v1.RankingItem, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if req == nil {
		return nil, nil, nil, errors.New("place bid request is required")
	}
	if req.GetLotId() == "" {
		return nil, nil, nil, errors.New("lot id is required")
	}
	if bidderID == "" {
		return nil, nil, nil, errors.New("user id is required")
	}
	if nickname == "" {
		return nil, nil, nil, errors.New("nickname is required")
	}
	if req.GetAmount() == nil || req.GetAmount().GetCurrency() == "" {
		return nil, nil, nil, errors.New("bid amount and currency are required")
	}
	if req.GetIdempotencyKey() == "" {
		return nil, nil, nil, fmt.Errorf("%w: bid idempotency key is required", apperr.ErrInvalidArgument)
	}

	lot, err := uc.lots.FindByID(ctx, req.GetLotId())
	if err != nil {
		return nil, nil, nil, err
	}
	expectedVersion := lot.Version
	previousLeaderID := lot.LeadingUserId

	replayLot, replayBid, replayRanking, found, err := uc.replayBidByIdempotencyKey(ctx, lot.Id, bidderID, req.GetIdempotencyKey())
	if err != nil {
		return nil, nil, nil, err
	}
	if found {
		return replayLot, replayBid, replayRanking, nil
	}

	bid := v1.Bid{
		Id:              idgen.New("bid"),
		LotId:           lot.Id,
		UserId:          bidderID,
		Nickname:        nickname,
		Amount:          req.GetAmount(),
		CreatedAtUnixMs: clock.NowMs(),
	}
	endsBeforeBid := lot.EndsAtUnixMs
	extendCountBeforeBid := int32(0)
	if lot.GetDuelState() != nil {
		extendCountBeforeBid = lot.GetDuelState().GetExtendCount()
	}
	if err := AcceptBid(lot, bid, clock.NowMs()); err != nil {
		bids, listErr := uc.bids.ListByLot(ctx, lot.Id)
		if listErr != nil {
			return nil, nil, nil, listErr
		}
		ranking := BuildRanking(bids)
		event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_REJECTED, lot)
		event.Reason = err.Error()
		event.Ranking = ranking
		if persistErr := uc.persistEvents(ctx, event); persistErr != nil {
			return nil, nil, nil, persistErr
		}
		if publishErr := uc.broadcast(ctx, event); publishErr != nil {
			return nil, nil, nil, publishErr
		}
		return proto.Clone(lot).(*v1.Lot), nil, ranking, err
	}

	bids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return nil, nil, nil, err
	}
	bids = append(bids, bid)
	ranking := BuildRanking(bids)
	nowMs := clock.NowMs()
	if IsAuctionOpenStatus(lot.Status) && !lot.GetDuelState().GetActive() && len(ranking) >= 2 && len(bids) >= 3 &&
		lot.EndsAtUnixMs-nowMs <= 60_000 &&
		ranking[0].GetAmount().GetAmount()-ranking[1].GetAmount().GetAmount() <= lot.GetRule().GetMinIncrement().GetAmount()*3 {
		_ = StartDuel(lot, ranking, nowMs, "", "")
	}

	acceptedEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED, lot)
	acceptedEvent.Bid = &bid
	acceptedEvent.Ranking = ranking
	rankingEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_RANKING_UPDATED, lot)
	rankingEvent.Ranking = ranking
	commitEvents := []v1.AuctionEvent{acceptedEvent, rankingEvent}
	if previousLeaderID != "" && previousLeaderID != bidderID {
		outbidEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_OUTBID, lot)
		outbidEvent.Bid = &bid
		outbidEvent.Ranking = ranking
		outbidEvent.Reason = previousLeaderID
		commitEvents = append(commitEvents, outbidEvent)
	}
	if lot.EndsAtUnixMs != endsBeforeBid || lot.GetDuelState().GetExtendCount() != extendCountBeforeBid {
		updatedEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_UPDATED, lot)
		updatedEvent.Bid = &bid
		updatedEvent.Ranking = ranking
		updatedEvent.DuelState = lot.DuelState
		commitEvents = append(commitEvents, updatedEvent)
		extendedEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_EXTENDED, lot)
		extendedEvent.Bid = &bid
		extendedEvent.Ranking = ranking
		extendedEvent.DuelState = lot.DuelState
		commitEvents = append(commitEvents, extendedEvent)
	}
	if lot.GetDuelState().GetActive() {
		duelEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED, lot)
		duelEvent.Ranking = ranking
		duelEvent.DuelState = lot.DuelState
		commitEvents = append(commitEvents, duelEvent)
	}
	var order *Order
	if AuctionStateOf(lot) == AuctionStateSold {
		createdOrder, err := NewOrderFromSettledLot(idgen.New("order"), lot, nowMs)
		if err != nil {
			return nil, nil, nil, err
		}
		order = createdOrder
		settledEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED, lot)
		settledEvent.Bid = &bid
		settledEvent.Ranking = ranking
		closedEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED, lot)
		closedEvent.Bid = &bid
		closedEvent.Ranking = ranking
		orderEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED, lot)
		orderEvent.Ranking = ranking
		orderEvent.Reason = orderCreatedPublicReason
		commitEvents = append(commitEvents, settledEvent, closedEvent, orderEvent)
	}
	if err := uc.bids.CommitAcceptedBid(ctx, bid, lot, expectedVersion, req.GetIdempotencyKey(), order, commitEvents); err != nil {
		replayLot, replayBid, replayRanking, found, replayErr := uc.replayBidByIdempotencyKey(ctx, lot.Id, bidderID, req.GetIdempotencyKey())
		if replayErr != nil {
			return nil, nil, nil, replayErr
		}
		if found {
			return replayLot, replayBid, replayRanking, nil
		}
		return nil, nil, nil, err
	}
	uc.bids.CacheIdempotencyKey(ctx, lot.Id, bidderID, req.GetIdempotencyKey(), bid)

	committedBids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return nil, nil, nil, err
	}
	ranking = BuildRanking(committedBids)
	if err := uc.broadcast(ctx, commitEvents...); err != nil {
		return nil, nil, nil, err
	}
	return proto.Clone(lot).(*v1.Lot), &bid, ranking, nil
}

func (uc *AuctionUsecase) replayBidByIdempotencyKey(ctx context.Context, lotID, userID, key string) (*v1.Lot, *v1.Bid, []*v1.RankingItem, bool, error) {
	old, found, err := uc.bids.FindByIdempotencyKey(ctx, lotID, userID, key)
	if err != nil || !found {
		return nil, nil, nil, false, err
	}
	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, nil, nil, false, err
	}
	bids, err := uc.bids.ListByLot(ctx, lotID)
	if err != nil {
		return nil, nil, nil, false, err
	}
	return proto.Clone(lot).(*v1.Lot), &old, BuildRanking(bids), true, nil
}

func (uc *AuctionUsecase) RevealTrustCard(ctx context.Context, lotID, cardID, operatorID string) (*v1.Lot, *v1.TrustRevealCard, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if lotID == "" {
		return nil, nil, errors.New("lot id is required")
	}
	if cardID == "" {
		return nil, nil, errors.New("trust card id is required")
	}

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, nil, err
	}
	expectedVersion := lot.Version
	card, err := RevealTrustCard(lot, cardID, clock.NowMs())
	if err != nil {
		return nil, nil, err
	}
	bids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return nil, nil, err
	}
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_TRUST_REVEALED, lot)
	event.TrustCard = card
	event.Ranking = BuildRanking(bids)
	if err := uc.lots.Save(ctx, lot, expectedVersion, []v1.AuctionEvent{event}); err != nil {
		return nil, nil, err
	}
	if err := uc.broadcast(ctx, event); err != nil {
		return nil, nil, err
	}
	return proto.Clone(lot).(*v1.Lot), card, nil
}

func (uc *AuctionUsecase) StartDuel(ctx context.Context, lotID, operatorID, userAID, userBID string) (*v1.Lot, *v1.DuelState, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if lotID == "" {
		return nil, nil, errors.New("lot id is required")
	}

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, nil, err
	}
	expectedVersion := lot.Version
	bids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return nil, nil, err
	}
	ranking := BuildRanking(bids)
	if err := StartDuel(lot, ranking, clock.NowMs(), userAID, userBID); err != nil {
		return nil, nil, err
	}
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED, lot)
	event.Ranking = ranking
	event.DuelState = lot.DuelState
	if err := uc.lots.Save(ctx, lot, expectedVersion, []v1.AuctionEvent{event}); err != nil {
		return nil, nil, err
	}
	if err := uc.broadcast(ctx, event); err != nil {
		return nil, nil, err
	}
	return proto.Clone(lot).(*v1.Lot), lot.DuelState, nil
}

func (uc *AuctionUsecase) SettleLot(ctx context.Context, lotID, operatorID string) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if lotID == "" {
		return nil, errors.New("lot id is required")
	}

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, err
	}
	expectedVersion := lot.Version
	if err := SettleLot(lot, clock.NowMs()); err != nil {
		return nil, err
	}
	bids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return nil, err
	}
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED, lot)
	event.Ranking = BuildRanking(bids)
	order, err := NewOrderFromSettledLot(idgen.New("order"), lot, clock.NowMs())
	if err != nil {
		return nil, err
	}
	if uc.orders == nil {
		return nil, errors.New("order repository is required")
	}
	closedEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED, lot)
	closedEvent.Ranking = event.Ranking
	orderEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED, lot)
	orderEvent.Ranking = event.Ranking
	orderEvent.Reason = orderCreatedPublicReason
	events := []v1.AuctionEvent{event, closedEvent, orderEvent}
	if err := uc.orders.CreateOrderForSettledLot(ctx, *order, lot, expectedVersion, events); err != nil {
		return nil, err
	}
	if err := uc.broadcast(ctx, events...); err != nil {
		return nil, err
	}
	return proto.Clone(lot).(*v1.Lot), nil
}

func (uc *AuctionUsecase) closeExpiredLot(ctx context.Context, lotID string, nowMs int64) (bool, bool, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return false, false, err
	}
	if !IsAuctionOpenStatus(lot.Status) || lot.EndsAtUnixMs == 0 || lot.EndsAtUnixMs > nowMs {
		return false, false, nil
	}
	expectedVersion := lot.Version
	bids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return false, false, err
	}
	ranking := BuildRanking(bids)
	if lot.LeadingUserId == "" {
		reason := "auction expired without accepted bid"
		if err := FailExpiredLot(lot, reason, nowMs); err != nil {
			return false, false, err
		}
		closedEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED, lot)
		closedEvent.Ranking = ranking
		closedEvent.Reason = reason
		failedEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CANCELLED, lot)
		failedEvent.Ranking = ranking
		failedEvent.Reason = reason
		events := []v1.AuctionEvent{closedEvent, failedEvent}
		if err := uc.lots.Save(ctx, lot, expectedVersion, events); err != nil {
			return false, false, err
		}
		if err := uc.broadcast(ctx, events...); err != nil {
			return false, false, err
		}
		return true, false, nil
	}
	if err := SettleLot(lot, nowMs); err != nil {
		return false, false, err
	}
	if uc.orders == nil {
		return false, false, errors.New("order repository is required")
	}
	order, err := NewOrderFromSettledLot(idgen.New("order"), lot, nowMs)
	if err != nil {
		return false, false, err
	}
	settledEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED, lot)
	settledEvent.Ranking = ranking
	closedEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_AUCTION_CLOSED, lot)
	closedEvent.Ranking = ranking
	orderEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED, lot)
	orderEvent.Ranking = ranking
	orderEvent.Reason = orderCreatedPublicReason
	events := []v1.AuctionEvent{closedEvent, settledEvent, orderEvent}
	if err := uc.orders.CreateOrderForSettledLot(ctx, *order, lot, expectedVersion, events); err != nil {
		return false, false, err
	}
	if err := uc.broadcast(ctx, events...); err != nil {
		return false, false, err
	}
	return true, true, nil
}

func (uc *AuctionUsecase) GetLotResult(ctx context.Context, lotID string, viewer LotResultViewer) (*LotResult, error) {
	if lotID == "" {
		return nil, fmt.Errorf("%w: lot id is required", apperr.ErrInvalidArgument)
	}
	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, err
	}
	result := &LotResult{Lot: LotForViewer(lot, viewer), AuctionState: AuctionStateOf(lot)}
	if uc.orders != nil {
		order, found, err := uc.orders.FindOrderByLot(ctx, lotID)
		if err != nil {
			return nil, err
		}
		if found && viewer.CanViewOrder(order) {
			summary := order.Summary()
			result.Order = &summary
		}
	}
	return result, nil
}

func (uc *AuctionUsecase) ListOrdersByBuyer(ctx context.Context, buyerUserID string) ([]OrderSummary, error) {
	if buyerUserID == "" {
		return nil, fmt.Errorf("%w: buyer user id is required", apperr.ErrInvalidArgument)
	}
	if uc.orders == nil {
		return nil, errors.New("order repository is required")
	}
	list, err := uc.ListOrders(ctx, OrderQuery{BuyerUserID: buyerUserID})
	if err != nil {
		return nil, err
	}
	return list.Orders, nil
}

func (uc *AuctionUsecase) ListOrders(ctx context.Context, query OrderQuery) (OrderList, error) {
	if uc.orders == nil {
		return OrderList{}, errors.New("order repository is required")
	}
	query.Page, query.PageSize = NormalizePagination(query.Page, query.PageSize)
	return uc.orders.ListOrders(ctx, query)
}

func (uc *AuctionUsecase) ListRoomEvents(ctx context.Context, query RoomEventQuery) (RoomEventList, error) {
	if query.RoomID == "" {
		return RoomEventList{}, errors.New("room id is required")
	}
	if uc.eventsStore == nil {
		return RoomEventList{}, errors.New("event repository is required")
	}
	return uc.eventsStore.ListRoomEvents(ctx, query)
}

func (uc *AuctionUsecase) ListOrdersByBuyerQuery(ctx context.Context, buyerUserID string, query OrderQuery) (OrderList, error) {
	if buyerUserID == "" {
		return OrderList{}, fmt.Errorf("%w: buyer user id is required", apperr.ErrInvalidArgument)
	}
	query.BuyerUserID = buyerUserID
	query.Buyer = ""
	return uc.ListOrders(ctx, query)
}

func (uc *AuctionUsecase) ListBidRecordsByBuyer(ctx context.Context, buyerUserID string, query BidRecordQuery) (BidRecordList, error) {
	if buyerUserID == "" {
		return BidRecordList{}, fmt.Errorf("%w: buyer user id is required", apperr.ErrInvalidArgument)
	}
	query.Page, query.PageSize = NormalizePagination(query.Page, query.PageSize)
	return uc.bids.ListBidRecordsByBuyer(ctx, buyerUserID, query)
}

func (uc *AuctionUsecase) MockPayOrder(ctx context.Context, buyerUserID, orderID string, req MockPayRequest) (*PaymentResult, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if buyerUserID == "" {
		return nil, fmt.Errorf("%w: buyer user id is required", apperr.ErrInvalidArgument)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w: order id is required", apperr.ErrInvalidArgument)
	}
	if uc.orders == nil || uc.payments == nil {
		return nil, errors.New("order and payment repositories are required")
	}
	order, err := uc.orders.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerUserID != buyerUserID {
		return nil, fmt.Errorf("%w: order does not belong to current buyer", apperr.ErrPermissionDenied)
	}
	if req.IdempotencyKey == "" {
		return nil, fmt.Errorf("%w: payment idempotency key is required", apperr.ErrInvalidArgument)
	}
	if existing, found, err := uc.payments.FindPaymentByIdempotencyKey(ctx, orderID, req.IdempotencyKey); err != nil {
		return nil, err
	} else if found {
		return uc.replayPayment(ctx, orderID, existing)
	}
	if req.Currency == "" {
		return nil, fmt.Errorf("%w: payment currency is required", apperr.ErrInvalidArgument)
	}
	expectedVersion := order.Version
	nowMs := clock.NowMs()
	payment, err := NewPayment(idgen.New("pay"), *order, req.IdempotencyKey, req.Amount, req.Currency, nowMs)
	if err != nil {
		return nil, err
	}
	if err := payment.MarkProcessing(nowMs); err != nil {
		return nil, err
	}
	if err := payment.MarkSuccess(nowMs); err != nil {
		return nil, err
	}
	if err := MarkOrderPaid(order, *payment, nowMs); err != nil {
		return nil, err
	}
	lot, err := uc.lots.FindByID(ctx, order.LotID)
	if err != nil {
		return nil, err
	}
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_PAYMENT_SUCCESS, lot)
	event.Reason = paymentSuccessPublicReason
	if err := uc.payments.CommitPaymentSuccess(ctx, *payment, *order, expectedVersion, []v1.AuctionEvent{event}); err != nil {
		if existing, found, replayErr := uc.payments.FindPaymentByIdempotencyKey(ctx, orderID, req.IdempotencyKey); replayErr != nil {
			return nil, replayErr
		} else if found {
			return uc.replayPayment(ctx, orderID, existing)
		}
		return nil, err
	}
	if err := uc.broadcast(ctx, event); err != nil {
		return nil, err
	}
	return &PaymentResult{Order: order.Summary(), Payment: payment.Summary(), Paid: true}, nil
}

func (uc *AuctionUsecase) replayPayment(ctx context.Context, orderID string, payment *Payment) (*PaymentResult, error) {
	if payment == nil {
		return nil, fmt.Errorf("%w: payment is required", apperr.ErrInvalidArgument)
	}
	order, err := uc.orders.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return &PaymentResult{Order: order.Summary(), Payment: payment.Summary(), Paid: payment.Status == PaymentStatusSuccess}, nil
}

func (uc *AuctionUsecase) CancelLot(ctx context.Context, lotID, operatorID, reason string) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if lotID == "" {
		return nil, errors.New("lot id is required")
	}

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, err
	}
	expectedVersion := lot.Version
	if err := CancelLot(lot, reason, clock.NowMs()); err != nil {
		return lot, err
	}
	bids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return nil, err
	}
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CANCELLED, lot)
	event.Ranking = BuildRanking(bids)
	event.Reason = reason
	if err := uc.lots.Save(ctx, lot, expectedVersion, []v1.AuctionEvent{event}); err != nil {
		return nil, err
	}
	if err := uc.broadcast(ctx, event); err != nil {
		return nil, err
	}
	return proto.Clone(lot).(*v1.Lot), nil
}

func (uc *AuctionUsecase) persistEvents(ctx context.Context, events ...v1.AuctionEvent) error {
	if uc.eventsStore == nil || len(events) == 0 {
		return nil
	}
	return uc.eventsStore.PersistEvents(ctx, events)
}

func (uc *AuctionUsecase) broadcast(ctx context.Context, events ...v1.AuctionEvent) error {
	if uc.events == nil {
		return nil
	}
	for _, event := range events {
		if err := uc.events.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (uc *AuctionUsecase) Snapshot(ctx context.Context, roomID string) (*v1.RoomSnapshot, error) {
	if roomID == "" {
		return nil, errors.New("room id is required")
	}

	lots, err := uc.lots.List(ctx, roomID, 0)
	if err != nil {
		return nil, err
	}
	var current *v1.Lot
	for _, lot := range lots {
		if IsAuctionOpenStatus(lot.Status) {
			current = lot
			break
		}
	}

	snapshot := &v1.RoomSnapshot{
		RoomId:           roomID,
		PlaybookStage:    v1.PlaybookStage_PLAYBOOK_STAGE_WARM_UP,
		ServerTimeUnixMs: clock.NowMs(),
	}
	if current == nil {
		return snapshot, nil
	}

	bids, err := uc.bids.ListByLot(ctx, current.Id)
	if err != nil {
		return nil, err
	}
	snapshot.CurrentLot = current
	snapshot.Ranking = BuildRanking(bids)
	start := 0
	if len(bids) > 20 {
		start = len(bids) - 20
	}
	for i := start; i < len(bids); i++ {
		bid := bids[i]
		snapshot.RecentBids = append(snapshot.RecentBids, &bid)
	}
	snapshot.PlaybookStage = current.PlaybookStage
	return snapshot, nil
}
