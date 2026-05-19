package auction

import (
	"errors"
	"fmt"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func NewLotFromRequest(id string, req *v1.CreateLotRequest) *v1.Lot {
	if req == nil {
		req = &v1.CreateLotRequest{}
	}
	if req.RoomId == "" {
		req.RoomId = "demo"
	}
	if req.Rule == nil {
		req.Rule = &v1.BidRule{}
	}
	if req.Rule.StartPrice == nil {
		req.Rule.StartPrice = CNY(0)
	}
	if req.Rule.StartPrice.Currency == "" {
		req.Rule.StartPrice.Currency = "CNY"
	}
	if req.Rule.MinIncrement == nil {
		req.Rule.MinIncrement = CNY(0)
	}
	if req.Rule.MinIncrement.Currency == "" {
		req.Rule.MinIncrement.Currency = "CNY"
	}
	if req.Rule.DurationSeconds == 0 {
		req.Rule.DurationSeconds = 300
	}
	if req.Rule.AntiSnipeWindowSeconds == 0 {
		req.Rule.AntiSnipeWindowSeconds = 15
	}
	if req.Rule.AntiSnipeExtendSeconds == 0 {
		req.Rule.AntiSnipeExtendSeconds = 15
	}
	if req.Rule.MaxExtendCount == 0 {
		req.Rule.MaxExtendCount = 3
	}

	trustCards := make([]*v1.TrustRevealCard, 0, len(req.GetTrustCards()))
	for _, card := range req.GetTrustCards() {
		if card == nil {
			continue
		}
		cp := *card
		trustCards = append(trustCards, &cp)
	}

	lot := &v1.Lot{
		Id:            id,
		RoomId:        req.GetRoomId(),
		Title:         req.GetTitle(),
		Description:   req.GetDescription(),
		ImageUrl:      req.GetImageUrl(),
		Status:        v1.LotStatus_LOT_STATUS_DRAFT,
		Rule:          req.GetRule(),
		CurrentPrice:  cloneMoney(req.GetRule().GetStartPrice()),
		FinalPrice:    CNY(0),
		Version:       1,
		TrustCards:    trustCards,
		DuelState:     &v1.DuelState{},
		PlaybookStage: v1.PlaybookStage_PLAYBOOK_STAGE_WARM_UP,
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

func StartLot(lot *v1.Lot, nowMs int64) error {
	if lot.Status != v1.LotStatus_LOT_STATUS_DRAFT {
		return fmt.Errorf("只有草稿拍品可以开拍，当前状态：%s", lot.Status)
	}
	lot.Status = v1.LotStatus_LOT_STATUS_LIVE
	lot.StartedAtUnixMs = nowMs
	lot.EndsAtUnixMs = nowMs + int64(lot.GetRule().GetDurationSeconds())*1000
	lot.CurrentPrice = cloneMoney(lot.GetRule().GetStartPrice())
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_WARM_UP
	bumpVersion(lot)
	return nil
}

func AcceptBid(lot *v1.Lot, bid v1.Bid, nowMs int64) error {
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		return errors.New("拍品未处于竞拍中")
	}
	if lot.EndsAtUnixMs > 0 && nowMs > lot.EndsAtUnixMs {
		return errors.New("竞拍已超时")
	}
	if bid.GetAmount().GetAmount() < lot.GetCurrentPrice().GetAmount()+lot.GetRule().GetMinIncrement().GetAmount() {
		return errors.New("出价低于当前价加最低加价幅度")
	}
	lot.CurrentPrice = cloneMoney(bid.GetAmount())
	lot.LeadingUserId = bid.UserId
	lot.LeadingNickname = bid.Nickname
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_BIDDING_ACTIVE

	remainingMs := lot.EndsAtUnixMs - nowMs
	windowMs := int64(lot.GetRule().GetAntiSnipeWindowSeconds()) * 1000
	if remainingMs > 0 && remainingMs <= windowMs && lot.GetDuelState().GetExtendCount() < lot.GetRule().GetMaxExtendCount() {
		lot.EndsAtUnixMs += int64(lot.GetRule().GetAntiSnipeExtendSeconds()) * 1000
		if lot.DuelState == nil {
			lot.DuelState = &v1.DuelState{}
		}
		lot.DuelState.ExtendCount++
	}

	bumpVersion(lot)
	return nil
}

func RevealTrustCard(lot *v1.Lot, cardID string, nowMs int64) (*v1.TrustRevealCard, error) {
	for _, card := range lot.TrustCards {
		if card.Id == cardID {
			card.Revealed = true
			card.RevealedAtUnixMs = nowMs
			lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_TRUST_BLOCKED
			bumpVersion(lot)
			return card, nil
		}
	}
	return nil, errors.New("信任卡片不存在")
}

func StartDuel(lot *v1.Lot, ranking []*v1.RankingItem, nowMs int64) error {
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		return errors.New("只有竞拍中的拍品可以进入 Duel Mode")
	}
	if len(ranking) < 2 {
		return errors.New("至少需要两名出价用户才能进入 Duel Mode")
	}
	extendCount := int32(0)
	if lot.DuelState != nil {
		extendCount = lot.DuelState.ExtendCount
	}
	lot.DuelState = &v1.DuelState{
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
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_DUEL_MODE
	bumpVersion(lot)
	return nil
}

func SettleLot(lot *v1.Lot, nowMs int64) error {
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		return fmt.Errorf("只有竞拍中的拍品可以落锤，当前状态：%s", lot.Status)
	}
	lot.Status = v1.LotStatus_LOT_STATUS_SETTLED
	lot.SettledAtUnixMs = nowMs
	lot.WinnerUserId = lot.LeadingUserId
	lot.WinnerNickname = lot.LeadingNickname
	lot.FinalPrice = cloneMoney(lot.GetCurrentPrice())
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_SETTLE_READY
	if lot.DuelState != nil {
		lot.DuelState.Active = false
	}
	bumpVersion(lot)
	return nil
}

func bumpVersion(lot *v1.Lot) {
	lot.Version++
}

func cloneMoney(m *v1.Money) *v1.Money {
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}
