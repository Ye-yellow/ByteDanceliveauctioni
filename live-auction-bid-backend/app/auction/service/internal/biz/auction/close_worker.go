package auction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"live-auction-bid/backend/app/auction/service/internal/observability"
)

type AuctionCloseWorker struct {
	usecase  *AuctionUsecase
	interval time.Duration
	limit    int

	leaseProvider LeaseProvider
	leaseKey      string
	instanceID    string
	leaseTTL      time.Duration
	renewInterval time.Duration
	leaseOwner    string
	mode          WorkerMode

	mu            sync.Mutex
	started       bool
	lastAttemptAt time.Time
	lastSuccessAt time.Time
	lastError     string
	lastSummary   CloseExpiredSummary
}

func NewAuctionCloseWorker(usecase *AuctionUsecase, interval time.Duration, limit int) *AuctionCloseWorker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if limit <= 0 {
		limit = 100
	}
	return &AuctionCloseWorker{usecase: usecase, interval: interval, limit: limit, mode: WorkerModeLeader}
}

func (w *AuctionCloseWorker) BindLease(provider LeaseProvider, key, instanceID string, ttl, renewInterval time.Duration) *AuctionCloseWorker {
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
	w.mode = WorkerModeAcquiring
	return w
}

func (w *AuctionCloseWorker) Start(ctx context.Context) {
	if w == nil || w.usecase == nil {
		return
	}
	w.mu.Lock()
	w.started = true
	w.mu.Unlock()
	go func() {
		var lease Lease
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

func (w *AuctionCloseWorker) Ping(context.Context) error {
	if w == nil || w.usecase == nil {
		return errors.New("auction close worker is not initialized")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.started {
		return errors.New("auction close worker is not started")
	}
	return nil
}

func (w *AuctionCloseWorker) WorkerStatus(ctx context.Context) WorkerStatus {
	w.mu.Lock()
	status := WorkerStatus{
		Name:          "auction_close_worker",
		Mode:          w.mode,
		InstanceID:    w.instanceID,
		LeaseKey:      w.leaseKey,
		LeaseOwner:    w.leaseOwner,
		Started:       w.started,
		LastAttemptAt: formatWorkerTime(w.lastAttemptAt),
		LastSuccessAt: formatWorkerTime(w.lastSuccessAt),
		LastError:     w.lastError,
		Extra: map[string]any{
			"last_summary": w.lastSummary,
		},
	}
	w.mu.Unlock()
	if status.LeaseOwner == "" && w.leaseProvider != nil && w.leaseKey != "" {
		if owner, err := w.leaseProvider.Owner(ctx, w.leaseKey); err == nil {
			status.LeaseOwner = owner
		}
	}
	return status
}

func (w *AuctionCloseWorker) RunOnce(ctx context.Context) (CloseExpiredSummary, error) {
	return w.runOnce(ctx)
}

func (w *AuctionCloseWorker) tick(ctx context.Context, lease *Lease, lastRun *time.Time) {
	if !w.ensureLease(ctx, lease) {
		return
	}
	if !lastRun.IsZero() && time.Since(*lastRun) < w.interval {
		return
	}
	*lastRun = time.Now()
	if _, err := w.runOnce(ctx); err != nil && w.leaseProvider != nil && lease != nil && *lease != nil {
		w.setLeaseMode(WorkerModeLeaderDegraded, (*lease).Owner())
		_ = (*lease).Release(ctx)
		*lease = nil
	}
}

func (w *AuctionCloseWorker) ensureLease(ctx context.Context, lease *Lease) bool {
	if w.leaseProvider == nil {
		w.setLeaseMode(WorkerModeLeader, w.instanceID)
		return true
	}
	if lease == nil || *lease == nil {
		w.setLeaseMode(WorkerModeAcquiring, "")
		acquired, ok, err := w.leaseProvider.TryAcquire(ctx, w.leaseKey, w.instanceID, w.leaseTTL)
		if err != nil {
			w.recordError(time.Now(), err)
			return false
		}
		if !ok {
			owner, _ := w.leaseProvider.Owner(ctx, w.leaseKey)
			w.setLeaseMode(WorkerModeStandby, owner)
			return false
		}
		*lease = acquired
		w.setLeaseMode(WorkerModeLeader, acquired.Owner())
		slog.Info("auction close worker acquired lease", "lease_key", w.leaseKey, "owner", w.instanceID)
		return true
	}
	ok, err := (*lease).Renew(ctx)
	if err != nil || !ok {
		if err != nil {
			w.recordError(time.Now(), err)
		}
		w.setLeaseMode(WorkerModeLostLease, "")
		*lease = nil
		slog.Warn("auction close worker lost lease", "lease_key", w.leaseKey, "owner", w.instanceID, "error", err)
		return false
	}
	w.setLeaseMode(WorkerModeLeader, (*lease).Owner())
	return true
}

func (w *AuctionCloseWorker) runOnce(ctx context.Context) (CloseExpiredSummary, error) {
	now := time.Now()
	summary, err := w.usecase.CloseExpiredLots(ctx, now.UnixMilli(), w.limit)
	w.mu.Lock()
	w.lastAttemptAt = now
	w.lastSummary = summary
	if err != nil {
		message := err.Error()
		if len(message) > 512 {
			message = message[:512]
		}
		w.lastError = message
		w.mu.Unlock()
		slog.Error("auction close worker scan failed", "error", err, "summary", summary)
		return summary, err
	}
	w.lastSuccessAt = now
	w.lastError = ""
	w.mu.Unlock()
	if summary.Closed > 0 || summary.Conflicts > 0 {
		slog.Info("auction close worker closed expired lots", "scanned", summary.Scanned, "closed", summary.Closed, "settled", summary.Settled, "failed", summary.Failed, "conflicts", summary.Conflicts)
	}
	return summary, nil
}

func (w *AuctionCloseWorker) setLeaseMode(mode WorkerMode, owner string) {
	w.mu.Lock()
	w.mode = mode
	w.leaseOwner = owner
	w.mu.Unlock()
	observability.SetWorkerLeaseActive("auction_close_worker", mode == WorkerModeLeader || mode == WorkerModeLeaderDegraded)
}

func (w *AuctionCloseWorker) recordError(now time.Time, err error) {
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

func (w *AuctionCloseWorker) String() string {
	if w == nil {
		return "auction close worker <nil>"
	}
	status := w.WorkerStatus(context.Background())
	return fmt.Sprintf("%s %s owner=%s", status.Name, status.Mode, status.LeaseOwner)
}
