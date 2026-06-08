package data

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
)

var renewLeaseScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
  return 1
end
return 0
`)

var releaseLeaseScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  redis.call('DEL', KEYS[1])
  return 1
end
return 0
`)

type RedisLeaseProvider struct {
	redis *redis.Client
}

func NewRedisLeaseProvider(store *Store) *RedisLeaseProvider {
	if store == nil {
		return &RedisLeaseProvider{}
	}
	return &RedisLeaseProvider{redis: store.redis}
}

func (p *RedisLeaseProvider) TryAcquire(ctx context.Context, key, owner string, ttl time.Duration) (auction.Lease, bool, error) {
	if p == nil || p.redis == nil {
		return nil, false, errors.New("redis lease provider is not initialized")
	}
	if key == "" {
		return nil, false, errors.New("lease key is required")
	}
	if owner == "" {
		return nil, false, errors.New("lease owner is required")
	}
	if ttl <= 0 {
		return nil, false, errors.New("lease ttl is required")
	}
	ok, err := p.redis.SetNX(ctx, key, owner, ttl).Result()
	if err != nil || !ok {
		return nil, ok, err
	}
	return &redisLease{redis: p.redis, key: key, owner: owner, ttl: ttl}, true, nil
}

func (p *RedisLeaseProvider) Owner(ctx context.Context, key string) (string, error) {
	if p == nil || p.redis == nil || key == "" {
		return "", nil
	}
	owner, err := p.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return owner, err
}

type redisLease struct {
	redis *redis.Client
	key   string
	owner string
	ttl   time.Duration
}

func (l *redisLease) Key() string   { return l.key }
func (l *redisLease) Owner() string { return l.owner }

func (l *redisLease) Renew(ctx context.Context) (bool, error) {
	if l == nil || l.redis == nil {
		return false, errors.New("redis lease is not initialized")
	}
	result, err := renewLeaseScript.Run(ctx, l.redis, []string{l.key}, l.owner, l.ttl.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (l *redisLease) Release(ctx context.Context) error {
	if l == nil || l.redis == nil {
		return nil
	}
	return releaseLeaseScript.Run(ctx, l.redis, []string{l.key}, l.owner).Err()
}
