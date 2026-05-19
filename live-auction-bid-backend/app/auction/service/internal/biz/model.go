package biz

type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

func CNY(amount int64) Money { return Money{Amount: amount, Currency: "CNY"} }

type LotStatus string

const (
	LotStatusDraft     LotStatus = "LOT_STATUS_DRAFT"
	LotStatusLive      LotStatus = "LOT_STATUS_LIVE"
	LotStatusSettled   LotStatus = "LOT_STATUS_SETTLED"
	LotStatusCancelled LotStatus = "LOT_STATUS_CANCELLED"
)

type TrustCardType string

const (
	TrustCardTypeCertificate TrustCardType = "TRUST_CARD_TYPE_CERTIFICATE"
	TrustCardTypeFlaw        TrustCardType = "TRUST_CARD_TYPE_FLAW"
	TrustCardTypeDetail      TrustCardType = "TRUST_CARD_TYPE_DETAIL"
	TrustCardTypeService     TrustCardType = "TRUST_CARD_TYPE_SERVICE"
	TrustCardTypePriceRef    TrustCardType = "TRUST_CARD_TYPE_PRICE_REF"
)

type PlaybookStage string

const (
	PlaybookStageWarmUp        PlaybookStage = "PLAYBOOK_STAGE_WARM_UP"
	PlaybookStageTrustBlocked  PlaybookStage = "PLAYBOOK_STAGE_TRUST_BLOCKED"
	PlaybookStageBiddingActive PlaybookStage = "PLAYBOOK_STAGE_BIDDING_ACTIVE"
	PlaybookStageDuelReady     PlaybookStage = "PLAYBOOK_STAGE_DUEL_READY"
	PlaybookStageDuelMode      PlaybookStage = "PLAYBOOK_STAGE_DUEL_MODE"
	PlaybookStageSettleReady   PlaybookStage = "PLAYBOOK_STAGE_SETTLE_READY"
)

type EventType string

const (
	EventRoomSnapshot   EventType = "AUCTION_EVENT_TYPE_ROOM_SNAPSHOT"
	EventLotCreated     EventType = "AUCTION_EVENT_TYPE_LOT_CREATED"
	EventLotStarted     EventType = "AUCTION_EVENT_TYPE_LOT_STARTED"
	EventLotUpdated     EventType = "AUCTION_EVENT_TYPE_LOT_UPDATED"
	EventBidAccepted    EventType = "AUCTION_EVENT_TYPE_BID_ACCEPTED"
	EventBidRejected    EventType = "AUCTION_EVENT_TYPE_BID_REJECTED"
	EventRankingUpdated EventType = "AUCTION_EVENT_TYPE_RANKING_UPDATED"
	EventTrustRevealed  EventType = "AUCTION_EVENT_TYPE_TRUST_REVEALED"
	EventDuelStarted    EventType = "AUCTION_EVENT_TYPE_DUEL_STARTED"
	EventDuelEnded      EventType = "AUCTION_EVENT_TYPE_DUEL_ENDED"
	EventLotSettled     EventType = "AUCTION_EVENT_TYPE_LOT_SETTLED"
)

type BidRule struct {
	StartPrice             Money `json:"startPrice"`
	MinIncrement           Money `json:"minIncrement"`
	DurationSeconds        int32 `json:"durationSeconds"`
	AntiSnipeWindowSeconds int32 `json:"antiSnipeWindowSeconds"`
	AntiSnipeExtendSeconds int32 `json:"antiSnipeExtendSeconds"`
	MaxExtendCount         int32 `json:"maxExtendCount"`
}

type TrustRevealCard struct {
	ID               string        `json:"id"`
	LotID            string        `json:"lotId"`
	Type             TrustCardType `json:"type"`
	Title            string        `json:"title"`
	Content          string        `json:"content"`
	ImageURL         string        `json:"imageUrl"`
	Revealed         bool          `json:"revealed"`
	RevealedAtUnixMs int64         `json:"revealedAtUnixMs"`
}

type Bid struct {
	ID              string `json:"id"`
	LotID           string `json:"lotId"`
	UserID          string `json:"userId"`
	Nickname        string `json:"nickname"`
	Amount          Money  `json:"amount"`
	CreatedAtUnixMs int64  `json:"createdAtUnixMs"`
}

type RankingItem struct {
	Rank        int32  `json:"rank"`
	UserID      string `json:"userId"`
	Nickname    string `json:"nickname"`
	Amount      Money  `json:"amount"`
	BidAtUnixMs int64  `json:"bidAtUnixMs"`
}

type DuelState struct {
	Active          bool   `json:"active"`
	LotID           string `json:"lotId"`
	UserAID         string `json:"userAId"`
	UserANickname   string `json:"userANickname"`
	UserBID         string `json:"userBId"`
	UserBNickname   string `json:"userBNickname"`
	StartedAtUnixMs int64  `json:"startedAtUnixMs"`
	EndsAtUnixMs    int64  `json:"endsAtUnixMs"`
	ExtendCount     int32  `json:"extendCount"`
	MaxExtendCount  int32  `json:"maxExtendCount"`
}

type Lot struct {
	ID              string            `json:"id"`
	RoomID          string            `json:"roomId"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	ImageURL        string            `json:"imageUrl"`
	Status          LotStatus         `json:"status"`
	Rule            BidRule           `json:"rule"`
	CurrentPrice    Money             `json:"currentPrice"`
	LeadingUserID   string            `json:"leadingUserId"`
	LeadingNickname string            `json:"leadingNickname"`
	StartedAtUnixMs int64             `json:"startedAtUnixMs"`
	EndsAtUnixMs    int64             `json:"endsAtUnixMs"`
	SettledAtUnixMs int64             `json:"settledAtUnixMs"`
	WinnerUserID    string            `json:"winnerUserId"`
	WinnerNickname  string            `json:"winnerNickname"`
	FinalPrice      Money             `json:"finalPrice"`
	Version         int64             `json:"version"`
	TrustCards      []TrustRevealCard `json:"trustCards"`
	DuelState       DuelState         `json:"duelState"`
	PlaybookStage   PlaybookStage     `json:"playbookStage"`
}

type RoomSnapshot struct {
	RoomID           string        `json:"roomId"`
	CurrentLot       *Lot          `json:"currentLot,omitempty"`
	Ranking          []RankingItem `json:"ranking"`
	RecentBids       []Bid         `json:"recentBids"`
	PlaybookStage    PlaybookStage `json:"playbookStage"`
	ServerTimeUnixMs int64         `json:"serverTimeUnixMs"`
}

type AuctionEvent struct {
	ID               string           `json:"id"`
	Type             EventType        `json:"type"`
	RoomID           string           `json:"roomId"`
	LotID            string           `json:"lotId"`
	OccurredAtUnixMs int64            `json:"occurredAtUnixMs"`
	Lot              *Lot             `json:"lot,omitempty"`
	Bid              *Bid             `json:"bid,omitempty"`
	Ranking          []RankingItem    `json:"ranking,omitempty"`
	TrustCard        *TrustRevealCard `json:"trustCard,omitempty"`
	DuelState        *DuelState       `json:"duelState,omitempty"`
	Snapshot         *RoomSnapshot    `json:"snapshot,omitempty"`
	Reason           string           `json:"reason,omitempty"`
}
