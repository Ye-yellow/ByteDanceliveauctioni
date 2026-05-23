package auction

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

func NewLotFromRequest(id string, req *v1.CreateLotRequest) (*v1.Lot, error) {
	return NewLotDraftFromRequest(id, req, true)
}

func NewLotDraftFromRequest(id string, req *v1.CreateLotRequest, requireComplete bool) (*v1.Lot, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: create lot request is required", apperr.ErrInvalidArgument)
	}
	if requireComplete && req.GetRoomId() == "" {
		return nil, fmt.Errorf("%w: room id is required", apperr.ErrInvalidArgument)
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
		return fmt.Errorf("%w: lot title is required", apperr.ErrInvalidArgument)
	}
	if imageURL == "" {
		return fmt.Errorf("%w: lot image url is required", apperr.ErrInvalidArgument)
	}
	if rule == nil || rule.GetStartPrice() == nil || rule.GetMinIncrement() == nil {
		return fmt.Errorf("%w: bid rule, start price and min increment are required", apperr.ErrInvalidArgument)
	}
	if rule.GetStartPrice().GetCurrency() == "" || rule.GetMinIncrement().GetCurrency() == "" {
		return fmt.Errorf("%w: start price and min increment currency are required", apperr.ErrInvalidArgument)
	}
	if rule.GetStartPrice().GetCurrency() != rule.GetMinIncrement().GetCurrency() {
		return fmt.Errorf("%w: start price and min increment currency must match", apperr.ErrInvalidArgument)
	}
	if rule.GetStartPrice().GetAmount() < 0 {
		return fmt.Errorf("%w: start price amount must be >= 0", apperr.ErrInvalidArgument)
	}
	if rule.GetMinIncrement().GetAmount() <= 0 {
		return fmt.Errorf("%w: min increment amount must be > 0", apperr.ErrInvalidArgument)
	}
	if rule.GetDurationSeconds() < 60 {
		return fmt.Errorf("%w: duration seconds must be >= 60", apperr.ErrInvalidArgument)
	}
	if rule.GetAntiSnipeWindowSeconds() <= 0 {
		return fmt.Errorf("%w: anti-snipe window seconds must be > 0", apperr.ErrInvalidArgument)
	}
	if rule.GetAntiSnipeExtendSeconds() < 10 || rule.GetAntiSnipeExtendSeconds() > 30 {
		return fmt.Errorf("%w: anti-snipe extend seconds must be between 10 and 30", apperr.ErrInvalidArgument)
	}
	if rule.GetMaxExtendCount() <= 0 {
		return fmt.Errorf("%w: max extend count must be > 0", apperr.ErrInvalidArgument)
	}
	if capPrice := rule.GetCapPrice(); capPrice != nil {
		if capPrice.GetCurrency() == "" {
			return fmt.Errorf("%w: cap price currency is required", apperr.ErrInvalidArgument)
		}
		if capPrice.GetCurrency() != rule.GetStartPrice().GetCurrency() || capPrice.GetCurrency() != rule.GetMinIncrement().GetCurrency() {
			return fmt.Errorf("%w: cap price currency must match start price and min increment currency", apperr.ErrInvalidArgument)
		}
		if capPrice.GetAmount() <= rule.GetStartPrice().GetAmount() {
			return fmt.Errorf("%w: cap price amount must be greater than start price amount", apperr.ErrInvalidArgument)
		}
	}
	return nil
}

func ApplyDraftPatch(lot *v1.Lot, req *v1.PatchLotDraftRequest) error {
	if lot == nil {
		return fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if req == nil {
		return fmt.Errorf("%w: patch lot draft request is required", apperr.ErrInvalidArgument)
	}
	if lot.Status == v1.LotStatus_LOT_STATUS_LIVE || lot.Status == v1.LotStatus_LOT_STATUS_SETTLED || lot.Status == v1.LotStatus_LOT_STATUS_CANCELLED {
		return fmt.Errorf("%w: only not-started lot can be edited, current status: %s", apperr.ErrInvalidArgument, lot.Status)
	}
	if lot.QueueStatus == v1.LotQueueStatus_LOT_QUEUE_STATUS_NEXT {
		return fmt.Errorf("%w: next lot cannot be edited from add-lot draft page", apperr.ErrInvalidArgument)
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
		return fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if lot.Status == v1.LotStatus_LOT_STATUS_LIVE || lot.Status == v1.LotStatus_LOT_STATUS_SETTLED || lot.Status == v1.LotStatus_LOT_STATUS_CANCELLED {
		return fmt.Errorf("%w: only not-started lot can be queued, current status: %s", apperr.ErrInvalidArgument, lot.Status)
	}
	if err := ValidateLotReady(lot.GetTitle(), lot.GetImageUrl(), lot.GetRule()); err != nil {
		return err
	}
	if lot.QueueStatus == v1.LotQueueStatus_LOT_QUEUE_STATUS_QUEUED && lot.QueuePosition > 0 {
		return nil
	}
	if queuePosition <= 0 {
		return fmt.Errorf("%w: queue position is required", apperr.ErrInvalidArgument)
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
		return fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if lot.GetRule() == nil || lot.GetRule().GetStartPrice() == nil {
		return fmt.Errorf("%w: lot bid rule and start price are required", apperr.ErrInvalidArgument)
	}
	if lot.Status != v1.LotStatus_LOT_STATUS_DRAFT && lot.Status != v1.LotStatus_LOT_STATUS_QUEUED {
		return fmt.Errorf("%w: only draft or queued lot can be started, current status: %s", apperr.ErrInvalidArgument, lot.Status)
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
		return fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if lot.GetRule() == nil || lot.GetRule().GetMinIncrement() == nil || lot.GetCurrentPrice() == nil {
		return fmt.Errorf("%w: lot rule, min increment and current price are required", apperr.ErrInvalidArgument)
	}
	if !IsAuctionOpenStatus(lot.Status) {
		return fmt.Errorf("%w: lot is not live", apperr.ErrInvalidArgument)
	}
	if lot.EndsAtUnixMs > 0 && nowMs > lot.EndsAtUnixMs {
		return fmt.Errorf("%w: auction has ended", apperr.ErrInvalidArgument)
	}
	if bid.GetAmount() == nil || bid.GetAmount().GetCurrency() == "" {
		return fmt.Errorf("%w: bid amount and currency are required", apperr.ErrInvalidArgument)
	}
	if bid.GetUserId() == "" || bid.GetNickname() == "" {
		return fmt.Errorf("%w: bid user id and nickname are required", apperr.ErrInvalidArgument)
	}
	if bid.GetAmount().GetCurrency() != lot.GetCurrentPrice().GetCurrency() {
		return fmt.Errorf("%w: bid currency must match lot currency", apperr.ErrInvalidArgument)
	}
	if bid.GetAmount().GetAmount() < lot.GetCurrentPrice().GetAmount()+lot.GetRule().GetMinIncrement().GetAmount() {
		return fmt.Errorf("%w: bid amount is lower than current price plus min increment", apperr.ErrInvalidArgument)
	}
	lot.CurrentPrice = bid.GetAmount()
	lot.LeadingUserId = bid.UserId
	lot.LeadingNickname = bid.Nickname
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_BIDDING_ACTIVE
	if lot.GetRule().GetCapPrice() != nil {
		if bid.GetAmount().GetCurrency() != lot.GetRule().GetCapPrice().GetCurrency() {
			return fmt.Errorf("%w: bid currency must match cap price currency", apperr.ErrInvalidArgument)
		}
		if bid.GetAmount().GetAmount() >= lot.GetRule().GetCapPrice().GetAmount() {
			lot.Status = v1.LotStatus_LOT_STATUS_SETTLED
			lot.SettledAtUnixMs = nowMs
			lot.WinnerUserId = bid.UserId
			lot.WinnerNickname = bid.Nickname
			lot.FinalPrice = bid.GetAmount()
			lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_SETTLE_READY
			if lot.DuelState != nil {
				lot.DuelState.Active = false
			}
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
		lot.Status = v1.LotStatus_LOT_STATUS_EXTENDED
	}

	lot.Version++
	return nil
}

func RevealTrustCard(lot *v1.Lot, cardID string, nowMs int64) (*v1.TrustRevealCard, error) {
	if lot == nil {
		return nil, fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if cardID == "" {
		return nil, fmt.Errorf("%w: trust card id is required", apperr.ErrInvalidArgument)
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
	return nil, fmt.Errorf("%w: trust card not found", apperr.ErrInvalidArgument)
}

func StartDuel(lot *v1.Lot, ranking []*v1.RankingItem, nowMs int64, userAID, userBID string) error {
	if lot == nil {
		return fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if !IsAuctionOpenStatus(lot.Status) {
		return fmt.Errorf("%w: only live lot can enter duel mode", apperr.ErrInvalidArgument)
	}
	if len(ranking) < 2 {
		return fmt.Errorf("%w: at least two bidders are required to enter duel mode", apperr.ErrInvalidArgument)
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
		return fmt.Errorf("%w: duel user A is not in ranking", apperr.ErrInvalidArgument)
	}
	if userBID != "" && userB == nil {
		return fmt.Errorf("%w: duel user B is not in ranking", apperr.ErrInvalidArgument)
	}
	if userA != nil && userB != nil && userA.UserId == userB.UserId {
		return fmt.Errorf("%w: duel users must be different", apperr.ErrInvalidArgument)
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
		return fmt.Errorf("%w: at least two distinct bidders are required to enter duel mode", apperr.ErrInvalidArgument)
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
		return fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if !IsAuctionOpenStatus(lot.Status) {
		return fmt.Errorf("%w: only live lot can be settled, current status: %s", apperr.ErrInvalidArgument, lot.Status)
	}
	if lot.LeadingUserId == "" {
		return fmt.Errorf("%w: lot has no accepted bid to settle", apperr.ErrInvalidArgument)
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
		return fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if !IsAuctionOpenStatus(lot.Status) {
		return fmt.Errorf("%w: only live lot can be cancelled, current status: %s", apperr.ErrInvalidArgument, lot.Status)
	}
	if reason == "" {
		return fmt.Errorf("%w: cancel reason is required", apperr.ErrInvalidArgument)
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

func FailExpiredLot(lot *v1.Lot, reason string, nowMs int64) error {
	if lot == nil {
		return fmt.Errorf("%w: lot is required", apperr.ErrInvalidArgument)
	}
	if !IsAuctionOpenStatus(lot.Status) {
		return fmt.Errorf("%w: only live lot can be failed, current status: %s", apperr.ErrInvalidArgument, lot.Status)
	}
	if reason == "" {
		return fmt.Errorf("%w: fail reason is required", apperr.ErrInvalidArgument)
	}
	lot.Status = v1.LotStatus_LOT_STATUS_FAILED
	lot.CancelReason = reason
	lot.CancelledAtUnixMs = nowMs
	lot.PlaybookStage = v1.PlaybookStage_PLAYBOOK_STAGE_SETTLE_READY
	if lot.DuelState != nil {
		lot.DuelState.Active = false
	}
	lot.Version++
	return nil
}
