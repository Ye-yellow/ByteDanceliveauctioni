package model

import v1 "live-auction-bid/backend/api/auction/service/v1"

type Money = v1.Money
type LotStatus = v1.LotStatus
type TrustCardType = v1.TrustCardType
type PlaybookStage = v1.PlaybookStage
type EventType = v1.AuctionEventType
type BidRule = v1.BidRule
type TrustRevealCard = v1.TrustRevealCard
type Bid = v1.Bid
type RankingItem = v1.RankingItem
type DuelState = v1.DuelState
type Lot = v1.Lot
type RoomSnapshot = v1.RoomSnapshot
type AuctionEvent = v1.AuctionEvent

const (
	LotStatusDraft     = v1.LotStatus_LOT_STATUS_DRAFT
	LotStatusLive      = v1.LotStatus_LOT_STATUS_LIVE
	LotStatusSettled   = v1.LotStatus_LOT_STATUS_SETTLED
	LotStatusCancelled = v1.LotStatus_LOT_STATUS_CANCELLED

	TrustCardTypeCertificate = v1.TrustCardType_TRUST_CARD_TYPE_CERTIFICATE
	TrustCardTypeFlaw        = v1.TrustCardType_TRUST_CARD_TYPE_FLAW
	TrustCardTypeDetail      = v1.TrustCardType_TRUST_CARD_TYPE_DETAIL
	TrustCardTypeService     = v1.TrustCardType_TRUST_CARD_TYPE_SERVICE
	TrustCardTypePriceRef    = v1.TrustCardType_TRUST_CARD_TYPE_PRICE_REF

	PlaybookStageWarmUp        = v1.PlaybookStage_PLAYBOOK_STAGE_WARM_UP
	PlaybookStageTrustBlocked  = v1.PlaybookStage_PLAYBOOK_STAGE_TRUST_BLOCKED
	PlaybookStageBiddingActive = v1.PlaybookStage_PLAYBOOK_STAGE_BIDDING_ACTIVE
	PlaybookStageDuelReady     = v1.PlaybookStage_PLAYBOOK_STAGE_DUEL_READY
	PlaybookStageDuelMode      = v1.PlaybookStage_PLAYBOOK_STAGE_DUEL_MODE
	PlaybookStageSettleReady   = v1.PlaybookStage_PLAYBOOK_STAGE_SETTLE_READY

	EventRoomSnapshot   = v1.AuctionEventType_AUCTION_EVENT_TYPE_ROOM_SNAPSHOT
	EventLotCreated     = v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_CREATED
	EventLotStarted     = v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_STARTED
	EventLotUpdated     = v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_UPDATED
	EventBidAccepted    = v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_ACCEPTED
	EventBidRejected    = v1.AuctionEventType_AUCTION_EVENT_TYPE_BID_REJECTED
	EventRankingUpdated = v1.AuctionEventType_AUCTION_EVENT_TYPE_RANKING_UPDATED
	EventTrustRevealed  = v1.AuctionEventType_AUCTION_EVENT_TYPE_TRUST_REVEALED
	EventDuelStarted    = v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_STARTED
	EventDuelEnded      = v1.AuctionEventType_AUCTION_EVENT_TYPE_DUEL_ENDED
	EventLotSettled     = v1.AuctionEventType_AUCTION_EVENT_TYPE_LOT_SETTLED
)

func CNY(amount int64) *Money {
	return &Money{Amount: amount, Currency: "CNY"}
}
