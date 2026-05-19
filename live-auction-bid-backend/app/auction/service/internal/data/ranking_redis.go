package data

import (
	"context"
	"errors"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
)

var ErrRedisRankingNotConfigured = errors.New("redis ranking store is not configured")

type RedisRankingStore struct{}

func NewRedisRankingStore() *RedisRankingStore { return &RedisRankingStore{} }

func (s *RedisRankingStore) Top(ctx context.Context, lotID string, limit int) ([]biz.RankingItem, error) {
	return nil, ErrRedisRankingNotConfigured
}
