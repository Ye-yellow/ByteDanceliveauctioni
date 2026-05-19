package auction

import v1 "live-auction-bid/backend/api/auction/service/v1"

// CreateLotCommand 是创建拍品的业务入参。
type CreateLotCommand struct {
	RoomID      string
	Title       string
	Description string
	ImageURL    string
	Rule        *v1.BidRule
	TrustCards  []*v1.TrustRevealCard
}

// PlaceBidCommand 是出价业务入参。
type PlaceBidCommand struct {
	LotID              string
	UserID             string
	Nickname           string
	Amount             *v1.Money
	ClientKnownVersion int64
	IdempotencyKey     string
}
