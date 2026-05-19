package biz

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

type CreateLotCommand struct {
	RoomID      string            `json:"roomId"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	ImageURL    string            `json:"imageUrl"`
	Rule        BidRule           `json:"rule"`
	TrustCards  []TrustRevealCard `json:"trustCards"`
}

type PlaceBidCommand struct {
	LotID              string `json:"lotId"`
	UserID             string `json:"userId"`
	Nickname           string `json:"nickname"`
	Amount             Money  `json:"amount"`
	ClientKnownVersion int64  `json:"clientKnownVersion"`
	IdempotencyKey     string `json:"idempotencyKey"`
}

type Service struct {
	mu          sync.Mutex
	repo        LotRepository
	pub         EventPublisher
	bidsByLot   map[string][]Bid
	idemByLot   map[string]map[string]Bid
}

func NewService(repo LotRepository, pub EventPublisher) *Service {
	return &Service{repo: repo, pub: pub, bidsByLot: map[string][]Bid{}, idemByLot: map[string]map[string]Bid{}}
}

func (s *Service) CreateLot(ctx context.Context, cmd CreateLotCommand) (*Lot, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	if cmd.RoomID == "" { cmd.RoomID = "demo" }
	if cmd.Rule.StartPrice.Currency == "" { cmd.Rule.StartPrice.Currency = "CNY" }
	if cmd.Rule.MinIncrement.Currency == "" { cmd.Rule.MinIncrement.Currency = "CNY" }
	if cmd.Rule.DurationSeconds == 0 { cmd.Rule.DurationSeconds = 300 }
	if cmd.Rule.AntiSnipeWindowSeconds == 0 { cmd.Rule.AntiSnipeWindowSeconds = 15 }
	if cmd.Rule.AntiSnipeExtendSeconds == 0 { cmd.Rule.AntiSnipeExtendSeconds = 15 }
	if cmd.Rule.MaxExtendCount == 0 { cmd.Rule.MaxExtendCount = 3 }
	lot := &Lot{ID: nextID("lot"), RoomID: cmd.RoomID, Title: cmd.Title, Description: cmd.Description, ImageURL: cmd.ImageURL, Status: LotStatusDraft, Rule: cmd.Rule, CurrentPrice: cmd.Rule.StartPrice, FinalPrice: CNY(0), Version: 1, TrustCards: cmd.TrustCards, PlaybookStage: PlaybookStageWarmUp}
	for i := range lot.TrustCards {
		if lot.TrustCards[i].ID == "" { lot.TrustCards[i].ID = nextID("card") }
		lot.TrustCards[i].LotID = lot.ID
		lot.TrustCards[i].Revealed = false
	}
	if err := s.repo.Create(ctx, lot); err != nil { return nil, err }
	s.publish(ctx, EventLotCreated, lot, nil, nil, nil, nil, "")
	return cloneLot(lot), nil
}

func (s *Service) GetLot(ctx context.Context, lotID string) (*Lot, error) { return s.repo.FindByID(ctx, lotID) }
func (s *Service) ListLots(ctx context.Context, roomID string, status LotStatus) ([]*Lot, error) { return s.repo.List(ctx, roomID, status) }

func (s *Service) StartLot(ctx context.Context, lotID string) (*Lot, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	lot, err := s.repo.FindByID(ctx, lotID); if err != nil { return nil, err }
	if lot.Status != LotStatusDraft { return nil, fmt.Errorf("只有草稿拍品可以开拍，当前状态：%s", lot.Status) }
	now := NowMs()
	lot.Status = LotStatusLive; lot.StartedAtUnixMs = now; lot.EndsAtUnixMs = now + int64(lot.Rule.DurationSeconds)*1000; lot.CurrentPrice = lot.Rule.StartPrice; lot.Version++; lot.PlaybookStage = PlaybookStageWarmUp
	if err := s.repo.Save(ctx, lot); err != nil { return nil, err }
	s.publish(ctx, EventLotStarted, lot, nil, s.rankingLocked(lot.ID), nil, nil, "")
	return cloneLot(lot), nil
}

func (s *Service) PlaceBid(ctx context.Context, cmd PlaceBidCommand) (*Lot, *Bid, []RankingItem, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	lot, err := s.repo.FindByID(ctx, cmd.LotID); if err != nil { return nil, nil, nil, err }
	if lot.Status != LotStatusLive { s.publish(ctx, EventBidRejected, lot, nil, nil, nil, nil, "拍品未处于竞拍中"); return cloneLot(lot), nil, s.rankingLocked(lot.ID), errors.New("拍品未处于竞拍中") }
	if lot.EndsAtUnixMs > 0 && NowMs() > lot.EndsAtUnixMs { s.publish(ctx, EventBidRejected, lot, nil, nil, nil, nil, "竞拍已超时"); return cloneLot(lot), nil, s.rankingLocked(lot.ID), errors.New("竞拍已超时") }
	if cmd.Amount.Amount < lot.CurrentPrice.Amount+lot.Rule.MinIncrement.Amount { reason := "出价低于当前价加最低加价幅度"; s.publish(ctx, EventBidRejected, lot, nil, nil, nil, nil, reason); return cloneLot(lot), nil, s.rankingLocked(lot.ID), errors.New(reason) }
	if cmd.UserID == "" { cmd.UserID = nextID("guest") }
	if cmd.Nickname == "" { cmd.Nickname = "游客" }
	if cmd.Amount.Currency == "" { cmd.Amount.Currency = "CNY" }
	if cmd.IdempotencyKey != "" {
		if s.idemByLot[lot.ID] == nil { s.idemByLot[lot.ID] = map[string]Bid{} }
		if old, ok := s.idemByLot[lot.ID][cmd.IdempotencyKey]; ok { ranking := s.rankingLocked(lot.ID); return cloneLot(lot), &old, ranking, nil }
	}
	bid := Bid{ID: nextID("bid"), LotID: lot.ID, UserID: cmd.UserID, Nickname: cmd.Nickname, Amount: cmd.Amount, CreatedAtUnixMs: NowMs()}
	s.bidsByLot[lot.ID] = append(s.bidsByLot[lot.ID], bid)
	if cmd.IdempotencyKey != "" { s.idemByLot[lot.ID][cmd.IdempotencyKey] = bid }
	lot.CurrentPrice = cmd.Amount; lot.LeadingUserID = cmd.UserID; lot.LeadingNickname = cmd.Nickname; lot.Version++; lot.PlaybookStage = PlaybookStageBiddingActive
	remaining := lot.EndsAtUnixMs - NowMs()
	if remaining > 0 && remaining <= int64(lot.Rule.AntiSnipeWindowSeconds)*1000 && lot.DuelState.ExtendCount < lot.Rule.MaxExtendCount {
		lot.EndsAtUnixMs += int64(lot.Rule.AntiSnipeExtendSeconds) * 1000
		lot.DuelState.ExtendCount++
	}
	ranking := s.rankingLocked(lot.ID)
	if !lot.DuelState.Active && shouldStartDuel(ranking, s.bidsByLot[lot.ID], lot) { s.startDuelLocked(lot, ranking) }
	if err := s.repo.Save(ctx, lot); err != nil { return nil, nil, nil, err }
	s.publish(ctx, EventBidAccepted, lot, &bid, ranking, nil, nil, "")
	s.publish(ctx, EventRankingUpdated, lot, nil, ranking, nil, nil, "")
	if lot.DuelState.Active { ds := lot.DuelState; s.publish(ctx, EventDuelStarted, lot, nil, ranking, nil, &ds, "") }
	return cloneLot(lot), &bid, ranking, nil
}

func (s *Service) RevealTrustCard(ctx context.Context, lotID, cardID, operatorID string) (*Lot, *TrustRevealCard, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	lot, err := s.repo.FindByID(ctx, lotID); if err != nil { return nil, nil, err }
	for i := range lot.TrustCards {
		if lot.TrustCards[i].ID == cardID {
			lot.TrustCards[i].Revealed = true; lot.TrustCards[i].RevealedAtUnixMs = NowMs(); lot.Version++; lot.PlaybookStage = PlaybookStageTrustBlocked
			card := lot.TrustCards[i]
			if err := s.repo.Save(ctx, lot); err != nil { return nil, nil, err }
			s.publish(ctx, EventTrustRevealed, lot, nil, s.rankingLocked(lot.ID), &card, nil, "")
			return cloneLot(lot), &card, nil
		}
	}
	return nil, nil, errors.New("信任卡片不存在")
}

func (s *Service) StartDuel(ctx context.Context, lotID, operatorID, a, b string) (*Lot, *DuelState, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	lot, err := s.repo.FindByID(ctx, lotID); if err != nil { return nil, nil, err }
	ranking := s.rankingLocked(lot.ID)
	if len(ranking) < 2 { return nil, nil, errors.New("至少需要两名出价用户才能进入 Duel Mode") }
	s.startDuelLocked(lot, ranking)
	if err := s.repo.Save(ctx, lot); err != nil { return nil, nil, err }
	ds := lot.DuelState
	s.publish(ctx, EventDuelStarted, lot, nil, ranking, nil, &ds, "")
	return cloneLot(lot), &ds, nil
}

func (s *Service) SettleLot(ctx context.Context, lotID, operatorID string) (*Lot, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	lot, err := s.repo.FindByID(ctx, lotID); if err != nil { return nil, err }
	if lot.Status != LotStatusLive { return nil, fmt.Errorf("只有竞拍中的拍品可以落锤，当前状态：%s", lot.Status) }
	lot.Status = LotStatusSettled; lot.SettledAtUnixMs = NowMs(); lot.WinnerUserID = lot.LeadingUserID; lot.WinnerNickname = lot.LeadingNickname; lot.FinalPrice = lot.CurrentPrice; lot.Version++; lot.PlaybookStage = PlaybookStageSettleReady
	if lot.DuelState.Active { lot.DuelState.Active = false }
	if err := s.repo.Save(ctx, lot); err != nil { return nil, err }
	s.publish(ctx, EventLotSettled, lot, nil, s.rankingLocked(lot.ID), nil, nil, "")
	return cloneLot(lot), nil
}

func (s *Service) Snapshot(ctx context.Context, roomID string) (*RoomSnapshot, error) {
	lots, err := s.repo.List(ctx, roomID, ""); if err != nil { return nil, err }
	var current *Lot
	for _, lot := range lots { if lot.Status == LotStatusLive { current = lot; break } }
	if current == nil && len(lots) > 0 { current = lots[len(lots)-1] }
	var ranking []RankingItem; var recent []Bid; stage := PlaybookStageWarmUp
	if current != nil { ranking = s.rankingLocked(current.ID); recent = s.recentBidsLocked(current.ID, 20); stage = current.PlaybookStage }
	return &RoomSnapshot{RoomID: roomID, CurrentLot: current, Ranking: ranking, RecentBids: recent, PlaybookStage: stage, ServerTimeUnixMs: NowMs()}, nil
}

func (s *Service) rankingLocked(lotID string) []RankingItem {
	best := map[string]RankingItem{}
	for _, b := range s.bidsByLot[lotID] {
		cur, ok := best[b.UserID]
		if !ok || b.Amount.Amount > cur.Amount.Amount || (b.Amount.Amount == cur.Amount.Amount && b.CreatedAtUnixMs < cur.BidAtUnixMs) { best[b.UserID] = RankingItem{UserID: b.UserID, Nickname: b.Nickname, Amount: b.Amount, BidAtUnixMs: b.CreatedAtUnixMs} }
	}
	items := make([]RankingItem, 0, len(best)); for _, v := range best { items = append(items, v) }
	sort.Slice(items, func(i, j int) bool { if items[i].Amount.Amount == items[j].Amount.Amount { return items[i].BidAtUnixMs < items[j].BidAtUnixMs }; return items[i].Amount.Amount > items[j].Amount.Amount })
	for i := range items { items[i].Rank = int32(i+1) }
	return items
}

func (s *Service) recentBidsLocked(lotID string, n int) []Bid { bids := s.bidsByLot[lotID]; if len(bids) <= n { return append([]Bid(nil), bids...) }; return append([]Bid(nil), bids[len(bids)-n:]...) }
func (s *Service) startDuelLocked(lot *Lot, ranking []RankingItem) { if len(ranking) < 2 { return }; now := NowMs(); lot.DuelState = DuelState{Active: true, LotID: lot.ID, UserAID: ranking[0].UserID, UserANickname: ranking[0].Nickname, UserBID: ranking[1].UserID, UserBNickname: ranking[1].Nickname, StartedAtUnixMs: now, EndsAtUnixMs: lot.EndsAtUnixMs, MaxExtendCount: lot.Rule.MaxExtendCount, ExtendCount: lot.DuelState.ExtendCount}; lot.PlaybookStage = PlaybookStageDuelMode; lot.Version++ }
func shouldStartDuel(ranking []RankingItem, bids []Bid, lot *Lot) bool { if len(ranking) < 2 || lot.EndsAtUnixMs-NowMs() > 60000 { return false }; if ranking[0].Amount.Amount-ranking[1].Amount.Amount > lot.Rule.MinIncrement.Amount*3 { return false }; return len(bids) >= 3 }
func (s *Service) publish(ctx context.Context, typ EventType, lot *Lot, bid *Bid, ranking []RankingItem, card *TrustRevealCard, duel *DuelState, reason string) { if s.pub == nil || lot == nil { return }; s.pub.Publish(ctx, AuctionEvent{ID: nextID("evt"), Type: typ, RoomID: lot.RoomID, LotID: lot.ID, OccurredAtUnixMs: NowMs(), Lot: cloneLot(lot), Bid: bid, Ranking: ranking, TrustCard: card, DuelState: duel, Reason: reason}) }
func nextID(prefix string) string { return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()) }
func cloneLot(l *Lot) *Lot { if l == nil { return nil }; cp := *l; cp.TrustCards = append([]TrustRevealCard(nil), l.TrustCards...); return &cp }
