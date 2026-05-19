package data

import (
	"context"
	"errors"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
)

var ErrRedisBidRepositoryNotConfigured = errors.New("redis bid repository is not configured")

type RedisBidRepository struct{}

func NewRedisBidRepository() *RedisBidRepository { return &RedisBidRepository{} }

type AtomicBidResult struct {
	Accepted bool       `json:"accepted"`
	Lot      *biz.Lot   `json:"lot,omitempty"`
	Bid      *biz.Bid   `json:"bid,omitempty"`
	Reason   string     `json:"reason,omitempty"`
	Ranking  []biz.RankingItem `json:"ranking,omitempty"`
}

func (r *RedisBidRepository) PlaceBidAtomic(ctx context.Context, lotID string, bid biz.Bid) (*AtomicBidResult, error) {
	return nil, ErrRedisBidRepositoryNotConfigured
}
