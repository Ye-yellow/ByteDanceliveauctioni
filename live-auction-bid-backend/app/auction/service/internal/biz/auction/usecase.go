package auction

import (
	"context"
	"sync"

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
	mu     sync.Mutex
	lots   LotRepository
	bids   BidRepository
	events EventPublisher
}

func NewAuctionUsecase(lots LotRepository, bids BidRepository, events EventPublisher) *AuctionUsecase {
	return &AuctionUsecase{lots: lots, bids: bids, events: events}
}

func (uc *AuctionUsecase) CreateLot(ctx context.Context, req *v1.CreateLotRequest) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := NewLotFromRequest(idgen.New("lot"), req)
	if err != nil {
		return nil, err
	}
	if err := uc.lots.Create(ctx, lot); err != nil {
		return nil, err
	}
	uc.publish(ctx, newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CREATED, lot))
	return CloneLot(lot), nil
}

func (uc *AuctionUsecase) GetLot(ctx context.Context, lotID string) (*v1.Lot, error) {
	return uc.lots.FindByID(ctx, lotID)
}

func (uc *AuctionUsecase) ListLots(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error) {
	return uc.lots.List(ctx, roomID, status)
}

func (uc *AuctionUsecase) StartLot(ctx context.Context, lotID string) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, err
	}
	if err := StartLot(lot, clock.NowMs()); err != nil {
		return nil, err
	}
	if err := uc.lots.Save(ctx, lot); err != nil {
		return nil, err
	}

	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_STARTED, lot)
	event.Ranking = uc.mustRanking(ctx, lot.Id)
	uc.publish(ctx, event)
	return CloneLot(lot), nil
}

func (uc *AuctionUsecase) PlaceBid(ctx context.Context, req *v1.PlaceBidRequest) (*v1.Lot, *v1.Bid, []*v1.RankingItem, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := uc.lots.FindByID(ctx, req.GetLotId())
	if err != nil {
		return nil, nil, nil, err
	}

	if req.GetIdempotencyKey() != "" {
		old, found, err := uc.bids.FindByIdempotencyKey(ctx, lot.Id, req.GetIdempotencyKey())
		if err != nil {
			return nil, nil, nil, err
		}
		if found {
			ranking := uc.mustRanking(ctx, lot.Id)
			return CloneLot(lot), &old, ranking, nil
		}
	}

	userID := req.GetUserId()
	if userID == "" {
		userID = idgen.New("guest")
	}
	nickname := req.GetNickname()
	if nickname == "" {
		nickname = "游客"
	}
	amount := req.GetAmount()
	if amount == nil {
		amount = CNY(0)
	}
	if amount.Currency == "" {
		amount.Currency = "CNY"
	}
	bid := v1.Bid{
		Id:              idgen.New("bid"),
		LotId:           lot.Id,
		UserId:          userID,
		Nickname:        nickname,
		Amount:          amount,
		CreatedAtUnixMs: clock.NowMs(),
	}
	if err := AcceptBid(lot, bid, clock.NowMs()); err != nil {
		event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_REJECTED, lot)
		event.Reason = err.Error()
		uc.publish(ctx, event)
		return CloneLot(lot), nil, uc.mustRanking(ctx, lot.Id), err
	}

	if err := uc.bids.Append(ctx, bid); err != nil {
		return nil, nil, nil, err
	}
	if req.GetIdempotencyKey() != "" {
		if err := uc.bids.SaveIdempotencyKey(ctx, lot.Id, req.GetIdempotencyKey(), bid); err != nil {
			return nil, nil, nil, err
		}
	}

	bids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return nil, nil, nil, err
	}
	ranking := BuildRanking(bids)
	nowMs := clock.NowMs()
	if !lot.GetDuelState().GetActive() && len(ranking) >= 2 && len(bids) >= 3 &&
		lot.EndsAtUnixMs-nowMs <= 60_000 &&
		ranking[0].GetAmount().GetAmount()-ranking[1].GetAmount().GetAmount() <= lot.GetRule().GetMinIncrement().GetAmount()*3 {
		_ = StartDuel(lot, ranking, nowMs)
	}

	if err := uc.lots.Save(ctx, lot); err != nil {
		return nil, nil, nil, err
	}

	acceptedEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED, lot)
	acceptedEvent.Bid = &bid
	acceptedEvent.Ranking = ranking
	uc.publish(ctx, acceptedEvent)

	rankingEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_RANKING_UPDATED, lot)
	rankingEvent.Ranking = ranking
	uc.publish(ctx, rankingEvent)

	if lot.GetDuelState().GetActive() {
		duelEvent := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED, lot)
		duelEvent.Ranking = ranking
		duelEvent.DuelState = lot.DuelState
		uc.publish(ctx, duelEvent)
	}
	return CloneLot(lot), &bid, ranking, nil
}

func (uc *AuctionUsecase) RevealTrustCard(ctx context.Context, lotID, cardID, operatorID string) (*v1.Lot, *v1.TrustRevealCard, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, nil, err
	}
	card, err := RevealTrustCard(lot, cardID, clock.NowMs())
	if err != nil {
		return nil, nil, err
	}
	if err := uc.lots.Save(ctx, lot); err != nil {
		return nil, nil, err
	}

	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_TRUST_REVEALED, lot)
	event.TrustCard = card
	event.Ranking = uc.mustRanking(ctx, lot.Id)
	uc.publish(ctx, event)
	return CloneLot(lot), card, nil
}

func (uc *AuctionUsecase) StartDuel(ctx context.Context, lotID, operatorID, userAID, userBID string) (*v1.Lot, *v1.DuelState, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, nil, err
	}
	ranking := uc.mustRanking(ctx, lot.Id)
	if err := StartDuel(lot, ranking, clock.NowMs()); err != nil {
		return nil, nil, err
	}
	if err := uc.lots.Save(ctx, lot); err != nil {
		return nil, nil, err
	}

	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED, lot)
	event.Ranking = ranking
	event.DuelState = lot.DuelState
	uc.publish(ctx, event)
	return CloneLot(lot), lot.DuelState, nil
}

func (uc *AuctionUsecase) SettleLot(ctx context.Context, lotID, operatorID string) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, err
	}
	if err := SettleLot(lot, clock.NowMs()); err != nil {
		return nil, err
	}
	if err := uc.lots.Save(ctx, lot); err != nil {
		return nil, err
	}

	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED, lot)
	event.Ranking = uc.mustRanking(ctx, lot.Id)
	uc.publish(ctx, event)
	return CloneLot(lot), nil
}

func (uc *AuctionUsecase) Snapshot(ctx context.Context, roomID string) (*v1.RoomSnapshot, error) {
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

func (uc *AuctionUsecase) mustRanking(ctx context.Context, lotID string) []*v1.RankingItem {
	bids, err := uc.bids.ListByLot(ctx, lotID)
	if err != nil {
		return nil
	}
	return BuildRanking(bids)
}

func (uc *AuctionUsecase) publish(ctx context.Context, event v1.AuctionEvent) {
	if uc.events != nil {
		uc.events.Publish(ctx, event)
	}
}
