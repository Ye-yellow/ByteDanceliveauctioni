package auction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type AuctionCloseWorker struct {
	usecase  *AuctionUsecase
	interval time.Duration
	limit    int

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
	return &AuctionCloseWorker{usecase: usecase, interval: interval, limit: limit}
}

func (w *AuctionCloseWorker) Start(ctx context.Context) {
	if w == nil || w.usecase == nil {
		return
	}
	w.mu.Lock()
	w.started = true
	w.mu.Unlock()
	go func() {
		w.runOnce(ctx)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.runOnce(ctx)
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
	if w.lastSuccessAt.IsZero() {
		if w.lastError != "" {
			return fmt.Errorf("auction close worker has no successful scan: %s", w.lastError)
		}
		return errors.New("auction close worker has no successful scan")
	}
	if w.lastError != "" && w.lastAttemptAt.After(w.lastSuccessAt) {
		return fmt.Errorf("auction close worker last scan failed: %s", w.lastError)
	}
	if time.Since(w.lastSuccessAt) > w.interval*3 {
		return fmt.Errorf("auction close worker stale: last success at %s", w.lastSuccessAt.Format(time.RFC3339))
	}
	return nil
}

func (w *AuctionCloseWorker) RunOnce(ctx context.Context) (CloseExpiredSummary, error) {
	return w.runOnce(ctx)
}

func (w *AuctionCloseWorker) runOnce(ctx context.Context) (CloseExpiredSummary, error) {
	now := time.Now()
	summary, err := w.usecase.CloseExpiredLots(ctx, now.UnixMilli(), w.limit)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastAttemptAt = now
	w.lastSummary = summary
	if err != nil {
		message := err.Error()
		if len(message) > 512 {
			message = message[:512]
		}
		w.lastError = message
		slog.Error("auction close worker scan failed", "error", err, "summary", summary)
		return summary, err
	}
	w.lastSuccessAt = now
	w.lastError = ""
	if summary.Closed > 0 || summary.Conflicts > 0 {
		slog.Info("auction close worker closed expired lots", "scanned", summary.Scanned, "closed", summary.Closed, "settled", summary.Settled, "failed", summary.Failed, "conflicts", summary.Conflicts)
	}
	return summary, nil
}
