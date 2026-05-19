package biz

import "context"

type AtmosphereRequest struct {
	Lot          *Lot          `json:"lot"`
	LatestBid    *Bid          `json:"latestBid,omitempty"`
	Ranking      []RankingItem `json:"ranking,omitempty"`
	OnlineUsers  int           `json:"onlineUsers,omitempty"`
	SecondsLeft  int           `json:"secondsLeft,omitempty"`
}

type AtmosphereAI interface {
	OnBid(ctx context.Context, lot *Lot, bid Bid) string
	SuggestStartPrice(ctx context.Context, title, description string, referencePrice Money) Money
	GenerateLine(ctx context.Context, req AtmosphereRequest) (string, error)
}
