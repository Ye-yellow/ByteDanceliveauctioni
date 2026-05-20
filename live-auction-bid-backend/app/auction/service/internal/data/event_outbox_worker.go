package data

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// EventOutboxWorker owns the production Redis Stream repair loop for persisted auction events.
// It is observable via Ping so readiness can fail when event delivery repair is stale or broken.
type EventOutboxWorker struct {
	store    *Store
	interval time.Duration
	limit    int

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
	return &EventOutboxWorker{store: store, interval: interval, limit: limit}
}

func (w *EventOutboxWorker) Start(ctx context.Context) {
	w.mu.Lock()
	w.started = true
	w.mu.Unlock()

	w.repair(ctx)
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.repair(ctx)
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
	if w.lastSuccessAt.IsZero() {
		if w.lastError != "" {
			return fmt.Errorf("event outbox worker has no successful repair: %s", w.lastError)
		}
		return errors.New("event outbox worker has no successful repair")
	}
	if w.lastError != "" && w.lastAttemptAt.After(w.lastSuccessAt) {
		return fmt.Errorf("event outbox worker last repair failed: %s", w.lastError)
	}
	if time.Since(w.lastSuccessAt) > w.interval*3 {
		return fmt.Errorf("event outbox worker stale: last success at %s", w.lastSuccessAt.Format(time.RFC3339))
	}
	return nil
}

func (w *EventOutboxWorker) repair(ctx context.Context) {
	now := time.Now()
	err := w.store.RepairEventStreamOutbox(ctx, w.limit)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastAttemptAt = now
	if err != nil {
		message := err.Error()
		if len(message) > 512 {
			message = message[:512]
		}
		w.lastError = message
		return
	}
	w.lastSuccessAt = now
	w.lastError = ""
}
