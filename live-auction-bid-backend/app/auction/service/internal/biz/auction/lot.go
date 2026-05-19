package auction

import (
	"errors"
	"fmt"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func NewLotFromRequest(id string, req *v1.CreateLotRequest) *v1.Lot {
	req = normalizeCreateLotRequest(req)
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
		TrustCards:    cloneTrustCards(req.GetTrustCards()),
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

func normalizeCreateLotRequest(req *v1.CreateLotRequest) *v1.CreateLotRequest {
	if req == nil {
		req = &v1.CreateLotRequest{}
	}
	if req.RoomId == "" {
		req.RoomId = "demo"
	}
	req.Rule = normalizeBidRule(req.Rule)
	return req
}

func normalizeBidRule(rule *v1.BidRule) *v1.BidRule {
	if rule == nil {
		rule = &v1.BidRule{}
	}
	if rule.StartPrice == nil {
		rule.StartPrice = CNY(0)
	}
	if rule.StartPrice.Currency == "" {
		rule.StartPrice.Currency = "CNY"
	}
	if rule.MinIncrement == nil {
		rule.MinIncrement = CNY(0)
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
	if bid.GetAmount().GetAmount() < NextBidAmount(lot) {
		return errors.New("出价低于当前价加最低加价幅度")
	}
	lot.CurrentPrice = cloneMoney(bid.GetAmount())
	lot.LeadingUserId = bid.UserId
	lot.LeadingNickname = bid.Nickname
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_BIDDING_ACTIVE
	extendIfAntiSnipe(lot, nowMs)
	bumpVersion(lot)
	return nil
}

func NextBidAmount(lot *v1.Lot) int64 {
	return lot.GetCurrentPrice().GetAmount() + lot.GetRule().GetMinIncrement().GetAmount()
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

func extendIfAntiSnipe(lot *v1.Lot, nowMs int64) {
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
		lot.DuelState = &v1.DuelState{}
	}
	lot.DuelState.ExtendCount++
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

func cloneTrustCards(cards []*v1.TrustRevealCard) []*v1.TrustRevealCard {
	out := make([]*v1.TrustRevealCard, 0, len(cards))
	for _, card := range cards {
		if card == nil {
			continue
		}
		cp := *card
		out = append(out, &cp)
	}
	return out
}
