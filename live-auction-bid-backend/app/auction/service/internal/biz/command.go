package biz

// CreateLotCommand 是主播创建拍品草稿的应用命令。
type CreateLotCommand struct {
	RoomID      string            `json:"roomId"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	ImageURL    string            `json:"imageUrl"`
	Rule        BidRule           `json:"rule"`
	TrustCards  []TrustRevealCard `json:"trustCards"`
}

// PlaceBidCommand 是观众出价命令。
type PlaceBidCommand struct {
	LotID              string `json:"lotId"`
	UserID             string `json:"userId"`
	Nickname           string `json:"nickname"`
	Amount             Money  `json:"amount"`
	ClientKnownVersion int64  `json:"clientKnownVersion"`
	IdempotencyKey     string `json:"idempotencyKey"`
}
