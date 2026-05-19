package auction

import (
	"errors"
	"fmt"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func NewLotFromRequest(id string, req *v1.CreateLotRequest) (*v1.Lot, error) {
	if req == nil {
		return nil, errors.New("create lot request is required")
	}
	if req.GetRoomId() == "" {
		return nil, errors.New("room id is required")
	}
	if req.GetTitle() == "" {
		return nil, errors.New("lot title is required")
	}
	if req.GetRule() == nil || req.GetRule().GetStartPrice() == nil || req.GetRule().GetMinIncrement() == nil {
		return nil, errors.New("bid rule, start price and min increment are required")
	}
	if req.GetRule().GetStartPrice().GetCurrency() == "" || req.GetRule().GetMinIncrement().GetCurrency() == "" {
		return nil, errors.New("start price and min increment currency are required")
	}
	if req.GetRule().GetDurationSeconds() <= 0 || req.GetRule().GetAntiSnipeWindowSeconds() <= 0 ||
		req.GetRule().GetAntiSnipeExtendSeconds() <= 0 || req.GetRule().GetMaxExtendCount() <= 0 {
		return nil, errors.New("duration, anti-snipe window, anti-snipe extension and max extension count must be greater than zero")
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
		CurrentPrice:  req.GetRule().GetStartPrice(),
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
	return lot, nil
}

func StartLot(lot *v1.Lot, nowMs int64) error {
	if lot.Status != v1.LotStatus_LOT_STATUS_DRAFT {
		return fmt.Errorf("only draft lot can be started, current status: %s", lot.Status)
	}
	lot.Status = v1.LotStatus_LOT_STATUS_LIVE
	lot.StartedAtUnixMs = nowMs
	lot.EndsAtUnixMs = nowMs + int64(lot.GetRule().GetDurationSeconds())*1000
	lot.CurrentPrice = lot.GetRule().GetStartPrice()
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_WARM_UP
	lot.Version++
	return nil
}

func AcceptBid(lot *v1.Lot, bid v1.Bid, nowMs int64) error {
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		return errors.New("lot is not live")
	}
	if lot.EndsAtUnixMs > 0 && nowMs > lot.EndsAtUnixMs {
		return errors.New("auction has ended")
	}
	if bid.GetAmount().GetAmount() < lot.GetCurrentPrice().GetAmount()+lot.GetRule().GetMinIncrement().GetAmount() {
		return errors.New("bid amount is lower than current price plus min increment")
	}
	lot.CurrentPrice = bid.GetAmount()
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

	lot.Version++
	return nil
}

func RevealTrustCard(lot *v1.Lot, cardID string, nowMs int64) (*v1.TrustRevealCard, error) {
	for _, card := range lot.TrustCards {
		if card.Id == cardID {
			card.Revealed = true
			card.RevealedAtUnixMs = nowMs
			lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_TRUST_BLOCKED
			lot.Version++
			return card, nil
		}
	}
	return nil, errors.New("trust card not found")
}

func StartDuel(lot *v1.Lot, ranking []*v1.RankingItem, nowMs int64) error {
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		return errors.New("only live lot can enter duel mode")
	}
	if len(ranking) < 2 {
		return errors.New("at least two bidders are required to enter duel mode")
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
	lot.Version++
	return nil
}

func SettleLot(lot *v1.Lot, nowMs int64) error {
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		return fmt.Errorf("only live lot can be settled, current status: %s", lot.Status)
	}
	lot.Status = v1.LotStatus_LOT_STATUS_SETTLED
	lot.SettledAtUnixMs = nowMs
	lot.WinnerUserId = lot.LeadingUserId
	lot.WinnerNickname = lot.LeadingNickname
	lot.FinalPrice = lot.GetCurrentPrice()
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_SETTLE_READY
	if lot.DuelState != nil {
		lot.DuelState.Active = false
	}
	lot.Version++
	return nil
}
