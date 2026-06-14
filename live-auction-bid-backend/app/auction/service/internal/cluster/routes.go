package cluster

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

var ErrShardUnavailable = errors.New("assigned shard is unavailable")

type RouteTable interface {
	AssignRoom(ctx context.Context, roomID string) (Shard, error)
	ResolveRoom(ctx context.Context, roomID string) (Shard, bool, error)
	BindRoom(ctx context.Context, roomID string, shardID int) error
	ClearRoom(ctx context.Context, roomID string) error
	BindLot(ctx context.Context, lotID string, shardID int) error
	ResolveLot(ctx context.Context, lotID string) (Shard, bool, error)
	BindOrder(ctx context.Context, orderID string, shardID int) error
	ResolveOrder(ctx context.Context, orderID string) (Shard, bool, error)
}

type MemoryRouteTable struct {
	registry *StaticRegistry
	mu       sync.RWMutex
	lots     map[string]int
	orders   map[string]int
}

func NewMemoryRouteTable(registry *StaticRegistry) (*MemoryRouteTable, error) {
	if registry == nil {
		return nil, errors.New("registry is required")
	}
	return &MemoryRouteTable{
		registry: registry,
		lots:     make(map[string]int),
		orders:   make(map[string]int),
	}, nil
}

func (t *MemoryRouteTable) AssignRoom(_ context.Context, roomID string) (Shard, error) {
	return t.registry.AssignRoom(roomID)
}

func (t *MemoryRouteTable) ResolveRoom(_ context.Context, roomID string) (Shard, bool, error) {
	shard, ok := t.registry.RouteExistingRoom(roomID)
	return shard, ok, nil
}

func (t *MemoryRouteTable) BindRoom(_ context.Context, roomID string, shardID int) error {
	return t.registry.AssignRoomToShard(roomID, shardID)
}

func (t *MemoryRouteTable) ClearRoom(_ context.Context, roomID string) error {
	return t.registry.ClearRoomAssignment(roomID)
}

func (t *MemoryRouteTable) BindLot(_ context.Context, lotID string, shardID int) error {
	lotID = strings.TrimSpace(lotID)
	if lotID == "" {
		return nil
	}
	if _, ok := t.registry.LookupShard(shardID); !ok {
		return fmt.Errorf("unknown shard: %d", shardID)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lots[lotID] = shardID
	return nil
}

func (t *MemoryRouteTable) ResolveLot(_ context.Context, lotID string) (Shard, bool, error) {
	lotID = strings.TrimSpace(lotID)
	if lotID == "" {
		return Shard{}, false, nil
	}
	t.mu.RLock()
	shardID, ok := t.lots[lotID]
	t.mu.RUnlock()
	if !ok {
		return Shard{}, false, nil
	}
	shard, ok := t.registry.LookupShard(shardID)
	if !ok || !shard.ServesExistingRooms() {
		return Shard{}, false, fmt.Errorf("%w: shard %d", ErrShardUnavailable, shardID)
	}
	return shard, true, nil
}

func (t *MemoryRouteTable) BindOrder(_ context.Context, orderID string, shardID int) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil
	}
	if _, ok := t.registry.LookupShard(shardID); !ok {
		return fmt.Errorf("unknown shard: %d", shardID)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.orders[orderID] = shardID
	return nil
}

func (t *MemoryRouteTable) ResolveOrder(_ context.Context, orderID string) (Shard, bool, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return Shard{}, false, nil
	}
	t.mu.RLock()
	shardID, ok := t.orders[orderID]
	t.mu.RUnlock()
	if !ok {
		return Shard{}, false, nil
	}
	shard, ok := t.registry.LookupShard(shardID)
	if !ok || !shard.ServesExistingRooms() {
		return Shard{}, false, fmt.Errorf("%w: shard %d", ErrShardUnavailable, shardID)
	}
	return shard, true, nil
}

func ShardIDFromHeader(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	id, err := strconv.Atoi(value)
	return id, err == nil
}
