package auction

import (
	"errors"
	"fmt"
)

func NewLotFromCommand(id string, cmd CreateLotCommand) *Lot {
	cmd.normalize()
	lot := &Lot{
		ID:            id,
		RoomID:        cmd.RoomID,
		Title:         cmd.Title,
		Description:   cmd.Description,
		ImageURL:      cmd.ImageURL,
		Status:        LotStatusDraft,
		Rule:          cmd.Rule,
		CurrentPrice:  cmd.Rule.StartPrice,
		FinalPrice:    CNY(0),
		Version:       1,
		TrustCards:    cmd.TrustCards,
		PlaybookStage: PlaybookStageWarmUp,
	}

	for i := range lot.TrustCards {
		if lot.TrustCards[i].ID == "" {
			lot.TrustCards[i].ID = NewID("card")
		}
		lot.TrustCards[i].LotID = lot.ID
		lot.TrustCards[i].Revealed = false
		lot.TrustCards[i].RevealedAtUnixMs = 0
	}

	return lot
}

func (cmd *CreateLotCommand) normalize() {
	if cmd.RoomID == "" {
		cmd.RoomID = "demo"
	}
	cmd.Rule = normalizeBidRule(cmd.Rule)
}

func normalizeBidRule(rule BidRule) BidRule {
	if rule.StartPrice.Currency == "" {
		rule.StartPrice.Currency = "CNY"
	}
	if rule.MinIncrement.Currency == "" {
		rule.MinIncrement.Currency = "CNY"
	}
	if rule.DurationSeconds == 0 {
		rule.DurationSeconds = 300
	}
	if rule.AntiSnipeWindowSeconds == 0 {
		rule.AntiSnipeWindowSeconds = 15
	}
	if rule.AntiSnipeExtendSeconds == 0 {
		rule.AntiSnipeExtendSeconds = 15
	}
	if rule.MaxExtendCount == 0 {
		rule.MaxExtendCount = 3
	}
	return rule
}

func (l *Lot) Start(nowMs int64) error {
	if l.Status != LotStatusDraft {
		return fmt.Errorf("只有草稿拍品可以开拍，当前状态：%s", l.Status)
	}
	l.Status = LotStatusLive
	l.StartedAtUnixMs = nowMs
	l.EndsAtUnixMs = nowMs + int64(l.Rule.DurationSeconds)*1000
	l.CurrentPrice = l.Rule.StartPrice
	l.PlaybookStage = PlaybookStageWarmUp
	l.bumpVersion()
	return nil
}

func (l *Lot) AcceptBid(bid Bid, nowMs int64) error {
	if l.Status != LotStatusLive {
		return errors.New("拍品未处于竞拍中")
	}
	if l.EndsAtUnixMs > 0 && nowMs > l.EndsAtUnixMs {
		return errors.New("竞拍已超时")
	}
	if bid.Amount.Amount < l.NextBidAmount() {
		return errors.New("出价低于当前价加最低加价幅度")
	}

	l.CurrentPrice = bid.Amount
	l.LeadingUserID = bid.UserID
	l.LeadingNickname = bid.Nickname
	l.PlaybookStage = PlaybookStageBiddingActive
	l.extendIfAntiSnipe(nowMs)
	l.bumpVersion()
	return nil
}

func (l *Lot) NextBidAmount() int64 {
	return l.CurrentPrice.Amount + l.Rule.MinIncrement.Amount
}

func (l *Lot) RevealTrustCard(cardID string, nowMs int64) (*TrustRevealCard, error) {
	for i := range l.TrustCards {
		if l.TrustCards[i].ID == cardID {
			l.TrustCards[i].Revealed = true
			l.TrustCards[i].RevealedAtUnixMs = nowMs
			l.PlaybookStage = PlaybookStageTrustBlocked
			l.bumpVersion()
			return &l.TrustCards[i], nil
		}
	}
	return nil, errors.New("信任卡片不存在")
}

func (l *Lot) StartDuel(ranking []RankingItem, nowMs int64) error {
	if l.Status != LotStatusLive {
		return errors.New("只有竞拍中的拍品可以进入 Duel Mode")
	}
	if len(ranking) < 2 {
		return errors.New("至少需要两名出价用户才能进入 Duel Mode")
	}
	l.DuelState = DuelState{
		Active:          true,
		LotID:           l.ID,
		UserAID:         ranking[0].UserID,
		UserANickname:   ranking[0].Nickname,
		UserBID:         ranking[1].UserID,
		UserBNickname:   ranking[1].Nickname,
		StartedAtUnixMs: nowMs,
		EndsAtUnixMs:    l.EndsAtUnixMs,
		ExtendCount:     l.DuelState.ExtendCount,
		MaxExtendCount:  l.Rule.MaxExtendCount,
	}
	l.PlaybookStage = PlaybookStageDuelMode
	l.bumpVersion()
	return nil
}

func (l *Lot) Settle(nowMs int64) error {
	if l.Status != LotStatusLive {
		return fmt.Errorf("只有竞拍中的拍品可以落锤，当前状态：%s", l.Status)
	}
	l.Status = LotStatusSettled
	l.SettledAtUnixMs = nowMs
	l.WinnerUserID = l.LeadingUserID
	l.WinnerNickname = l.LeadingNickname
	l.FinalPrice = l.CurrentPrice
	l.PlaybookStage = PlaybookStageSettleReady
	l.DuelState.Active = false
	l.bumpVersion()
	return nil
}

func (l *Lot) extendIfAntiSnipe(nowMs int64) {
	remainingMs := l.EndsAtUnixMs - nowMs
	windowMs := int64(l.Rule.AntiSnipeWindowSeconds) * 1000
	if remainingMs <= 0 || remainingMs > windowMs {
		return
	}
	if l.DuelState.ExtendCount >= l.Rule.MaxExtendCount {
		return
	}
	l.EndsAtUnixMs += int64(l.Rule.AntiSnipeExtendSeconds) * 1000
	l.DuelState.ExtendCount++
}

func (l *Lot) bumpVersion() {
	l.Version++
}

func CloneLot(l *Lot) *Lot {
	if l == nil {
		return nil
	}
	cp := *l
	cp.TrustCards = append([]TrustRevealCard(nil), l.TrustCards...)
	return &cp
}
