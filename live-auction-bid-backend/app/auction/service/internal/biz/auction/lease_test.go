package auction

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestAuctionCloseWorkerLeaseStandbyAndTakeover(t *testing.T) {
	ctx := context.Background()
	provider := newFakeLeaseProvider()

	var firstLease Lease
	first := NewAuctionCloseWorker(nil, time.Second, 10).BindLease(provider, "auction:lease:test-close", "worker-a", time.Second, 100*time.Millisecond)
	if !first.ensureLease(ctx, &firstLease) {
		t.Fatal("first worker should acquire lease")
	}
	if first.WorkerStatus(ctx).Mode != WorkerModeLeader {
		t.Fatalf("first worker should be leader: %+v", first.WorkerStatus(ctx))
	}

	var secondLease Lease
	second := NewAuctionCloseWorker(nil, time.Second, 10).BindLease(provider, "auction:lease:test-close", "worker-b", time.Second, 100*time.Millisecond)
	if second.ensureLease(ctx, &secondLease) {
		t.Fatal("second worker must not acquire an owned lease")
	}
	secondStatus := second.WorkerStatus(ctx)
	if secondStatus.Mode != WorkerModeStandby || secondStatus.LeaseOwner != "worker-a" {
		t.Fatalf("second worker should be standby behind worker-a: %+v", secondStatus)
	}

	if err := firstLease.Release(ctx); err != nil {
		t.Fatalf("release first lease failed: %v", err)
	}
	if !second.ensureLease(ctx, &secondLease) {
		t.Fatal("second worker should acquire lease after release")
	}
	if second.WorkerStatus(ctx).Mode != WorkerModeLeader {
		t.Fatalf("second worker should become leader: %+v", second.WorkerStatus(ctx))
	}
}

type fakeLeaseProvider struct {
	mu     sync.Mutex
	owners map[string]string
}

func newFakeLeaseProvider() *fakeLeaseProvider {
	return &fakeLeaseProvider{owners: make(map[string]string)}
}

func (p *fakeLeaseProvider) TryAcquire(_ context.Context, key, owner string, _ time.Duration) (Lease, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if existing := p.owners[key]; existing != "" {
		return nil, false, nil
	}
	p.owners[key] = owner
	return &fakeLease{provider: p, key: key, owner: owner}, true, nil
}

func (p *fakeLeaseProvider) Owner(_ context.Context, key string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.owners[key], nil
}

type fakeLease struct {
	provider *fakeLeaseProvider
	key      string
	owner    string
}

func (l *fakeLease) Key() string {
	return l.key
}

func (l *fakeLease) Owner() string {
	return l.owner
}

func (l *fakeLease) Renew(context.Context) (bool, error) {
	l.provider.mu.Lock()
	defer l.provider.mu.Unlock()
	return l.provider.owners[l.key] == l.owner, nil
}

func (l *fakeLease) Release(context.Context) error {
	l.provider.mu.Lock()
	defer l.provider.mu.Unlock()
	if l.provider.owners[l.key] == l.owner {
		delete(l.provider.owners, l.key)
	}
	return nil
}
