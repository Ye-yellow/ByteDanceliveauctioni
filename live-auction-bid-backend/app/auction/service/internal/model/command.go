package model

// CreateLotCommand 是创建拍品的业务入参。
type CreateLotCommand struct {
	RoomID      string
	Title       string
	Description string
	ImageURL    string
	Rule        *BidRule
	TrustCards  []*TrustRevealCard
}

// PlaceBidCommand 是出价业务入参。
type PlaceBidCommand struct {
	LotID              string
	UserID             string
	Nickname           string
	Amount             *Money
	ClientKnownVersion int64
	IdempotencyKey     string
}
