package auction

import (
	"context"
	"sync"

	eventbuilder "live-auction-bid/backend/app/auction/service/internal/event"
	"live-auction-bid/backend/app/auction/service/internal/model"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

// AuctionUsecase 编排直播竞拍业务流程。
//
// 分层约束：
// - 领域规则放在 Lot / Ranking 等领域对象里；
// - 存储细节通过 Repository 接口隔离；
// - 实时广播通过 EventPublisher 接口隔离；
// - HTTP/WS 入参适配不放在 biz 层。
type AuctionUsecase struct {
	mu     sync.Mutex
	lots   LotRepository
	bids   BidRepository
	events EventPublisher
}

func NewAuctionUsecase(lots LotRepository, bids BidRepository, events EventPublisher) *AuctionUsecase {
	return &AuctionUsecase{lots: lots, bids: bids, events: events}
}

func (uc *AuctionUsecase) CreateLot(ctx context.Context, cmd model.CreateLotCommand) (*model.Lot, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot := NewLotFromCommand(idgen.New("lot"), cmd)
	if err := uc.lots.Create(ctx, lot); err != nil {
		return nil, err
	}
	uc.publish(ctx, eventbuilder.NewAuctionEvent(model.EventLotCreated, lot))
	return model.CloneLot(lot), nil
}

func (uc *AuctionUsecase) GetLot(ctx context.Context, lotID string) (*model.Lot, error) {
	return uc.lots.FindByID(ctx, lotID)
}

func (uc *AuctionUsecase) ListLots(ctx context.Context, roomID string, status model.LotStatus) ([]*model.Lot, error) {
	return uc.lots.List(ctx, roomID, status)
}

func (uc *AuctionUsecase) StartLot(ctx context.Context, lotID string) (*model.Lot, error) {
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

	event := eventbuilder.NewAuctionEvent(model.EventLotStarted, lot)
	event.Ranking = uc.mustRanking(ctx, lot.ID)
	uc.publish(ctx, event)
	return model.CloneLot(lot), nil
}

func (uc *AuctionUsecase) PlaceBid(ctx context.Context, cmd model.PlaceBidCommand) (*model.Lot, *model.Bid, []model.RankingItem, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := uc.lots.FindByID(ctx, cmd.LotID)
	if err != nil {
		return nil, nil, nil, err
	}

	if cmd.IdempotencyKey != "" {
		old, found, err := uc.bids.FindByIdempotencyKey(ctx, lot.ID, cmd.IdempotencyKey)
		if err != nil {
			return nil, nil, nil, err
		}
		if found {
			ranking := uc.mustRanking(ctx, lot.ID)
			return model.CloneLot(lot), &old, ranking, nil
		}
	}

	bid := newBidFromCommand(lot.ID, cmd)
	if err := AcceptBid(lot, bid, clock.NowMs()); err != nil {
		uc.publishRejected(ctx, lot, err.Error())
		return model.CloneLot(lot), nil, uc.mustRanking(ctx, lot.ID), err
	}

	if err := uc.bids.Append(ctx, bid); err != nil {
		return nil, nil, nil, err
	}
	if cmd.IdempotencyKey != "" {
		if err := uc.bids.SaveIdempotencyKey(ctx, lot.ID, cmd.IdempotencyKey, bid); err != nil {
			return nil, nil, nil, err
		}
	}

	bids, err := uc.bids.ListByLot(ctx, lot.ID)
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
	if lot.DuelState.Active {
		uc.publishDuelStarted(ctx, lot, ranking)
	}
	return model.CloneLot(lot), &bid, ranking, nil
}

func (uc *AuctionUsecase) RevealTrustCard(ctx context.Context, lotID, cardID, operatorID string) (*model.Lot, *model.TrustRevealCard, error) {
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

	event := eventbuilder.NewAuctionEvent(model.EventTrustRevealed, lot)
	event.TrustCard = card
	event.Ranking = uc.mustRanking(ctx, lot.ID)
	uc.publish(ctx, event)
	return model.CloneLot(lot), card, nil
}

func (uc *AuctionUsecase) StartDuel(ctx context.Context, lotID, operatorID, userAID, userBID string) (*model.Lot, *model.DuelState, error) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	lot, err := uc.lots.FindByID(ctx, lotID)
	if err != nil {
		return nil, nil, err
	}
	ranking := uc.mustRanking(ctx, lot.ID)
	if err := StartDuel(lot, ranking, clock.NowMs()); err != nil {
		return nil, nil, err
	}
	if err := uc.lots.Save(ctx, lot); err != nil {
		return nil, nil, err
	}

	uc.publishDuelStarted(ctx, lot, ranking)
	return model.CloneLot(lot), &lot.DuelState, nil
}

func (uc *AuctionUsecase) SettleLot(ctx context.Context, lotID, operatorID string) (*model.Lot, error) {
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

	event := eventbuilder.NewAuctionEvent(model.EventLotSettled, lot)
	event.Ranking = uc.mustRanking(ctx, lot.ID)
	uc.publish(ctx, event)
	return model.CloneLot(lot), nil
}

func (uc *AuctionUsecase) Snapshot(ctx context.Context, roomID string) (*model.RoomSnapshot, error) {
	lots, err := uc.lots.List(ctx, roomID, "")
	if err != nil {
		return nil, err
	}
	current := pickCurrentLot(lots)

	snapshot := &model.RoomSnapshot{
		RoomID:           roomID,
		PlaybookStage:    model.PlaybookStageWarmUp,
		ServerTimeUnixMs: clock.NowMs(),
	}
	if current == nil {
		return snapshot, nil
	}

	bids, err := uc.bids.ListByLot(ctx, current.ID)
	if err != nil {
		return nil, err
	}
	snapshot.CurrentLot = current
	snapshot.Ranking = BuildRanking(bids)
	snapshot.RecentBids = RecentBids(bids, 20)
	snapshot.PlaybookStage = current.PlaybookStage
	return snapshot, nil
}

func newBidFromCommand(lotID string, cmd model.PlaceBidCommand) model.Bid {
	if cmd.UserID == "" {
		cmd.UserID = idgen.New("guest")
	}
	if cmd.Nickname == "" {
		cmd.Nickname = "游客"
	}
	if cmd.Amount.Currency == "" {
		cmd.Amount.Currency = "CNY"
	}
	return model.Bid{
		ID:              idgen.New("bid"),
		LotID:           lotID,
		UserID:          cmd.UserID,
		Nickname:        cmd.Nickname,
		Amount:          cmd.Amount,
		CreatedAtUnixMs: clock.NowMs(),
	}
}

func pickCurrentLot(lots []*model.Lot) *model.Lot {
	var fallback *model.Lot
	for _, lot := range lots {
		if lot.Status == model.LotStatusLive {
			return lot
		}
		fallback = lot
	}
	return fallback
}

func (uc *AuctionUsecase) mustRanking(ctx context.Context, lotID string) []model.RankingItem {
	bids, err := uc.bids.ListByLot(ctx, lotID)
	if err != nil {
		return nil
	}
	return BuildRanking(bids)
}

func (uc *AuctionUsecase) publish(ctx context.Context, event model.AuctionEvent) {
	if uc.events != nil {
		uc.events.Publish(ctx, event)
	}
}

func (uc *AuctionUsecase) publishRejected(ctx context.Context, lot *model.Lot, reason string) {
	event := eventbuilder.NewAuctionEvent(model.EventBidRejected, lot)
	event.Reason = reason
	uc.publish(ctx, event)
}

func (uc *AuctionUsecase) publishBidAccepted(ctx context.Context, lot *model.Lot, bid model.Bid, ranking []model.RankingItem) {
	event := eventbuilder.NewAuctionEvent(model.EventBidAccepted, lot)
	event.Bid = &bid
	event.Ranking = ranking
	uc.publish(ctx, event)
}

func (uc *AuctionUsecase) publishRankingUpdated(ctx context.Context, lot *model.Lot, ranking []model.RankingItem) {
	event := eventbuilder.NewAuctionEvent(model.EventRankingUpdated, lot)
	event.Ranking = ranking
	uc.publish(ctx, event)
}

func (uc *AuctionUsecase) publishDuelStarted(ctx context.Context, lot *model.Lot, ranking []model.RankingItem) {
	event := eventbuilder.NewAuctionEvent(model.EventDuelStarted, lot)
	event.Ranking = ranking
	event.DuelState = &lot.DuelState
	uc.publish(ctx, event)
}
