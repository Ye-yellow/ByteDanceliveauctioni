package data

import (
	"context"
	"errors"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
)

var ErrOllamaClientNotConfigured = errors.New("ollama ai client is not configured")

type OllamaAIClient struct {
	Endpoint string
	Model    string
}

func NewOllamaAIClient(endpoint, model string) *OllamaAIClient {
	return &OllamaAIClient{Endpoint: endpoint, Model: model}
}

func (c *OllamaAIClient) SuggestAuctionRule(ctx context.Context, title, description string, referencePrice biz.Money) (*biz.PriceSuggestion, error) {
	return nil, ErrOllamaClientNotConfigured
}

func (c *OllamaAIClient) GenerateLine(ctx context.Context, req biz.AtmosphereRequest) (string, error) {
	return "", ErrOllamaClientNotConfigured
}
