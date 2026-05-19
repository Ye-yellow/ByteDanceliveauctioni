package auction

import (
	"errors"
	"fmt"

	"live-auction-bid/backend/app/auction/service/internal/model"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func NewLotFromCommand(id string, cmd model.CreateLotCommand) *model.Lot {
	cmd = normalizeCreateLotCommand(cmd)
	lot := &model.Lot{
		Id:            id,
		RoomId:        cmd.RoomID,
		Title:         cmd.Title,
		Description:   cmd.Description,
		ImageUrl:      cmd.ImageURL,
		Status:        model.LotStatusDraft,
		Rule:          cmd.Rule,
		CurrentPrice:  cloneMoney(cmd.Rule.GetStartPrice()),
		FinalPrice:    model.CNY(0),
		Version:       1,
		TrustCards:    cloneTrustCards(cmd.TrustCards),
		DuelState:     &model.DuelState{},
		PlaybookStage: model.PlaybookStageWarmUp,
	}

	for _, card := range lot.TrustCards {
		if card.Id == "" {
			card.Id = idgen.New("card")
		}
		card.LotId = lot.Id
		card.Revealed = false
		card.RevealedAtUnixMs = 0
	}
	return lot
}

func normalizeCreateLotCommand(cmd model.CreateLotCommand) model.CreateLotCommand {
	if cmd.RoomID == "" {
		cmd.RoomID = "demo"
	}
	cmd.Rule = normalizeBidRule(cmd.Rule)
	return cmd
}

func normalizeBidRule(rule *model.BidRule) *model.BidRule {
	if rule == nil {
		rule = &model.BidRule{}
	}
	if rule.StartPrice == nil {
		rule.StartPrice = model.CNY(0)
	}
	if rule.StartPrice.Currency == "" {
		rule.StartPrice.Currency = "CNY"
	}
	if rule.MinIncrement == nil {
		rule.MinIncrement = model.CNY(0)
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

func StartLot(lot *model.Lot, nowMs int64) error {
	if lot.Status != model.LotStatusDraft {
		return fmt.Errorf("只有草稿拍品可以开拍，当前状态：%s", lot.Status)
	}
	lot.Status = model.LotStatusLive
	lot.StartedAtUnixMs = nowMs
	lot.EndsAtUnixMs = nowMs + int64(lot.GetRule().GetDurationSeconds())*1000
	lot.CurrentPrice = cloneMoney(lot.GetRule().GetStartPrice())
	lot.PlaybookStage = model.PlaybookStageWarmUp
	bumpVersion(lot)
	return nil
}

func AcceptBid(lot *model.Lot, bid model.Bid, nowMs int64) error {
	if lot.Status != model.LotStatusLive {
		return errors.New("拍品未处于竞拍中")
	}
	if lot.EndsAtUnixMs > 0 && nowMs > lot.EndsAtUnixMs {
		return errors.New("竞拍已超时")
	}
	if bid.GetAmount().GetAmount() < NextBidAmount(lot) {
		return errors.New("出价低于当前价加最低加价幅度")
	}
	lot.CurrentPrice = cloneMoney(bid.GetAmount())
	lot.LeadingUserId = bid.UserId
	lot.LeadingNickname = bid.Nickname
	lot.PlaybookStage = model.PlaybookStageBiddingActive
	extendIfAntiSnipe(lot, nowMs)
	bumpVersion(lot)
	return nil
}

func NextBidAmount(lot *model.Lot) int64 {
	return lot.GetCurrentPrice().GetAmount() + lot.GetRule().GetMinIncrement().GetAmount()
}

func RevealTrustCard(lot *model.Lot, cardID string, nowMs int64) (*model.TrustRevealCard, error) {
	for _, card := range lot.TrustCards {
		if card.Id == cardID {
			card.Revealed = true
			card.RevealedAtUnixMs = nowMs
			lot.PlaybookStage = model.PlaybookStageTrustBlocked
			bumpVersion(lot)
			return card, nil
		}
	}
	return nil, errors.New("信任卡片不存在")
}

func StartDuel(lot *model.Lot, ranking []*model.RankingItem, nowMs int64) error {
	if lot.Status != model.LotStatusLive {
		return errors.New("只有竞拍中的拍品可以进入 Duel Mode")
	}
	if len(ranking) < 2 {
		return errors.New("至少需要两名出价用户才能进入 Duel Mode")
	}
	extendCount := int32(0)
	if lot.DuelState != nil {
		extendCount = lot.DuelState.ExtendCount
	}
	lot.DuelState = &model.DuelState{
		Active:          true,
		LotId:           lot.Id,
		UserAId:         ranking[0].UserId,
		UserANickname:   ranking[0].Nickname,
		UserBId:         ranking[1].UserId,
		UserBNickname:   ranking[1].Nickname,
		StartedAtUnixMs: nowMs,
		EndsAtUnixMs:    lot.EndsAtUnixMs,
		ExtendCount:     extendCount,
		MaxExtendCount:  lot.GetRule().GetMaxExtendCount(),
	}
	lot.PlaybookStage = model.PlaybookStageDuelMode
	bumpVersion(lot)
	return nil
}

func SettleLot(lot *model.Lot, nowMs int64) error {
	if lot.Status != model.LotStatusLive {
		return fmt.Errorf("只有竞拍中的拍品可以落锤，当前状态：%s", lot.Status)
	}
	lot.Status = model.LotStatusSettled
	lot.SettledAtUnixMs = nowMs
	lot.WinnerUserId = lot.LeadingUserId
	lot.WinnerNickname = lot.LeadingNickname
	lot.FinalPrice = cloneMoney(lot.GetCurrentPrice())
	lot.PlaybookStage = model.PlaybookStageSettleReady
	if lot.DuelState != nil {
		lot.DuelState.Active = false
	}
	bumpVersion(lot)
	return nil
}

func extendIfAntiSnipe(lot *model.Lot, nowMs int64) {
	remainingMs := lot.EndsAtUnixMs - nowMs
	windowMs := int64(lot.GetRule().GetAntiSnipeWindowSeconds()) * 1000
	if remainingMs <= 0 || remainingMs > windowMs {
		return
	}
	if lot.GetDuelState().GetExtendCount() >= lot.GetRule().GetMaxExtendCount() {
		return
	}
	lot.EndsAtUnixMs += int64(lot.GetRule().GetAntiSnipeExtendSeconds()) * 1000
	if lot.DuelState == nil {
		lot.DuelState = &model.DuelState{}
	}
	lot.DuelState.ExtendCount++
}

func bumpVersion(lot *model.Lot) {
	lot.Version++
}

func cloneMoney(m *model.Money) *model.Money {
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}

func cloneTrustCards(cards []*model.TrustRevealCard) []*model.TrustRevealCard {
	out := make([]*model.TrustRevealCard, 0, len(cards))
	for _, card := range cards {
		if card == nil {
			continue
		}
		cp := *card
		out = append(out, &cp)
	}
	return out
}
