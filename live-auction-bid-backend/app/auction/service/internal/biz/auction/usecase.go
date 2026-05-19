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

func (uc *AuctionUsecase) CreateLot(ctx context.Context, cmd CreateLotCommand) (*v1.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot := NewLotFromCommand(idgen.New("lot"), cmd)
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

func (uc *AuctionUsecase) PlaceBid(ctx context.Context, cmd PlaceBidCommand) (*v1.Lot, *v1.Bid, []*v1.RankingItem, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := uc.lots.FindByID(ctx, cmd.LotID)
	if err != nil {
		return nil, nil, nil, err
	}

	if cmd.IdempotencyKey != "" {
		old, found, err := uc.bids.FindByIdempotencyKey(ctx, lot.Id, cmd.IdempotencyKey)
		if err != nil {
			return nil, nil, nil, err
		}
		if found {
			ranking := uc.mustRanking(ctx, lot.Id)
			return CloneLot(lot), &old, ranking, nil
		}
	}

	bid := newBidFromCommand(lot.Id, cmd)
	if err := AcceptBid(lot, bid, clock.NowMs()); err != nil {
		uc.publishRejected(ctx, lot, err.Error())
		return CloneLot(lot), nil, uc.mustRanking(ctx, lot.Id), err
	}

	if err := uc.bids.Append(ctx, bid); err != nil {
		return nil, nil, nil, err
	}
	if cmd.IdempotencyKey != "" {
		if err := uc.bids.SaveIdempotencyKey(ctx, lot.Id, cmd.IdempotencyKey, bid); err != nil {
			return nil, nil, nil, err
		}
	}

	bids, err := uc.bids.ListByLot(ctx, lot.Id)
	if err != nil {
		return nil, nil, nil, err
	}
	ranking := BuildRanking(bids)
	if ShouldAutoStartDuel(lot, ranking, bids, clock.NowMs()) {
		_ = StartDuel(lot, ranking, clock.NowMs())
	}

	if err := uc.lots.Save(ctx, lot); err != nil {
		return nil, nil, nil, err
	}

	uc.publishBidAccepted(ctx, lot, bid, ranking)
	uc.publishRankingUpdated(ctx, lot, ranking)
	if lot.GetDuelState().GetActive() {
		uc.publishDuelStarted(ctx, lot, ranking)
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

	uc.publishDuelStarted(ctx, lot, ranking)
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
	current := pickCurrentLot(lots)

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
	snapshot.RecentBids = RecentBids(bids, 20)
	snapshot.PlaybookStage = current.PlaybookStage
	return snapshot, nil
}

func newBidFromCommand(lotID string, cmd PlaceBidCommand) v1.Bid {
	if cmd.UserID == "" {
		cmd.UserID = idgen.New("guest")
	}
	if cmd.Nickname == "" {
		cmd.Nickname = "游客"
	}
	if cmd.Amount == nil {
		cmd.Amount = CNY(0)
	}
	if cmd.Amount.Currency == "" {
		cmd.Amount.Currency = "CNY"
	}
	return v1.Bid{
		Id:              idgen.New("bid"),
		LotId:           lotID,
		UserId:          cmd.UserID,
		Nickname:        cmd.Nickname,
		Amount:          cmd.Amount,
		CreatedAtUnixMs: clock.NowMs(),
	}
}

func pickCurrentLot(lots []*v1.Lot) *v1.Lot {
	var fallback *v1.Lot
	for _, lot := range lots {
		if lot.Status == v1.LotStatus_LOT_STATUS_LIVE {
			return lot
		}
		fallback = lot
	}
	return fallback
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

func (uc *AuctionUsecase) publishRejected(ctx context.Context, lot *v1.Lot, reason string) {
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_REJECTED, lot)
	event.Reason = reason
	uc.publish(ctx, event)
}

func (uc *AuctionUsecase) publishBidAccepted(ctx context.Context, lot *v1.Lot, bid v1.Bid, ranking []*v1.RankingItem) {
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED, lot)
	event.Bid = &bid
	event.Ranking = ranking
	uc.publish(ctx, event)
}

func (uc *AuctionUsecase) publishRankingUpdated(ctx context.Context, lot *v1.Lot, ranking []*v1.RankingItem) {
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_RANKING_UPDATED, lot)
	event.Ranking = ranking
	uc.publish(ctx, event)
}

func (uc *AuctionUsecase) publishDuelStarted(ctx context.Context, lot *v1.Lot, ranking []*v1.RankingItem) {
	event := newAuctionEvent(v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED, lot)
	event.Ranking = ranking
	event.DuelState = lot.DuelState
	uc.publish(ctx, event)
}
