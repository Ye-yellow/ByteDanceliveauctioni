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
		ID:            id,
		RoomID:        cmd.RoomID,
		Title:         cmd.Title,
		Description:   cmd.Description,
		ImageURL:      cmd.ImageURL,
		Status:        model.LotStatusDraft,
		Rule:          cmd.Rule,
		CurrentPrice:  cmd.Rule.StartPrice,
		FinalPrice:    model.CNY(0),
		Version:       1,
		TrustCards:    cmd.TrustCards,
		PlaybookStage: model.PlaybookStageWarmUp,
	}

	for i := range lot.TrustCards {
		if lot.TrustCards[i].ID == "" {
			lot.TrustCards[i].ID = idgen.New("card")
		}
		lot.TrustCards[i].LotID = lot.ID
		lot.TrustCards[i].Revealed = false
		lot.TrustCards[i].RevealedAtUnixMs = 0
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

func normalizeBidRule(rule model.BidRule) model.BidRule {
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

func StartLot(lot *model.Lot, nowMs int64) error {
	if lot.Status != model.LotStatusDraft {
		return fmt.Errorf("只有草稿拍品可以开拍，当前状态：%s", lot.Status)
	}
	lot.Status = model.LotStatusLive
	lot.StartedAtUnixMs = nowMs
	lot.EndsAtUnixMs = nowMs + int64(lot.Rule.DurationSeconds)*1000
	lot.CurrentPrice = lot.Rule.StartPrice
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
	if bid.Amount.Amount < NextBidAmount(lot) {
		return errors.New("出价低于当前价加最低加价幅度")
	}
	lot.CurrentPrice = bid.Amount
	lot.LeadingUserID = bid.UserID
	lot.LeadingNickname = bid.Nickname
	lot.PlaybookStage = model.PlaybookStageBiddingActive
	extendIfAntiSnipe(lot, nowMs)
	bumpVersion(lot)
	return nil
}

func NextBidAmount(lot *model.Lot) int64 {
	return lot.CurrentPrice.Amount + lot.Rule.MinIncrement.Amount
}

func RevealTrustCard(lot *model.Lot, cardID string, nowMs int64) (*model.TrustRevealCard, error) {
	for i := range lot.TrustCards {
		if lot.TrustCards[i].ID == cardID {
			lot.TrustCards[i].Revealed = true
			lot.TrustCards[i].RevealedAtUnixMs = nowMs
			lot.PlaybookStage = model.PlaybookStageTrustBlocked
			bumpVersion(lot)
			return &lot.TrustCards[i], nil
		}
	}
	return nil, errors.New("信任卡片不存在")
}

func StartDuel(lot *model.Lot, ranking []model.RankingItem, nowMs int64) error {
	if lot.Status != model.LotStatusLive {
		return errors.New("只有竞拍中的拍品可以进入 Duel Mode")
	}
	if len(ranking) < 2 {
		return errors.New("至少需要两名出价用户才能进入 Duel Mode")
	}
	lot.DuelState = model.DuelState{
		Active:          true,
		LotID:           lot.ID,
		UserAID:         ranking[0].UserID,
		UserANickname:   ranking[0].Nickname,
		UserBID:         ranking[1].UserID,
		UserBNickname:   ranking[1].Nickname,
		StartedAtUnixMs: nowMs,
		EndsAtUnixMs:    lot.EndsAtUnixMs,
		ExtendCount:     lot.DuelState.ExtendCount,
		MaxExtendCount:  lot.Rule.MaxExtendCount,
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
	lot.WinnerUserID = lot.LeadingUserID
	lot.WinnerNickname = lot.LeadingNickname
	lot.FinalPrice = lot.CurrentPrice
	lot.PlaybookStage = model.PlaybookStageSettleReady
	lot.DuelState.Active = false
	bumpVersion(lot)
	return nil
}

func extendIfAntiSnipe(lot *model.Lot, nowMs int64) {
	remainingMs := lot.EndsAtUnixMs - nowMs
	windowMs := int64(lot.Rule.AntiSnipeWindowSeconds) * 1000
	if remainingMs <= 0 || remainingMs > windowMs {
		return
	}
	if lot.DuelState.ExtendCount >= lot.Rule.MaxExtendCount {
		return
	}
	lot.EndsAtUnixMs += int64(lot.Rule.AntiSnipeExtendSeconds) * 1000
	lot.DuelState.ExtendCount++
}

func bumpVersion(lot *model.Lot) {
	lot.Version++
}
