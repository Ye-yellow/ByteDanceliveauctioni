package biz

import "context"

type PriceSuggestion struct {
	StartPrice   Money    `json:"startPrice"`
	ReservePrice Money    `json:"reservePrice"`
	MinIncrement Money    `json:"minIncrement"`
	Confidence   float64  `json:"confidence"`
	Reason       string   `json:"reason"`
	Signals      []string `json:"signals,omitempty"`
}

type PricingAI interface {
	SuggestAuctionRule(ctx context.Context, title, description string, referencePrice Money) (*PriceSuggestion, error)
}
