package auction

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func NewLotFromRequest(id string, req *v1.CreateLotRequest) (*v1.Lot, error) {
	return NewLotDraftFromRequest(id, req, true)
}

func NewLotDraftFromRequest(id string, req *v1.CreateLotRequest, requireComplete bool) (*v1.Lot, error) {
	if req == nil {
		return nil, errors.New("create lot request is required")
	}
	if requireComplete && req.GetRoomId() == "" {
		return nil, errors.New("room id is required")
	}
	if requireComplete {
		if err := ValidateLotReady(req.GetTitle(), req.GetImageUrl(), req.GetRule()); err != nil {
			return nil, err
		}
	}

	rule := cloneRule(req.GetRule())
	if rule == nil {
		rule = &v1.BidRule{}
	}
	currentPrice := rule.GetStartPrice()
	if currentPrice == nil {
		currentPrice = &v1.Money{}
	}

	trustCards := cloneTrustCards(req.GetTrustCards())
	lot := &v1.Lot{
		Id:            id,
		RoomId:        req.GetRoomId(),
		Title:         req.GetTitle(),
		Description:   req.GetDescription(),
		ImageUrl:      req.GetImageUrl(),
		Status:        v1.LotStatus_LOT_STATUS_DRAFT,
		QueueStatus:   v1.LotQueueStatus_LOT_QUEUE_STATUS_NONE,
		Rule:          rule,
		CurrentPrice:  currentPrice,
		FinalPrice:    &v1.Money{Currency: currentPrice.GetCurrency()},
		Version:       1,
		TrustCards:    trustCards,
		DuelState:     &v1.DuelState{},
		PlaybookStage: v1.PlaybookStage_PLAYBOOK_STAGE_WARM_UP,
	}
	normalizeTrustCards(lot)
	return lot, nil
}

func ValidateLotReady(title, imageURL string, rule *v1.BidRule) error {
	if title == "" {
		return errors.New("lot title is required")
	}
	if imageURL == "" {
		return errors.New("lot image url is required")
	}
	if rule == nil || rule.GetStartPrice() == nil || rule.GetMinIncrement() == nil {
		return errors.New("bid rule, start price and min increment are required")
	}
	if rule.GetStartPrice().GetCurrency() == "" || rule.GetMinIncrement().GetCurrency() == "" {
		return errors.New("start price and min increment currency are required")
	}
	if rule.GetStartPrice().GetCurrency() != rule.GetMinIncrement().GetCurrency() {
		return errors.New("start price and min increment currency must match")
	}
	if rule.GetStartPrice().GetAmount() < 0 {
		return errors.New("start price amount must be >= 0")
	}
	if rule.GetMinIncrement().GetAmount() <= 0 {
		return errors.New("min increment amount must be > 0")
	}
	if rule.GetDurationSeconds() < 60 {
		return errors.New("duration seconds must be >= 60")
	}
	if rule.GetAntiSnipeWindowSeconds() <= 0 {
		return errors.New("anti-snipe window seconds must be > 0")
	}
	if rule.GetAntiSnipeExtendSeconds() < 10 || rule.GetAntiSnipeExtendSeconds() > 30 {
		return errors.New("anti-snipe extend seconds must be between 10 and 30")
	}
	if rule.GetMaxExtendCount() <= 0 {
		return errors.New("max extend count must be > 0")
	}
	if capPrice := rule.GetCapPrice(); capPrice != nil {
		if capPrice.GetCurrency() == "" {
			return errors.New("cap price currency is required")
		}
		if capPrice.GetCurrency() != rule.GetStartPrice().GetCurrency() || capPrice.GetCurrency() != rule.GetMinIncrement().GetCurrency() {
			return errors.New("cap price currency must match start price and min increment currency")
		}
		if capPrice.GetAmount() <= rule.GetStartPrice().GetAmount() {
			return errors.New("cap price amount must be greater than start price amount")
		}
	}
	return nil
}

func ApplyDraftPatch(lot *v1.Lot, req *v1.PatchLotDraftRequest) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	if req == nil {
		return errors.New("patch lot draft request is required")
	}
	if lot.Status == v1.LotStatus_LOT_STATUS_LIVE || lot.Status == v1.LotStatus_LOT_STATUS_SETTLED || lot.Status == v1.LotStatus_LOT_STATUS_CANCELLED {
		return fmt.Errorf("only not-started lot can be edited, current status: %s", lot.Status)
	}
	if lot.QueueStatus == v1.LotQueueStatus_LOT_QUEUE_STATUS_NEXT {
		return errors.New("next lot cannot be edited from add-lot draft page")
	}
	if req.GetRoomId() != "" {
		lot.RoomId = req.GetRoomId()
	}
	if req.GetTitle() != "" {
		lot.Title = req.GetTitle()
	}
	if req.GetDescription() != "" {
		lot.Description = req.GetDescription()
	}
	if req.GetImageUrl() != "" {
		lot.ImageUrl = req.GetImageUrl()
	}
	if req.GetRule() != nil {
		lot.Rule = cloneRule(req.GetRule())
		if lot.Rule.GetStartPrice() != nil {
			lot.CurrentPrice = lot.Rule.GetStartPrice()
			lot.FinalPrice = &v1.Money{Currency: lot.Rule.GetStartPrice().GetCurrency()}
		}
	}
	if len(req.GetTrustCards()) > 0 {
		lot.TrustCards = cloneTrustCards(req.GetTrustCards())
		normalizeTrustCards(lot)
	}
	if lot.QueueStatus == v1.LotQueueStatus_LOT_QUEUE_STATUS_UNSPECIFIED {
		lot.QueueStatus = v1.LotQueueStatus_LOT_QUEUE_STATUS_NONE
	}
	lot.Version++
	return nil
}

func QueueLot(lot *v1.Lot, queuePosition int32) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	if lot.Status == v1.LotStatus_LOT_STATUS_LIVE || lot.Status == v1.LotStatus_LOT_STATUS_SETTLED || lot.Status == v1.LotStatus_LOT_STATUS_CANCELLED {
		return fmt.Errorf("only not-started lot can be queued, current status: %s", lot.Status)
	}
	if err := ValidateLotReady(lot.GetTitle(), lot.GetImageUrl(), lot.GetRule()); err != nil {
		return err
	}
	if lot.QueueStatus == v1.LotQueueStatus_LOT_QUEUE_STATUS_QUEUED && lot.QueuePosition > 0 {
		return nil
	}
	if queuePosition <= 0 {
		return errors.New("queue position is required")
	}
	lot.Status = v1.LotStatus_LOT_STATUS_QUEUED
	lot.QueueStatus = v1.LotQueueStatus_LOT_QUEUE_STATUS_QUEUED
	lot.QueuePosition = queuePosition
	lot.CurrentPrice = lot.GetRule().GetStartPrice()
	lot.FinalPrice = &v1.Money{Currency: lot.GetRule().GetStartPrice().GetCurrency()}
	lot.Version++
	return nil
}

func cloneRule(rule *v1.BidRule) *v1.BidRule {
	if rule == nil {
		return nil
	}
	return proto.Clone(rule).(*v1.BidRule)
}

func cloneTrustCards(cards []*v1.TrustRevealCard) []*v1.TrustRevealCard {
	out := make([]*v1.TrustRevealCard, 0, len(cards))
	for _, card := range cards {
		if card == nil {
			continue
		}
		out = append(out, proto.Clone(card).(*v1.TrustRevealCard))
	}
	return out
}

func normalizeTrustCards(lot *v1.Lot) {
	for _, card := range lot.TrustCards {
		if card.Id == "" {
			card.Id = idgen.New("card")
		}
		card.LotId = lot.Id
		card.Revealed = false
		card.RevealedAtUnixMs = 0
	}
}

func StartLot(lot *v1.Lot, nowMs int64) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	if lot.GetRule() == nil || lot.GetRule().GetStartPrice() == nil {
		return errors.New("lot bid rule and start price are required")
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_DRAFT && lot.Status != v1.LotStatus_LOT_STATUS_QUEUED {
		return fmt.Errorf("only draft or queued lot can be started, current status: %s", lot.Status)
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
	if lot == nil {
		return errors.New("lot is required")
	}
	if lot.GetRule() == nil || lot.GetRule().GetMinIncrement() == nil || lot.GetCurrentPrice() == nil {
		return errors.New("lot rule, min increment and current price are required")
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		return errors.New("lot is not live")
	}
	if lot.EndsAtUnixMs > 0 && nowMs > lot.EndsAtUnixMs {
		return errors.New("auction has ended")
	}
	if bid.GetAmount() == nil || bid.GetAmount().GetCurrency() == "" {
		return errors.New("bid amount and currency are required")
	}
	if bid.GetUserId() == "" || bid.GetNickname() == "" {
		return errors.New("bid user id and nickname are required")
	}
	if bid.GetAmount().GetCurrency() != lot.GetCurrentPrice().GetCurrency() {
		return errors.New("bid currency must match lot currency")
	}
	if bid.GetAmount().GetAmount() < lot.GetCurrentPrice().GetAmount()+lot.GetRule().GetMinIncrement().GetAmount() {
		return errors.New("bid amount is lower than current price plus min increment")
	}
	lot.CurrentPrice = bid.GetAmount()
	lot.LeadingUserId = bid.UserId
	lot.LeadingNickname = bid.Nickname
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_BIDDING_ACTIVE
	if lot.GetRule().GetCapPrice() != nil {
		if bid.GetAmount().GetCurrency() != lot.GetRule().GetCapPrice().GetCurrency() {
			return errors.New("bid currency must match cap price currency")
		}
		if bid.GetAmount().GetAmount() >= lot.GetRule().GetCapPrice().GetAmount() {
			lot.Status = v1.LotStatus_LOT_STATUS_SETTLED
			lot.SettledAtUnixMs = nowMs
			lot.WinnerUserId = bid.UserId
			lot.WinnerNickname = bid.Nickname
			lot.FinalPrice = bid.GetAmount()
			lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_SETTLE_READY
			lot.Version++
			return nil
		}
	}

	remainingMs := lot.EndsAtUnixMs - nowMs
	windowMs := int64(lot.GetRule().GetAntiSnipeWindowSeconds()) * 1000
	extendCount := int32(0)
	if lot.DuelState != nil {
		extendCount = lot.DuelState.ExtendCount
	}
	if remainingMs > 0 && remainingMs <= windowMs && extendCount < lot.GetRule().GetMaxExtendCount() {
		lot.EndsAtUnixMs += int64(lot.GetRule().GetAntiSnipeExtendSeconds()) * 1000
		if lot.DuelState == nil {
			lot.DuelState = &v1.DuelState{LotId: lot.Id, MaxExtendCount: lot.GetRule().GetMaxExtendCount()}
		}
		lot.DuelState.ExtendCount++
		lot.DuelState.LotId = lot.Id
		lot.DuelState.EndsAtUnixMs = lot.EndsAtUnixMs
		lot.DuelState.MaxExtendCount = lot.GetRule().GetMaxExtendCount()
	}

	lot.Version++
	return nil
}

func RevealTrustCard(lot *v1.Lot, cardID string, nowMs int64) (*v1.TrustRevealCard, error) {
	if lot == nil {
		return nil, errors.New("lot is required")
	}
	if cardID == "" {
		return nil, errors.New("trust card id is required")
	}
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

func StartDuel(lot *v1.Lot, ranking []*v1.RankingItem, nowMs int64, userAID, userBID string) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		return errors.New("only live lot can enter duel mode")
	}
	if len(ranking) < 2 {
		return errors.New("at least two bidders are required to enter duel mode")
	}

	var userA, userB *v1.RankingItem
	for _, item := range ranking {
		if userAID != "" && item.UserId == userAID {
			userA = item
		}
		if userBID != "" && item.UserId == userBID {
			userB = item
		}
	}
	if userAID != "" && userA == nil {
		return errors.New("duel user A is not in ranking")
	}
	if userBID != "" && userB == nil {
		return errors.New("duel user B is not in ranking")
	}
	if userA != nil && userB != nil && userA.UserId == userB.UserId {
		return errors.New("duel users must be different")
	}
	for _, item := range ranking {
		if userA == nil && (userB == nil || item.UserId != userB.UserId) {
			userA = item
			continue
		}
		if userB == nil && item.UserId != userA.UserId {
			userB = item
			break
		}
	}
	if userA == nil || userB == nil || userA.UserId == userB.UserId {
		return errors.New("at least two distinct bidders are required to enter duel mode")
	}

	extendCount := int32(0)
	if lot.DuelState != nil {
		extendCount = lot.DuelState.ExtendCount
	}
	lot.DuelState = &v1.DuelState{
		Active:          true,
		LotId:           lot.Id,
		UserAId:         userA.UserId,
		UserANickname:   userA.Nickname,
		UserBId:         userB.UserId,
		UserBNickname:   userB.Nickname,
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
	if lot == nil {
		return errors.New("lot is required")
	}
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

func CancelLot(lot *v1.Lot, reason string, nowMs int64) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_LIVE {
		return fmt.Errorf("only live lot can be cancelled, current status: %s", lot.Status)
	}
	if reason == "" {
		return errors.New("cancel reason is required")
	}
	lot.Status = v1.LotStatus_LOT_STATUS_CANCELLED
	lot.CancelReason = reason
	lot.CancelledAtUnixMs = nowMs
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_SETTLE_READY
	if lot.DuelState != nil {
		lot.DuelState.Active = false
	}
	lot.Version++
	return nil
}
