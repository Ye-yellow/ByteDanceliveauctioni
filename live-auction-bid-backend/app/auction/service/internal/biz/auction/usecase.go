package auction

import (
	"context"
	"errors"
	"sync"

	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
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
	bids        BidRepository
	eventsStore EventRepository
	events      EventPublisher
}

func NewAuctionUsecase(lots LotRepository, bids BidRepository, eventStore EventRepository, events EventPublisher) *AuctionUsecase {
	return &AuctionUsecase{lots: lots, bids: bids, eventsStore: eventStore, events: events}
}

func (uc *AuctionUsecase) CreateLot(ctx context.Context, req *v1.CreateLotRequest) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := NewLotFromRequest(idgen.New("lot"), req)
	if err != nil {
		return nil, err
	}
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CREATED, lot)
	if err := uc.lots.Create(ctx, lot, []v1.AuctionEvent{event}); err != nil {
		return nil, err
	}
	if err := uc.broadcast(ctx, event); err != nil {
		return nil, err
	}
	return proto.Clone(lot).(*v1.Lot), nil
}

func (uc *AuctionUsecase) GetLot(ctx context.Context, lotID string) (*v1.Lot, error) {
	if lotID == "" {
		return nil, errors.New("lot id is required")
	}
	return uc.lots.FindByID(ctx, lotID)
}

func (uc *AuctionUsecase) ListLots(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error) {
	if roomID == "" {
		return nil, errors.New("room id is required")
	}
	return uc.lots.List(ctx, roomID, status)
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

	lot, err := uc.lots.FindByID(ctx, req.GetLotId())
	if err != nil {
		return nil, nil, nil, err
	}
	expectedVersion := lot.Version

	if req.GetIdempotencyKey() != "" {
		old, found, err := uc.bids.FindByIdempotencyKey(ctx, lot.Id, req.GetIdempotencyKey())
		if err != nil {
			return nil, nil, nil, err
		}
		if found {
			bids, err := uc.bids.ListByLot(ctx, lot.Id)
			if err != nil {
				return nil, nil, nil, err
			}
			return proto.Clone(lot).(*v1.Lot), &old, BuildRanking(bids), nil
		}
	}

	bid := v1.Bid{
		Id:              idgen.New("bid"),
		LotId:           lot.Id,
		UserId:          bidderID,
		Nickname:        nickname,
		Amount:          req.GetAmount(),
		CreatedAtUnixMs: clock.NowMs(),
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
	if !lot.GetDuelState().GetActive() && len(ranking) >= 2 && len(bids) >= 3 &&
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
	if lot.GetDuelState().GetActive() {
		duelEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED, lot)
		duelEvent.Ranking = ranking
		duelEvent.DuelState = lot.DuelState
		commitEvents = append(commitEvents, duelEvent)
	}
	if err := uc.bids.CommitAcceptedBid(ctx, bid, lot, expectedVersion, req.GetIdempotencyKey(), commitEvents); err != nil {
		return nil, nil, nil, err
	}
	if req.GetIdempotencyKey() != "" {
		uc.bids.CacheIdempotencyKey(ctx, lot.Id, req.GetIdempotencyKey(), bid)
	}

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
	if err := uc.lots.Save(ctx, lot, expectedVersion, []v1.AuctionEvent{event}); err != nil {
		return nil, err
	}
	if err := uc.broadcast(ctx, event); err != nil {
		return nil, err
	}
	return proto.Clone(lot).(*v1.Lot), nil
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
		if lot.Status == v1.LotStatus_LOT_STATUS_LIVE {
			current = lot
			break
		}
		current = lot
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
