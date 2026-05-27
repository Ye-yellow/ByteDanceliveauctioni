package auction

import (
	"strings"

	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func (v LotResultViewer) CanViewPrivateAuctionData() bool {
	switch v.Role {
	case v1.UserRole_USER_ROLE_ADMIN, v1.UserRole_USER_ROLE_ANCHOR, v1.UserRole_USER_ROLE_OPERATOR:
		return true
	default:
		return false
	}
}

func (v LotResultViewer) CanViewBuyerIdentity(userID string) bool {
	if v.CanViewPrivateAuctionData() {
		return true
	}
	return v.Role == v1.UserRole_USER_ROLE_BUYER && v.UserID != "" && v.UserID == userID
}

func MaskBuyerNickname(nickname string) string {
	value := strings.TrimSpace(nickname)
	if value == "" {
		return ""
	}
	for _, char := range value {
		if char != '*' {
			return string(char) + "***"
		}
	}
	return "***"
}

func LotForViewer(lot *v1.Lot, viewer LotResultViewer) *v1.Lot {
	if lot == nil {
		return nil
	}
	cloned := proto.Clone(lot).(*v1.Lot)
	RedactLotForViewer(cloned, viewer)
	return cloned
}

func LotsForViewer(lots []*v1.Lot, viewer LotResultViewer) []*v1.Lot {
	if lots == nil {
		return nil
	}
	out := make([]*v1.Lot, 0, len(lots))
	for _, lot := range lots {
		out = append(out, LotForViewer(lot, viewer))
	}
	return out
}

func BidForViewer(bid *v1.Bid, viewer LotResultViewer) *v1.Bid {
	if bid == nil {
		return nil
	}
	cloned := proto.Clone(bid).(*v1.Bid)
	RedactBidForViewer(cloned, viewer)
	return cloned
}

func RankingForViewer(ranking []*v1.RankingItem, viewer LotResultViewer) []*v1.RankingItem {
	if ranking == nil {
		return nil
	}
	out := make([]*v1.RankingItem, 0, len(ranking))
	for _, item := range ranking {
		if item == nil {
			out = append(out, nil)
			continue
		}
		cloned := proto.Clone(item).(*v1.RankingItem)
		out = append(out, cloned)
	}
	RedactRankingForViewer(out, viewer)
	return out
}

func SnapshotForViewer(snapshot *v1.RoomSnapshot, viewer LotResultViewer) *v1.RoomSnapshot {
	if snapshot == nil {
		return nil
	}
	if viewer.CanViewPrivateAuctionData() {
		return snapshot
	}
	cloned := proto.Clone(snapshot).(*v1.RoomSnapshot)
	RedactSnapshotForViewer(cloned, viewer)
	return cloned
}

func EventForViewer(event v1.AuctionEvent, viewer LotResultViewer) v1.AuctionEvent {
	if viewer.CanViewPrivateAuctionData() {
		return event
	}
	cloned := proto.Clone(&event).(*v1.AuctionEvent)
	RedactEventForViewer(cloned, viewer)
	return *cloned
}

func RedactEventForViewer(event *v1.AuctionEvent, viewer LotResultViewer) {
	if event == nil || viewer.CanViewPrivateAuctionData() {
		return
	}
	RedactLotForViewer(event.Lot, viewer)
	RedactBidForViewer(event.Bid, viewer)
	RedactRankingForViewer(event.Ranking, viewer)
	RedactDuelStateForViewer(event.DuelState, viewer)
	RedactSnapshotForViewer(event.Snapshot, viewer)
	switch event.Type {
	case v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_OUTBID:
		event.Reason = "previous_leader_outbid"
	case v1.AuctionEventType_AUCTION_EVENT_TYPE_ORDER_CREATED,
		v1.AuctionEventType_AUCTION_EVENT_TYPE_PAYMENT_SUCCESS:
		event.Reason = ""
	}
}

func RedactSnapshotForViewer(snapshot *v1.RoomSnapshot, viewer LotResultViewer) {
	if snapshot == nil || viewer.CanViewPrivateAuctionData() {
		return
	}
	RedactLotForViewer(snapshot.CurrentLot, viewer)
	RedactRankingForViewer(snapshot.Ranking, viewer)
	for _, bid := range snapshot.RecentBids {
		RedactBidForViewer(bid, viewer)
	}
}

func RedactLotForViewer(lot *v1.Lot, viewer LotResultViewer) {
	if lot == nil || viewer.CanViewPrivateAuctionData() {
		return
	}
	if !viewer.CanViewBuyerIdentity(lot.GetLeadingUserId()) {
		lot.LeadingUserId = ""
		if IsAuctionOpenStatus(lot.Status) {
			lot.LeadingNickname = MaskBuyerNickname(lot.GetLeadingNickname())
		} else {
			lot.LeadingNickname = ""
		}
	}
	if !viewer.CanViewBuyerIdentity(lot.GetWinnerUserId()) {
		lot.WinnerUserId = ""
		lot.WinnerNickname = MaskBuyerNickname(lot.GetWinnerNickname())
	}
	RedactDuelStateForViewer(lot.DuelState, viewer)
}

func RedactBidForViewer(bid *v1.Bid, viewer LotResultViewer) {
	if bid == nil || viewer.CanViewPrivateAuctionData() || viewer.CanViewBuyerIdentity(bid.GetUserId()) {
		return
	}
	bid.UserId = ""
	bid.Nickname = MaskBuyerNickname(bid.GetNickname())
}

func RedactRankingForViewer(ranking []*v1.RankingItem, viewer LotResultViewer) {
	if viewer.CanViewPrivateAuctionData() {
		return
	}
	for _, item := range ranking {
		if item == nil || viewer.CanViewBuyerIdentity(item.GetUserId()) {
			continue
		}
		item.UserId = ""
		item.Nickname = MaskBuyerNickname(item.GetNickname())
	}
}

func RedactDuelStateForViewer(duel *v1.DuelState, viewer LotResultViewer) {
	if duel == nil || viewer.CanViewPrivateAuctionData() {
		return
	}
	if !viewer.CanViewBuyerIdentity(duel.GetUserAId()) {
		duel.UserAId = ""
		duel.UserANickname = MaskBuyerNickname(duel.GetUserANickname())
	}
	if !viewer.CanViewBuyerIdentity(duel.GetUserBId()) {
		duel.UserBId = ""
		duel.UserBNickname = MaskBuyerNickname(duel.GetUserBNickname())
	}
}
