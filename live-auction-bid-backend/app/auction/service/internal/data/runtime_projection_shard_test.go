package data

import (
	"context"
	"sync"
	"testing"
	"time"

	"live-auction-bid/backend/app/auction/service/internal/biz/auction"

	"gorm.io/gorm/schema"
)

func TestRuntimeProjectionShardStableAndInRange(t *testing.T) {
	const shardCount = 16
	first := runtimeProjectionShard("lot_stable", shardCount)
	for i := 0; i < 10; i++ {
		got := runtimeProjectionShard("lot_stable", shardCount)
		if got != first {
			t.Fatalf("shard should be stable: first=%d got=%d", first, got)
		}
		if got < 0 || got >= shardCount {
			t.Fatalf("shard out of range: got=%d shard_count=%d", got, shardCount)
		}
	}
}

func TestStoreRuntimeProjectionShardUsesConfiguredShardCount(t *testing.T) {
	store := &Store{runtimeProjectionShards: 4}
	for _, lotID := range []string{"lot-a", "lot-b", "lot-c", "lot-d"} {
		got := store.runtimeProjectionShard(lotID)
		if got < 0 || got >= 4 {
			t.Fatalf("configured shard out of range: lot=%s got=%d", lotID, got)
		}
	}
}

func TestRuntimeProjectionShardOffsetShardIDDoesNotAutoIncrement(t *testing.T) {
	parsed, err := schema.Parse(&AuctionRuntimeProjectionShardOffsetModel{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse shard offset schema: %v", err)
	}
	field := parsed.LookUpField("ShardID")
	if field == nil {
		t.Fatal("ShardID field should exist")
	}
	if field.AutoIncrement {
		t.Fatal("ShardID must not auto-increment because shard 0 is a valid projection shard")
	}
}

func TestRuntimeProjectionWorkerShardLeaseStandbyAndTakeover(t *testing.T) {
	ctx := context.Background()
	provider := newDataFakeLeaseProvider()
	store := &Store{runtimeProjectionShards: 2}

	first := NewRuntimeProjectionWorker(store, nil, time.Second, 10).BindLease(provider, "worker-a", time.Second, 100*time.Millisecond)
	if !first.ensureShardLease(ctx, 0) {
		t.Fatal("first projector should acquire shard lease")
	}
	firstStatus := first.WorkerStatus(ctx)
	if firstStatus.Mode != auction.WorkerModePartialOwner || len(firstStatus.OwnedShards) != 1 || firstStatus.OwnedShards[0] != 0 {
		t.Fatalf("first projector should partially own shard 0: %+v", firstStatus)
	}

	second := NewRuntimeProjectionWorker(store, nil, time.Second, 10).BindLease(provider, "worker-b", time.Second, 100*time.Millisecond)
	if second.ensureShardLease(ctx, 0) {
		t.Fatal("second projector must not acquire an owned shard lease")
	}
	secondStatus := second.WorkerStatus(ctx)
	if secondStatus.Shards[0].Mode != auction.WorkerModeStandby || secondStatus.Shards[0].LeaseOwner != "worker-a" {
		t.Fatalf("second projector should be standby behind worker-a: %+v", secondStatus.Shards[0])
	}

	first.releaseShardLease(ctx, 0)
	if !second.ensureShardLease(ctx, 0) {
		t.Fatal("second projector should acquire shard lease after release")
	}
	if second.WorkerStatus(ctx).Shards[0].Mode != auction.WorkerModeLeader {
		t.Fatalf("second projector should become shard leader: %+v", second.WorkerStatus(ctx).Shards[0])
	}
}

type dataFakeLeaseProvider struct {
	mu     sync.Mutex
	owners map[string]string
}

func newDataFakeLeaseProvider() *dataFakeLeaseProvider {
	return &dataFakeLeaseProvider{owners: make(map[string]string)}
}

func (p *dataFakeLeaseProvider) TryAcquire(_ context.Context, key, owner string, _ time.Duration) (auction.Lease, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if existing := p.owners[key]; existing != "" {
		return nil, false, nil
	}
	p.owners[key] = owner
	return &dataFakeLease{provider: p, key: key, owner: owner}, true, nil
}

func (p *dataFakeLeaseProvider) Owner(_ context.Context, key string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.owners[key], nil
}

type dataFakeLease struct {
	provider *dataFakeLeaseProvider
	key      string
	owner    string
}

func (l *dataFakeLease) Key() string {
	return l.key
}

func (l *dataFakeLease) Owner() string {
	return l.owner
}

func (l *dataFakeLease) Renew(context.Context) (bool, error) {
	l.provider.mu.Lock()
	defer l.provider.mu.Unlock()
	return l.provider.owners[l.key] == l.owner, nil
}

func (l *dataFakeLease) Release(context.Context) error {
	l.provider.mu.Lock()
	defer l.provider.mu.Unlock()
	if l.provider.owners[l.key] == l.owner {
		delete(l.provider.owners, l.key)
	}
	return nil
}
