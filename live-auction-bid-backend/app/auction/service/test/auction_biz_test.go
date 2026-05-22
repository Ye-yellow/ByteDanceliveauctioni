package test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"

	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/clock"
)

func TestLotStateMachine(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_1", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("创建拍品失败：%v", err)
	}

	if err := auction.AcceptBid(lot, v1.Bid{Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, clock.NowMs()); err == nil {
		t.Fatal("DRAFT 状态不应该允许出价")
	}
	if err := auction.StartLot(lot, 1000); err != nil {
		t.Fatalf("开拍失败：%v", err)
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		t.Fatalf("期望状态 LIVE，实际 %s", lot.Status)
	}
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 10500, Currency: "CNY"}}, 2000); err == nil {
		t.Fatal("低于最低加价的出价应该被拒绝")
	}
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, 2000); err != nil {
		t.Fatalf("合法出价失败：%v", err)
	}
	if lot.GetCurrentPrice().GetAmount() != 11000 || lot.LeadingUserId != "u1" {
		t.Fatalf("出价后领先状态错误：%+v", lot)
	}
	if err := auction.SettleLot(lot, 3000); err != nil {
		t.Fatalf("落锤失败：%v", err)
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_SETTLED || lot.WinnerUserId != "u1" || lot.GetFinalPrice().GetAmount() != 11000 {
		t.Fatalf("成交状态错误：%+v", lot)
	}
}

func TestCancelLotStateMachineAllowsOnlyLiveAndRecordsReason(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_cancel", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "取消测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 0, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if err := auction.CancelLot(lot, "主播网络异常", 2000); err == nil || !strings.Contains(err.Error(), "only live lot") {
		t.Fatalf("draft lot should not be cancellable, got %v", err)
	}
	if err := auction.StartLot(lot, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if err := auction.CancelLot(lot, "", 2000); err == nil || !strings.Contains(err.Error(), "cancel reason") {
		t.Fatalf("empty cancel reason should be rejected, got %v", err)
	}
	if err := auction.CancelLot(lot, "主播网络异常", 3000); err != nil {
		t.Fatalf("cancel live lot failed: %v", err)
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_CANCELLED || lot.CancelReason != "主播网络异常" || lot.CancelledAtUnixMs != 3000 {
		t.Fatalf("cancelled lot state mismatch: %+v", lot)
	}
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 1000, Currency: "CNY"}}, 4000); err == nil || !strings.Contains(err.Error(), "lot is not live") {
		t.Fatalf("cancelled lot should reject bids, got %v", err)
	}
}

func TestAntiSnipeExtensionKeepsDuelStateInSync(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_extend", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "延时测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        60,
			AntiSnipeWindowSeconds: 10,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         2,
		},
	})
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if err := auction.StartLot(lot, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	originalEndsAt := lot.EndsAtUnixMs
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, originalEndsAt-5_000); err != nil {
		t.Fatalf("bid in anti-snipe window failed: %v", err)
	}
	if lot.EndsAtUnixMs != originalEndsAt+15_000 {
		t.Fatalf("expected lot ends_at to extend from %d to %d, got %d", originalEndsAt, originalEndsAt+15_000, lot.EndsAtUnixMs)
	}
	if lot.GetDuelState().GetExtendCount() != 1 || lot.GetDuelState().GetLotId() != lot.Id || lot.GetDuelState().GetEndsAtUnixMs() != lot.EndsAtUnixMs || lot.GetDuelState().GetMaxExtendCount() != 2 {
		t.Fatalf("duel state should mirror anti-snipe extension counters: %+v", lot.GetDuelState())
	}

	secondEndsAt := lot.EndsAtUnixMs
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u2", Nickname: "用户2", Amount: &v1.Money{Amount: 12000, Currency: "CNY"}}, secondEndsAt-5_000); err != nil {
		t.Fatalf("second extension bid failed: %v", err)
	}
	thirdEndsAt := lot.EndsAtUnixMs
	if err := auction.AcceptBid(lot, v1.Bid{UserId: "u3", Nickname: "用户3", Amount: &v1.Money{Amount: 13000, Currency: "CNY"}}, thirdEndsAt-5_000); err != nil {
		t.Fatalf("third bid at max extension boundary failed: %v", err)
	}
	if lot.EndsAtUnixMs != thirdEndsAt || lot.GetDuelState().GetExtendCount() != 2 {
		t.Fatalf("extension should stop at max count, lot=%+v duel=%+v", lot, lot.GetDuelState())
	}
}

func TestBuildRanking(t *testing.T) {
	bids := []v1.Bid{
		{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, CreatedAtUnixMs: 1000},
		{UserId: "u2", Nickname: "用户2", Amount: &v1.Money{Amount: 12000, Currency: "CNY"}, CreatedAtUnixMs: 2000},
		{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 13000, Currency: "CNY"}, CreatedAtUnixMs: 3000},
	}

	ranking := auction.BuildRanking(bids)
	if len(ranking) != 2 {
		t.Fatalf("期望 2 个用户，实际 %d", len(ranking))
	}
	if ranking[0].UserId != "u1" || ranking[0].GetAmount().GetAmount() != 13000 {
		t.Fatalf("排名第一错误：%+v", ranking[0])
	}
	if ranking[1].UserId != "u2" || ranking[1].GetAmount().GetAmount() != 12000 {
		t.Fatalf("排名第二错误：%+v", ranking[1])
	}
}

func TestCreateLotRejectsMismatchedCurrency(t *testing.T) {
	_, err := auction.NewLotFromRequest("lot_1", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Currency: "CNY", Amount: 10000},
			MinIncrement:           &v1.Money{Currency: "USD", Amount: 1000},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "currency must match") {
		t.Fatalf("expected currency mismatch error, got %v", err)
	}
}

func TestCreateLotValidatesRequiredImageAndCapPrice(t *testing.T) {
	base := &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "封顶价测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
			CapPrice:               &v1.Money{Amount: 50000, Currency: "CNY"},
		},
	}
	lot, err := auction.NewLotFromRequest("lot_cap", base)
	if err != nil {
		t.Fatalf("valid cap price should pass: %v", err)
	}
	if lot.GetRule().GetCapPrice().GetAmount() != 50000 {
		t.Fatalf("cap price should be kept on lot: %+v", lot.GetRule().GetCapPrice())
	}

	missingImage := proto.Clone(base).(*v1.CreateLotRequest)
	missingImage.ImageUrl = ""
	if _, err := auction.NewLotFromRequest("lot_no_image", missingImage); err == nil || !strings.Contains(err.Error(), "image url") {
		t.Fatalf("missing image should be rejected, got %v", err)
	}

	badCap := proto.Clone(base).(*v1.CreateLotRequest)
	badCap.Rule.CapPrice = &v1.Money{Amount: 9000, Currency: "CNY"}
	if _, err := auction.NewLotFromRequest("lot_bad_cap", badCap); err == nil || !strings.Contains(err.Error(), "greater than start price") {
		t.Fatalf("cap <= start should be rejected, got %v", err)
	}
}

func TestPlaceBidRejectsMissingUserInsteadOfDefaulting(t *testing.T) {
	store := newTestStore()
	uc := auction.NewAuctionUsecase(store, store, store, nil)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "测试拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}

	_, _, _, err = uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId:  lot.Id,
		Amount: &v1.Money{Amount: 11000, Currency: "CNY"},
	}, "", "用户1")
	if err == nil || !strings.Contains(err.Error(), "user id is required") {
		t.Fatalf("expected user id error, got %v", err)
	}
}

func TestLotSaveRejectsStaleExpectedVersion(t *testing.T) {
	store := newTestStore()
	ctx := context.Background()
	lot, err := auction.NewLotFromRequest("lot_conflict", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "版本冲突测试",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if err := store.Create(ctx, lot, "", nil); err != nil {
		t.Fatalf("persist lot failed: %v", err)
	}

	fresh, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find lot failed: %v", err)
	}
	expectedVersion := fresh.Version
	if err := auction.StartLot(fresh, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if err := store.Save(ctx, fresh, expectedVersion, nil); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	stale, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find lot failed: %v", err)
	}
	if err := auction.AcceptBid(stale, v1.Bid{UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, 2000); err != nil {
		t.Fatalf("accept bid failed: %v", err)
	}
	if err := store.Save(ctx, stale, expectedVersion, nil); err == nil || !strings.Contains(err.Error(), "lot version conflict") {
		t.Fatalf("expected stale version conflict, got %v", err)
	}
}

func TestCommitAcceptedBidRejectsStaleLotWithoutAppendingBid(t *testing.T) {
	store := newTestStore()
	ctx := context.Background()
	lot, err := auction.NewLotFromRequest("lot_bid_conflict", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "出价事务冲突测试",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if err := store.Create(ctx, lot, "", nil); err != nil {
		t.Fatalf("persist lot failed: %v", err)
	}
	fresh, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find lot failed: %v", err)
	}
	if err := auction.StartLot(fresh, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if err := store.Save(ctx, fresh, fresh.Version-1, nil); err != nil {
		t.Fatalf("save started lot failed: %v", err)
	}

	stale := proto.Clone(fresh).(*v1.Lot)
	bid := v1.Bid{Id: "bid_conflict", LotId: stale.Id, UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, CreatedAtUnixMs: 2000}
	if err := auction.AcceptBid(stale, bid, 2000); err != nil {
		t.Fatalf("accept bid failed: %v", err)
	}
	if err := store.CommitAcceptedBid(ctx, bid, stale, fresh.Version-1, "idem-conflict", nil); err == nil || !strings.Contains(err.Error(), "lot version conflict") {
		t.Fatalf("expected lot version conflict, got %v", err)
	}
	bids, err := store.ListByLot(ctx, stale.Id)
	if err != nil {
		t.Fatalf("list bids failed: %v", err)
	}
	if len(bids) != 0 {
		t.Fatalf("stale commit must not append bid, got %+v", bids)
	}
}

type testStore struct {
	mu        sync.RWMutex
	lots      map[string]*v1.Lot
	bidsByLot map[string][]v1.Bid
	idemByLot map[string]map[string]v1.Bid
	events    []v1.AuctionEvent
}

func newTestStore() *testStore {
	return &testStore{
		lots:      make(map[string]*v1.Lot),
		bidsByLot: make(map[string][]v1.Bid),
		idemByLot: make(map[string]map[string]v1.Bid),
	}
}

func (s *testStore) Create(ctx context.Context, lot *v1.Lot, ownerUserID string, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lots[lot.Id] = proto.Clone(lot).(*v1.Lot)
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) Save(ctx context.Context, lot *v1.Lot, expectedVersion int64, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.lots[lot.Id]
	if !ok {
		return errors.New("lot not found")
	}
	if expectedVersion <= 0 {
		return errors.New("lot expected version is required")
	}
	if current.Version != expectedVersion {
		return apperr.ErrLotVersionConflict
	}
	s.lots[lot.Id] = proto.Clone(lot).(*v1.Lot)
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) FindByID(ctx context.Context, lotID string) (*v1.Lot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lot, ok := s.lots[lotID]
	if !ok {
		return nil, errors.New("lot not found")
	}
	return proto.Clone(lot).(*v1.Lot), nil
}

func (s *testStore) List(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lots := make([]*v1.Lot, 0, len(s.lots))
	for _, lot := range s.lots {
		if roomID != "" && lot.RoomId != roomID {
			continue
		}
		if status != 0 && lot.Status != status {
			continue
		}
		lots = append(lots, proto.Clone(lot).(*v1.Lot))
	}
	sort.Slice(lots, func(i, j int) bool { return lots[i].Id < lots[j].Id })
	return lots, nil
}

func (s *testStore) CommitAcceptedBid(ctx context.Context, bid v1.Bid, lot *v1.Lot, expectedLotVersion int64, idempotencyKey string, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.lots[lot.Id]
	if !ok {
		return errors.New("lot not found")
	}
	if expectedLotVersion <= 0 {
		return errors.New("lot expected version is required")
	}
	if current.Version != expectedLotVersion {
		return apperr.ErrLotVersionConflict
	}
	s.lots[lot.Id] = proto.Clone(lot).(*v1.Lot)
	s.bidsByLot[bid.LotId] = append(s.bidsByLot[bid.LotId], bid)
	if idempotencyKey != "" {
		if s.idemByLot[bid.LotId] == nil {
			s.idemByLot[bid.LotId] = make(map[string]v1.Bid)
		}
		s.idemByLot[bid.LotId][idempotencyKey] = bid
	}
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) PersistEvents(ctx context.Context, events []v1.AuctionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, events...)
	return nil
}

func (s *testStore) eventTypes() []v1.AuctionEventType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	types := make([]v1.AuctionEventType, 0, len(s.events))
	for _, event := range s.events {
		types = append(types, event.Type)
	}
	return types
}

func (s *testStore) ListByLot(ctx context.Context, lotID string) ([]v1.Bid, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]v1.Bid(nil), s.bidsByLot[lotID]...), nil
}

func (s *testStore) FindByIdempotencyKey(ctx context.Context, lotID, key string) (v1.Bid, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.idemByLot[lotID] == nil {
		return v1.Bid{}, false, nil
	}
	bid, ok := s.idemByLot[lotID][key]
	return bid, ok, nil
}

func (s *testStore) CacheIdempotencyKey(ctx context.Context, lotID, key string, bid v1.Bid) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.idemByLot[lotID] == nil {
		s.idemByLot[lotID] = make(map[string]v1.Bid)
	}
	if _, exists := s.idemByLot[lotID][key]; exists {
		return
	}
	s.idemByLot[lotID][key] = bid
}

func TestAuctionUsecaseCoreClosurePublishesEventsAndSnapshot(t *testing.T) {
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:      "room_core",
		Title:       "核心闭环拍品",
		ImageUrl:    "https://example.com/lot.jpg",
		Description: "核心业务闭环测试",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
		TrustCards: []*v1.TrustRevealCard{{Title: "证书", Content: "可复检"}},
	}, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if len(lot.TrustCards) != 1 || lot.TrustCards[0].Id == "" || lot.TrustCards[0].LotId != lot.Id {
		t.Fatalf("trust card should be normalized on create: %+v", lot.TrustCards)
	}
	if _, err := uc.StartLot(ctx, lot.Id); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	if _, bid, ranking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "idem-1",
	}, "u1", "用户1"); err != nil || bid == nil || len(ranking) != 1 {
		t.Fatalf("first bid failed: bid=%+v ranking=%+v err=%v", bid, ranking, err)
	}
	if _, bid, ranking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}, IdempotencyKey: "idem-1",
	}, "u1", "用户1"); err != nil || bid == nil || len(ranking) != 1 {
		t.Fatalf("idempotent bid replay failed: bid=%+v ranking=%+v err=%v", bid, ranking, err)
	}
	if _, _, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 12000, Currency: "CNY"},
	}, "u2", "用户2"); err != nil {
		t.Fatalf("second bid failed: %v", err)
	}
	if _, _, _, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{
		LotId: lot.Id, Amount: &v1.Money{Amount: 13000, Currency: "CNY"},
	}, "u1", "用户1"); err != nil {
		t.Fatalf("third bid failed: %v", err)
	}
	if _, card, err := uc.RevealTrustCard(ctx, lot.Id, lot.TrustCards[0].Id, "op"); err != nil || card == nil || !card.Revealed {
		t.Fatalf("reveal trust card failed: card=%+v err=%v", card, err)
	}
	if lotAfterDuel, duel, err := uc.StartDuel(ctx, lot.Id, "op", "u2", "u1"); err != nil || duel == nil || !duel.Active || duel.UserAId != "u2" || duel.UserBId != "u1" || lotAfterDuel.PlaybookStage != v1.PlaybookStage_PLAYBOOK_STAGE_DUEL_MODE {
		t.Fatalf("start duel failed: lot=%+v duel=%+v err=%v", lotAfterDuel, duel, err)
	}
	settled, err := uc.SettleLot(ctx, lot.Id, "op")
	if err != nil {
		t.Fatalf("settle failed: %v", err)
	}
	if settled.Status != v1.LotStatus_LOT_STATUS_SETTLED || settled.WinnerUserId != "u1" || settled.GetFinalPrice().GetAmount() != 13000 || settled.GetDuelState().GetActive() {
		t.Fatalf("settled lot state mismatch: %+v", settled)
	}
	snapshot, err := uc.Snapshot(ctx, "room_core")
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if snapshot.CurrentLot == nil || len(snapshot.Ranking) != 2 || len(snapshot.RecentBids) != 3 {
		t.Fatalf("snapshot mismatch: %+v", snapshot)
	}

	pub.assertContains(t,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CREATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_STARTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_RANKING_UPDATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_TRUST_REVEALED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED,
	)
	assertEventTypesContain(t, store.eventTypes(),
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CREATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_STARTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_RANKING_UPDATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_TRUST_REVEALED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED,
	)
}

func TestAuctionUsecaseCancelLotPersistsAndPublishesEvent(t *testing.T) {
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_cancel",
		Title:    "异常取消拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 0, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if _, err := uc.CancelLot(ctx, lot.Id, "op", "未开拍误操作"); err == nil || !strings.Contains(err.Error(), "only live lot") {
		t.Fatalf("draft cancel should be rejected, got %v", err)
	}
	if _, err := uc.StartLot(ctx, lot.Id); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	cancelled, err := uc.CancelLot(ctx, lot.Id, "op", "主播网络异常")
	if err != nil {
		t.Fatalf("cancel lot failed: %v", err)
	}
	if cancelled.Status != v1.LotStatus_LOT_STATUS_CANCELLED || cancelled.CancelReason != "主播网络异常" || cancelled.CancelledAtUnixMs == 0 {
		t.Fatalf("cancelled lot mismatch: %+v", cancelled)
	}
	fresh, err := store.FindByID(ctx, lot.Id)
	if err != nil {
		t.Fatalf("find cancelled lot failed: %v", err)
	}
	if fresh.Status != v1.LotStatus_LOT_STATUS_CANCELLED || fresh.CancelReason != "主播网络异常" {
		t.Fatalf("persisted cancelled lot mismatch: %+v", fresh)
	}
	pub.assertContains(t, v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CANCELLED)
	assertEventTypesContain(t, store.eventTypes(), v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CANCELLED)
}

func TestPlaceBidAntiSnipeExtensionPersistsLotUpdatedEvent(t *testing.T) {
	store := newTestStore()
	pub := &testPublisher{}
	uc := auction.NewAuctionUsecase(store, store, store, pub)
	ctx := context.Background()

	lot, err := uc.CreateLot(ctx, &v1.CreateLotRequest{
		RoomId:   "room_extend",
		Title:    "延时事件拍品",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        60,
			AntiSnipeWindowSeconds: 70,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	}, "test-owner")
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	started, err := uc.StartLot(ctx, lot.Id)
	if err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	originalEndsAt := started.EndsAtUnixMs
	updated, bid, ranking, err := uc.PlaceBid(ctx, &v1.PlaceBidRequest{LotId: lot.Id, Amount: &v1.Money{Amount: 11000, Currency: "CNY"}}, "u1", "用户1")
	if err != nil || bid == nil || len(ranking) != 1 {
		t.Fatalf("extension bid failed: updated=%+v bid=%+v ranking=%+v err=%v", updated, bid, ranking, err)
	}
	if updated.EndsAtUnixMs <= originalEndsAt || updated.GetDuelState().GetExtendCount() != 1 || updated.GetDuelState().GetLotId() != updated.Id || updated.GetDuelState().GetEndsAtUnixMs() != updated.EndsAtUnixMs {
		t.Fatalf("accepted bid should extend live lot and sync duel state: before=%d after=%+v", originalEndsAt, updated)
	}
	pub.assertContains(t, v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_UPDATED)
	assertEventTypesContain(t, store.eventTypes(), v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_UPDATED)
}

type testPublisher struct {
	mu     sync.Mutex
	events []v1.AuctionEvent
}

func assertEventTypesContain(t *testing.T, got []v1.AuctionEventType, want ...v1.AuctionEventType) {
	t.Helper()
	seen := make(map[v1.AuctionEventType]bool, len(got))
	for _, typ := range got {
		seen[typ] = true
	}
	for _, typ := range want {
		if !seen[typ] {
			t.Fatalf("missing persisted event type %s in %+v", typ, got)
		}
	}
}

func (p *testPublisher) Publish(ctx context.Context, event v1.AuctionEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) assertContains(t *testing.T, types ...v1.AuctionEventType) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	seen := make(map[v1.AuctionEventType]bool)
	for _, event := range p.events {
		seen[event.Type] = true
	}
	for _, typ := range types {
		if !seen[typ] {
			t.Fatalf("missing event type %s in %+v", typ, p.events)
		}
	}
}

func TestStartDuelWithOnlyUserBDoesNotDuplicateUsers(t *testing.T) {
	lot, err := auction.NewLotFromRequest("lot_duel", &v1.CreateLotRequest{
		RoomId:   "demo",
		Title:    "Duel 指定测试",
		ImageUrl: "https://example.com/lot.jpg",
		Rule: &v1.BidRule{
			StartPrice:             &v1.Money{Amount: 10000, Currency: "CNY"},
			MinIncrement:           &v1.Money{Amount: 1000, Currency: "CNY"},
			DurationSeconds:        300,
			AntiSnipeWindowSeconds: 15,
			AntiSnipeExtendSeconds: 15,
			MaxExtendCount:         3,
		},
	})
	if err != nil {
		t.Fatalf("create lot failed: %v", err)
	}
	if err := auction.StartLot(lot, 1000); err != nil {
		t.Fatalf("start lot failed: %v", err)
	}
	ranking := []*v1.RankingItem{
		{Rank: 1, UserId: "u1", Nickname: "用户1", Amount: &v1.Money{Amount: 13000, Currency: "CNY"}, BidAtUnixMs: 3000},
		{Rank: 2, UserId: "u2", Nickname: "用户2", Amount: &v1.Money{Amount: 12000, Currency: "CNY"}, BidAtUnixMs: 2000},
	}
	if err := auction.StartDuel(lot, ranking, 4000, "", "u1"); err != nil {
		t.Fatalf("start duel failed: %v", err)
	}
	if lot.GetDuelState().GetUserAId() != "u2" || lot.GetDuelState().GetUserBId() != "u1" {
		t.Fatalf("expected service to fill distinct user A around requested B, got %+v", lot.GetDuelState())
	}
}
