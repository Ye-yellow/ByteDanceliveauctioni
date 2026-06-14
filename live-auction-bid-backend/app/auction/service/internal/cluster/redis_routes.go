package cluster

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRouteConfig struct {
	Addr      string
	Password  string
	DB        int
	KeyPrefix string
	TTL       time.Duration
}

type RedisRouteTable struct {
	registry *StaticRegistry
	client   *redis.Client
	prefix   string
	ttl      time.Duration
}

func NewRedisRouteTable(registry *StaticRegistry, cfg RedisRouteConfig) (*RedisRouteTable, error) {
	if registry == nil {
		return nil, errors.New("registry is required")
	}
	cfg.Addr = strings.TrimSpace(cfg.Addr)
	if cfg.Addr == "" {
		return nil, errors.New("redis route table addr is required")
	}
	cfg.KeyPrefix = strings.Trim(strings.TrimSpace(cfg.KeyPrefix), ":")
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "auction:route"
	}
	return &RedisRouteTable{
		registry: registry,
		client:   redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB}),
		prefix:   cfg.KeyPrefix,
		ttl:      cfg.TTL,
	}, nil
}

func (t *RedisRouteTable) Close() error {
	if t == nil || t.client == nil {
		return nil
	}
	return t.client.Close()
}

func (t *RedisRouteTable) AssignRoom(ctx context.Context, roomID string) (Shard, error) {
	if shard, ok, err := t.ResolveRoom(ctx, roomID); err != nil {
		return Shard{}, err
	} else if ok {
		return shard, nil
	}
	shard, err := t.registry.AssignRoom(roomID)
	if err != nil {
		return Shard{}, err
	}
	ok, err := t.client.SetNX(ctx, t.key("room", roomID), strconv.Itoa(shard.ID), t.ttl).Result()
	if err != nil {
		return Shard{}, err
	}
	if ok {
		return shard, nil
	}
	resolved, found, err := t.ResolveRoom(ctx, roomID)
	if err != nil {
		return Shard{}, err
	}
	if found {
		return resolved, nil
	}
	return shard, nil
}

func (t *RedisRouteTable) ResolveRoom(ctx context.Context, roomID string) (Shard, bool, error) {
	return t.resolve(ctx, "room", roomID)
}

func (t *RedisRouteTable) BindRoom(ctx context.Context, roomID string, shardID int) error {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return errors.New("room id is required")
	}
	shard, ok := t.registry.LookupShard(shardID)
	if !ok {
		return fmt.Errorf("unknown shard: %d", shardID)
	}
	if !shard.ServesExistingRooms() {
		return fmt.Errorf("shard %d is unavailable", shardID)
	}
	if err := t.client.Set(ctx, t.key("room", roomID), strconv.Itoa(shardID), t.ttl).Err(); err != nil {
		return err
	}
	if err := t.registry.AssignRoomToShard(roomID, shardID); err != nil {
		_ = t.client.Del(ctx, t.key("room", roomID)).Err()
		return err
	}
	return nil
}

func (t *RedisRouteTable) ClearRoom(ctx context.Context, roomID string) error {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return errors.New("room id is required")
	}
	if err := t.client.Del(ctx, t.key("room", roomID)).Err(); err != nil {
		return err
	}
	return t.registry.ClearRoomAssignment(roomID)
}

func (t *RedisRouteTable) BindLot(ctx context.Context, lotID string, shardID int) error {
	return t.bind(ctx, "lot", lotID, shardID)
}

func (t *RedisRouteTable) ResolveLot(ctx context.Context, lotID string) (Shard, bool, error) {
	return t.resolve(ctx, "lot", lotID)
}

func (t *RedisRouteTable) BindOrder(ctx context.Context, orderID string, shardID int) error {
	return t.bind(ctx, "order", orderID, shardID)
}

func (t *RedisRouteTable) ResolveOrder(ctx context.Context, orderID string) (Shard, bool, error) {
	return t.resolve(ctx, "order", orderID)
}

func (t *RedisRouteTable) bind(ctx context.Context, kind, id string, shardID int) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if _, ok := t.registry.LookupShard(shardID); !ok {
		return fmt.Errorf("unknown shard: %d", shardID)
	}
	return t.client.Set(ctx, t.key(kind, id), strconv.Itoa(shardID), t.ttl).Err()
}

func (t *RedisRouteTable) resolve(ctx context.Context, kind, id string) (Shard, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Shard{}, false, nil
	}
	raw, err := t.client.Get(ctx, t.key(kind, id)).Result()
	if errors.Is(err, redis.Nil) {
		return Shard{}, false, nil
	}
	if err != nil {
		return Shard{}, false, err
	}
	shardID, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return Shard{}, false, err
	}
	shard, ok := t.registry.LookupShard(shardID)
	if !ok || !shard.ServesExistingRooms() {
		return Shard{}, false, fmt.Errorf("%w: shard %d", ErrShardUnavailable, shardID)
	}
	return shard, true, nil
}

func (t *RedisRouteTable) key(kind, id string) string {
	return t.prefix + ":" + strings.Trim(kind, ":") + ":" + strings.TrimSpace(id)
}
