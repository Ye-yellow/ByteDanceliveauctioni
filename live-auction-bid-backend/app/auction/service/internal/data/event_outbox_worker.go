package data

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/observability"
)

// EventOutboxWorker owns the production Redis Stream repair loop for persisted auction events.
type EventOutboxWorker struct {
	store    *Store
	interval time.Duration
	limit    int

	leaseProvider auction.LeaseProvider
	leaseKey      string
	instanceID    string
	leaseTTL      time.Duration
	renewInterval time.Duration
	leaseOwner    string
	mode          auction.WorkerMode

	mu            sync.Mutex
	started       bool
	lastAttemptAt time.Time
	lastSuccessAt time.Time
	lastError     string
}

func NewEventOutboxWorker(store *Store, interval time.Duration, limit int) *EventOutboxWorker {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	if limit <= 0 {
		limit = 100
	}
	return &EventOutboxWorker{store: store, interval: interval, limit: limit, mode: auction.WorkerModeLeader}
}

func (w *EventOutboxWorker) BindLease(provider auction.LeaseProvider, key, instanceID string, ttl, renewInterval time.Duration) *EventOutboxWorker {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.leaseProvider = provider
	w.leaseKey = key
	w.instanceID = instanceID
	w.leaseTTL = normalizeLeaseTTL(ttl)
	w.renewInterval = normalizeRenewInterval(renewInterval)
	w.mode = auction.WorkerModeAcquiring
	return w
}

func (w *EventOutboxWorker) Start(ctx context.Context) {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.started = true
	w.mu.Unlock()

	go func() {
		var lease auction.Lease
		var lastRun time.Time
		ticker := time.NewTicker(minWorkerInterval(w.interval, w.renewInterval))
		defer ticker.Stop()
		for {
			w.tick(ctx, &lease, &lastRun)
			select {
			case <-ctx.Done():
				if lease != nil {
					_ = lease.Release(context.Background())
				}
				return
			case <-ticker.C:
			}
		}
	}()
}

func (w *EventOutboxWorker) Ping(context.Context) error {
	if w == nil || w.store == nil {
		return errors.New("event outbox worker is not initialized")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.started {
		return errors.New("event outbox worker is not started")
	}
	return nil
}

func (w *EventOutboxWorker) WorkerStatus(ctx context.Context) auction.WorkerStatus {
	w.mu.Lock()
	status := auction.WorkerStatus{
		Name:          "event_outbox_worker",
		Mode:          w.mode,
		InstanceID:    w.instanceID,
		LeaseKey:      w.leaseKey,
		LeaseOwner:    w.leaseOwner,
		Started:       w.started,
		LastAttemptAt: formatWorkerTime(w.lastAttemptAt),
		LastSuccessAt: formatWorkerTime(w.lastSuccessAt),
		LastError:     w.lastError,
	}
	w.mu.Unlock()
	if status.LeaseOwner == "" && w.leaseProvider != nil && w.leaseKey != "" {
		if owner, err := w.leaseProvider.Owner(ctx, w.leaseKey); err == nil {
			status.LeaseOwner = owner
		}
	}
	return status
}

func (w *EventOutboxWorker) tick(ctx context.Context, lease *auction.Lease, lastRun *time.Time) {
	if w == nil || w.store == nil {
		return
	}
	if !w.ensureLease(ctx, lease) {
		return
	}
	if !lastRun.IsZero() && time.Since(*lastRun) < w.interval {
		return
	}
	*lastRun = time.Now()
	if err := w.repair(ctx); err != nil && w.leaseProvider != nil && lease != nil && *lease != nil {
		w.setLeaseMode(auction.WorkerModeLeaderDegraded, (*lease).Owner())
		_ = (*lease).Release(ctx)
		*lease = nil
	}
}

func (w *EventOutboxWorker) ensureLease(ctx context.Context, lease *auction.Lease) bool {
	if w.leaseProvider == nil {
		w.setLeaseMode(auction.WorkerModeLeader, w.instanceID)
		return true
	}
	if lease == nil || *lease == nil {
		w.setLeaseMode(auction.WorkerModeAcquiring, "")
		acquired, ok, err := w.leaseProvider.TryAcquire(ctx, w.leaseKey, w.instanceID, w.leaseTTL)
		if err != nil {
			w.recordError(time.Now(), err)
			return false
		}
		if !ok {
			owner, _ := w.leaseProvider.Owner(ctx, w.leaseKey)
			w.setLeaseMode(auction.WorkerModeStandby, owner)
			return false
		}
		*lease = acquired
		w.setLeaseMode(auction.WorkerModeLeader, acquired.Owner())
		slog.Info("event outbox worker acquired lease", "lease_key", w.leaseKey, "owner", w.instanceID)
		return true
	}
	ok, err := (*lease).Renew(ctx)
	if err != nil || !ok {
		if err != nil {
			w.recordError(time.Now(), err)
		}
		w.setLeaseMode(auction.WorkerModeLostLease, "")
		*lease = nil
		slog.Warn("event outbox worker lost lease", "lease_key", w.leaseKey, "owner", w.instanceID, "error", err)
		return false
	}
	w.setLeaseMode(auction.WorkerModeLeader, (*lease).Owner())
	return true
}

func (w *EventOutboxWorker) repair(ctx context.Context) error {
	now := time.Now()
	err := w.store.RepairEventStreamOutbox(ctx, w.limit)
	if err != nil {
		w.recordError(now, err)
		return err
	}
	if pending, countErr := w.store.CountPendingEventOutbox(ctx); countErr == nil {
		observability.SetOutboxPendingCount(pending)
	}
	w.mu.Lock()
	w.lastAttemptAt = now
	w.lastSuccessAt = now
	w.lastError = ""
	w.mu.Unlock()
	return nil
}

func (w *EventOutboxWorker) setLeaseMode(mode auction.WorkerMode, owner string) {
	w.mu.Lock()
	w.mode = mode
	w.leaseOwner = owner
	w.mu.Unlock()
	observability.SetWorkerLeaseActive("event_outbox_worker", mode == auction.WorkerModeLeader || mode == auction.WorkerModeLeaderDegraded)
}

func (w *EventOutboxWorker) recordError(now time.Time, err error) {
	if err == nil {
		return
	}
	message := err.Error()
	if len(message) > 512 {
		message = message[:512]
	}
	w.mu.Lock()
	w.lastAttemptAt = now
	w.lastError = message
	w.mu.Unlock()
}
