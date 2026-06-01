package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	"live-auction-bid/backend/app/auction/service/internal/observability"
	"live-auction-bid/backend/app/auction/service/internal/pkg/apperr"
)

const runtimeProjectionGroup = "auction-runtime-projector"

type RuntimeProjectionMetrics struct {
	projectionPendingCount atomic.Int64
	projectionLagMs        atomic.Int64
	projectionFailedTotal  atomic.Int64
	projectionGapTotal     atomic.Int64
}

func (m *RuntimeProjectionMetrics) Snapshot(xaddTotal int64) map[string]any {
	return map[string]any{
		"runtime_event_xadd_total": xaddTotal,
		"projection_pending_count": m.projectionPendingCount.Load(),
		"projection_lag_ms":        m.projectionLagMs.Load(),
		"projection_failed_total":  m.projectionFailedTotal.Load(),
		"projection_gap_total":     m.projectionGapTotal.Load(),
	}
}

type RuntimeProjectionWorker struct {
	store                *Store
	publisher            auction.EventPublisher
	interval             time.Duration
	limit                int
	consumer             string
	metrics              RuntimeProjectionMetrics
	shards               int
	projectLegacyStreams bool
	maxDrainBatches      int

	leaseProvider auction.LeaseProvider
	instanceID    string
	leaseTTL      time.Duration
	renewInterval time.Duration

	mu            sync.Mutex
	started       bool
	lastAttemptAt time.Time
	lastSuccessAt time.Time
	lastError     string
	shardStates   map[int]*runtimeProjectionShardState
}

type runtimeProjectionShardState struct {
	lease         auction.Lease
	mode          auction.WorkerMode
	leaseKey      string
	leaseOwner    string
	lastRun       time.Time
	lastAttemptAt time.Time
	lastSuccessAt time.Time
	lastError     string
}

func NewRuntimeProjectionWorker(store *Store, publisher auction.EventPublisher, interval time.Duration, limit int) *RuntimeProjectionWorker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if limit <= 0 {
		limit = 100
	}
	maxDrainBatches := projectionEnvInt("AUCTION_RUNTIME_PROJECTION_DRAIN_BATCHES", 8)
	if maxDrainBatches <= 0 {
		maxDrainBatches = 8
	}
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local"
	}
	shards := defaultRuntimeProjectionShards
	if store != nil && store.runtimeProjectionShards > 0 {
		shards = store.runtimeProjectionShards
	}
	return &RuntimeProjectionWorker{
		store:                store,
		publisher:            publisher,
		interval:             interval,
		limit:                limit,
		consumer:             fmt.Sprintf("%s-%d", hostname, os.Getpid()),
		shards:               shards,
		projectLegacyStreams: projectionEnvBool("AUCTION_RUNTIME_PROJECT_LEGACY_STREAMS", false),
		maxDrainBatches:      maxDrainBatches,
		shardStates:          make(map[int]*runtimeProjectionShardState),
	}
}

func projectionEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func projectionEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (w *RuntimeProjectionWorker) BindLease(provider auction.LeaseProvider, instanceID string, ttl, renewInterval time.Duration) *RuntimeProjectionWorker {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.leaseProvider = provider
	w.instanceID = instanceID
	w.leaseTTL = normalizeLeaseTTL(ttl)
	w.renewInterval = normalizeRenewInterval(renewInterval)
	if w.shards <= 0 {
		w.shards = defaultRuntimeProjectionShards
	}
	for shard := 0; shard < w.shards; shard++ {
		state := w.ensureShardStateLocked(shard)
		state.mode = auction.WorkerModeAcquiring
	}
	return w
}

func (w *RuntimeProjectionWorker) Start(ctx context.Context) {
	w.mu.Lock()
	w.started = true
	w.mu.Unlock()

	go func() {
		ticker := time.NewTicker(minWorkerInterval(w.interval, w.renewInterval))
		defer ticker.Stop()
		for {
			w.project(ctx)
			select {
			case <-ctx.Done():
				w.releaseShardLeases(context.Background())
				return
			case <-ticker.C:
			}
		}
	}()
}

func (w *RuntimeProjectionWorker) Ping(context.Context) error {
	if w == nil || w.store == nil {
		return errors.New("runtime projection worker is not initialized")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.started {
		return errors.New("runtime projection worker is not started")
	}
	return nil
}

func (w *RuntimeProjectionWorker) WorkerStatus(ctx context.Context) auction.WorkerStatus {
	w.mu.Lock()
	status := auction.WorkerStatus{
		Name:          "runtime_projection_worker",
		Mode:          auction.WorkerModeStandby,
		InstanceID:    w.instanceID,
		Started:       w.started,
		LastAttemptAt: formatWorkerTime(w.lastAttemptAt),
		LastSuccessAt: formatWorkerTime(w.lastSuccessAt),
		LastError:     w.lastError,
	}
	owned := make([]int, 0)
	standby := make([]int, 0)
	failed := make([]int, 0)
	shards := make([]auction.WorkerShardState, 0, w.shards)
	for shard := 0; shard < w.shards; shard++ {
		state := w.ensureShardStateLocked(shard)
		if state.mode == auction.WorkerModeLeader || state.mode == auction.WorkerModeLeaderDegraded {
			owned = append(owned, shard)
		} else {
			standby = append(standby, shard)
		}
		if state.lastError != "" {
			failed = append(failed, shard)
		}
		shards = append(shards, auction.WorkerShardState{
			ShardID:       shard,
			Mode:          state.mode,
			LeaseKey:      state.leaseKey,
			LeaseOwner:    state.leaseOwner,
			LastAttemptAt: formatWorkerTime(state.lastAttemptAt),
			LastSuccessAt: formatWorkerTime(state.lastSuccessAt),
			LastError:     state.lastError,
		})
	}
	status.OwnedShards = owned
	status.StandbyShards = standby
	status.FailedShards = failed
	status.Shards = shards
	if len(owned) > 0 {
		status.Mode = auction.WorkerModeLeader
		if len(owned) < w.shards {
			status.Mode = auction.WorkerModePartialOwner
		}
	}
	w.mu.Unlock()
	for i := range status.Shards {
		if status.Shards[i].LeaseOwner == "" && w.leaseProvider != nil && status.Shards[i].LeaseKey != "" {
			if owner, err := w.leaseProvider.Owner(ctx, status.Shards[i].LeaseKey); err == nil {
				status.Shards[i].LeaseOwner = owner
			}
		}
	}
	return status
}

func (w *RuntimeProjectionWorker) MetricsSnapshot(ctx context.Context) map[string]any {
	xaddTotal, _ := w.store.redis.Get(ctx, runtimeMetricKey("runtime_event_xadd_total")).Int64()
	w.syncPrometheusMetrics(xaddTotal)
	return w.metrics.Snapshot(xaddTotal)
}

func (w *RuntimeProjectionWorker) project(ctx context.Context) {
	now := time.Now()
	err := w.projectReadyShards(ctx)
	xaddTotal, _ := w.store.redis.Get(ctx, runtimeMetricKey("runtime_event_xadd_total")).Int64()
	w.syncPrometheusMetrics(xaddTotal)
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

func (w *RuntimeProjectionWorker) projectOnce(ctx context.Context) error {
	return w.projectReadyShards(ctx)
}

func (w *RuntimeProjectionWorker) projectReadyShards(ctx context.Context) error {
	if w == nil || w.store == nil || w.store.redis == nil {
		return errors.New("runtime projection store is required")
	}
	totalPending := int64(0)
	var firstErr error
	for shard := 0; shard < w.shards; shard++ {
		if !w.ensureShardLease(ctx, shard) {
			continue
		}
		shardPending := int64(0)
		var shardErr error
		for batch := 0; batch < w.maxDrainBatches; batch++ {
			batchStartedAt := time.Now()
			pending, err := w.projectShardOnce(ctx, shard)
			observability.RecordProjectionBatchDuration(shard, time.Since(batchStartedAt))
			shardPending += pending
			if err != nil {
				shardErr = err
				break
			}
			if pending < int64(w.limit) {
				break
			}
		}
		w.recordShardAttempt(shard, shardErr)
		if shardErr != nil {
			w.releaseShardLease(ctx, shard)
			if firstErr == nil {
				firstErr = shardErr
			}
			totalPending += shardPending
			continue
		}
		totalPending += shardPending
		if shard == 0 && w.projectLegacyStreams {
			if err := w.projectLegacyRoomStreams(ctx); err != nil {
				w.recordShardAttempt(shard, err)
				w.releaseShardLease(ctx, shard)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
		}
	}
	w.metrics.projectionPendingCount.Store(totalPending)
	w.syncPrometheusMetricsFromStore(ctx)
	return firstErr
}

func (w *RuntimeProjectionWorker) projectLegacyRoomStreams(ctx context.Context) error {
	roomIDs, err := w.store.redis.SMembers(ctx, runtimeEventRoomsKey()).Result()
	if err != nil {
		return err
	}
	for _, roomID := range roomIDs {
		if roomID == "" {
			continue
		}
		stream := runtimeEventStreamKey(roomID)
		if err := w.ensureGroup(ctx, stream); err != nil {
			return err
		}
		pending, err := w.projectClaimed(ctx, stream)
		if err != nil {
			return err
		}
		_ = pending
		if err := w.projectNew(ctx, stream); err != nil {
			return err
		}
	}
	return nil
}

func (w *RuntimeProjectionWorker) projectShardOnce(ctx context.Context, shard int) (int64, error) {
	stream := runtimeEventShardStreamKey(shard)
	offset, err := w.store.runtimeProjectionShardOffset(ctx, shard)
	if err != nil {
		return 0, err
	}
	start := "(0-0"
	if offset.LastStreamID != "" {
		start = "(" + offset.LastStreamID
	}
	messages, err := w.store.redis.XRangeN(ctx, stream, start, "+", int64(w.limit)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var processed int64
	lastID := ""
	for _, message := range messages {
		if err := w.projectMessage(ctx, stream, message, false); err != nil {
			if lastID != "" {
				_ = w.store.saveRuntimeProjectionShardOffset(ctx, shard, lastID, time.Now().UnixMilli())
			}
			return processed, err
		}
		processed++
		lastID = message.ID
	}
	if lastID != "" {
		if err := w.store.saveRuntimeProjectionShardOffset(ctx, shard, lastID, time.Now().UnixMilli()); err != nil {
			return processed, err
		}
	}
	return processed, nil
}

func (w *RuntimeProjectionWorker) ensureShardLease(ctx context.Context, shard int) bool {
	if w.leaseProvider == nil {
		w.setShardMode(shard, auction.WorkerModeLeader, w.instanceID, nil)
		return true
	}
	w.mu.Lock()
	state := w.ensureShardStateLocked(shard)
	if state.lease == nil {
		state.mode = auction.WorkerModeAcquiring
		state.leaseOwner = ""
		w.mu.Unlock()
		lease, ok, err := w.leaseProvider.TryAcquire(ctx, state.leaseKey, w.instanceID, w.leaseTTL)
		if err != nil {
			w.setShardMode(shard, auction.WorkerModeStandby, "", err)
			return false
		}
		if !ok {
			owner, _ := w.leaseProvider.Owner(ctx, state.leaseKey)
			w.setShardMode(shard, auction.WorkerModeStandby, owner, nil)
			return false
		}
		w.mu.Lock()
		state = w.ensureShardStateLocked(shard)
		state.lease = lease
		state.mode = auction.WorkerModeLeader
		state.leaseOwner = lease.Owner()
		w.mu.Unlock()
		observability.SetWorkerLeaseActive("runtime_projection_worker", true)
		slog.Info("runtime projection worker acquired shard lease", "shard", shard, "lease_key", lease.Key(), "owner", w.instanceID)
		return true
	}
	lease := state.lease
	w.mu.Unlock()
	ok, err := lease.Renew(ctx)
	if err != nil || !ok {
		w.setShardMode(shard, auction.WorkerModeLostLease, "", err)
		w.mu.Lock()
		w.ensureShardStateLocked(shard).lease = nil
		w.mu.Unlock()
		slog.Warn("runtime projection worker lost shard lease", "shard", shard, "lease_key", lease.Key(), "owner", w.instanceID, "error", err)
		return false
	}
	w.setShardMode(shard, auction.WorkerModeLeader, lease.Owner(), nil)
	return true
}

func (w *RuntimeProjectionWorker) releaseShardLease(ctx context.Context, shard int) {
	w.mu.Lock()
	state := w.ensureShardStateLocked(shard)
	lease := state.lease
	state.lease = nil
	if state.mode == auction.WorkerModeLeader {
		state.mode = auction.WorkerModeLeaderDegraded
	}
	w.mu.Unlock()
	if lease != nil {
		_ = lease.Release(ctx)
	}
	observability.SetWorkerLeaseActive("runtime_projection_worker", w.hasLeaderShard())
}

func (w *RuntimeProjectionWorker) releaseShardLeases(ctx context.Context) {
	for shard := 0; shard < w.shards; shard++ {
		w.releaseShardLease(ctx, shard)
	}
}

func (w *RuntimeProjectionWorker) recordShardAttempt(shard int, err error) {
	now := time.Now()
	w.mu.Lock()
	state := w.ensureShardStateLocked(shard)
	state.lastRun = now
	state.lastAttemptAt = now
	w.lastAttemptAt = now
	if err != nil {
		message := err.Error()
		if len(message) > 512 {
			message = message[:512]
		}
		state.lastError = message
		w.lastError = message
		w.mu.Unlock()
		return
	}
	state.lastSuccessAt = now
	state.lastError = ""
	w.lastSuccessAt = now
	w.lastError = ""
	w.mu.Unlock()
}

func (w *RuntimeProjectionWorker) setShardMode(shard int, mode auction.WorkerMode, owner string, err error) {
	w.mu.Lock()
	state := w.ensureShardStateLocked(shard)
	state.mode = mode
	state.leaseOwner = owner
	if err != nil {
		message := err.Error()
		if len(message) > 512 {
			message = message[:512]
		}
		state.lastAttemptAt = time.Now()
		state.lastError = message
		w.lastError = message
	}
	w.mu.Unlock()
	observability.SetWorkerLeaseActive("runtime_projection_worker", w.hasLeaderShard())
}

func (w *RuntimeProjectionWorker) hasLeaderShard() bool {
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for shard := 0; shard < w.shards; shard++ {
		state := w.ensureShardStateLocked(shard)
		if state.mode == auction.WorkerModeLeader || state.mode == auction.WorkerModeLeaderDegraded {
			return true
		}
	}
	return false
}

func (w *RuntimeProjectionWorker) syncPrometheusMetricsFromStore(ctx context.Context) {
	if w == nil || w.store == nil || w.store.redis == nil {
		return
	}
	xaddTotal, _ := w.store.redis.Get(ctx, runtimeMetricKey("runtime_event_xadd_total")).Int64()
	w.syncPrometheusMetrics(xaddTotal)
}

func (w *RuntimeProjectionWorker) syncPrometheusMetrics(xaddTotal int64) {
	if w == nil {
		return
	}
	observability.SetRuntimeProjectionMetrics(
		xaddTotal,
		w.metrics.projectionPendingCount.Load(),
		w.metrics.projectionLagMs.Load(),
		w.metrics.projectionFailedTotal.Load(),
		w.metrics.projectionGapTotal.Load(),
	)
}

func (w *RuntimeProjectionWorker) getShardState(shard int) *runtimeProjectionShardState {
	w.mu.Lock()
	defer w.mu.Unlock()
	state := w.ensureShardStateLocked(shard)
	copyState := *state
	return &copyState
}

func (w *RuntimeProjectionWorker) ensureShardStateLocked(shard int) *runtimeProjectionShardState {
	if w.shardStates == nil {
		w.shardStates = make(map[int]*runtimeProjectionShardState)
	}
	state := w.shardStates[shard]
	if state == nil {
		state = &runtimeProjectionShardState{
			mode:     auction.WorkerModeAcquiring,
			leaseKey: runtimeProjectionShardLeaseKey(shard),
		}
		w.shardStates[shard] = state
	}
	return state
}

func (w *RuntimeProjectionWorker) ensureGroup(ctx context.Context, stream string) error {
	err := w.store.redis.XGroupCreateMkStream(ctx, stream, runtimeProjectionGroup, "0").Err()
	if err == nil || strings.Contains(err.Error(), "BUSYGROUP") {
		return nil
	}
	return err
}

func (w *RuntimeProjectionWorker) projectClaimed(ctx context.Context, stream string) (int64, error) {
	var processed int64
	start := "0-0"
	for {
		messages, next, err := w.store.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    runtimeProjectionGroup,
			Consumer: w.consumer,
			MinIdle:  5 * time.Second,
			Start:    start,
			Count:    int64(w.limit),
		}).Result()
		if err != nil {
			return processed, err
		}
		if len(messages) == 0 {
			return processed, nil
		}
		processed += int64(len(messages))
		for _, message := range messages {
			if err := w.projectMessage(ctx, stream, message, true); err != nil {
				return processed, err
			}
		}
		if next == "" || next == "0-0" {
			return processed, nil
		}
		start = next
	}
}

func (w *RuntimeProjectionWorker) projectNew(ctx context.Context, stream string) error {
	streams, err := w.store.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    runtimeProjectionGroup,
		Consumer: w.consumer,
		Streams:  []string{stream, ">"},
		Count:    int64(w.limit),
		Block:    time.Millisecond,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, result := range streams {
		for _, message := range result.Messages {
			if err := w.projectMessage(ctx, stream, message, true); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *RuntimeProjectionWorker) projectMessage(ctx context.Context, stream string, message redis.XMessage, ack bool) error {
	projection, err := w.decodeRuntimeProjection(ctx, message)
	if err != nil {
		w.metrics.projectionFailedTotal.Add(1)
		slog.Error("runtime projection decode failed", "stream", stream, "id", message.ID, "error", err)
		return err
	}
	events, order, err := auction.BuildRuntimeBidProjectionArtifacts(projection)
	if err != nil {
		w.metrics.projectionFailedTotal.Add(1)
		return err
	}
	outcome, err := w.store.ProjectRuntimeEvent(ctx, projection, order, events)
	if err != nil {
		w.metrics.projectionFailedTotal.Add(1)
		if apperr.IsRuntimeProjectionGap(err) {
			w.metrics.projectionGapTotal.Add(1)
			slog.Warn("runtime projection gap",
				"stream", stream,
				"stream_id", message.ID,
				"lot_id", projection.LotID,
				"previous_lot_version", projection.PreviousLotVersion,
				"lot_version", projection.LotVersion,
				"runtime_event_id", projection.RuntimeEventID,
				"error", err,
			)
			return fmt.Errorf("%w: stream=%s stream_id=%s lot_id=%s previous_lot_version=%d lot_version=%d",
				err, stream, message.ID, projection.LotID, projection.PreviousLotVersion, projection.LotVersion)
		}
		slog.Warn("runtime projection failed",
			"stream", stream,
			"stream_id", message.ID,
			"lot_id", projection.LotID,
			"previous_lot_version", projection.PreviousLotVersion,
			"lot_version", projection.LotVersion,
			"runtime_event_id", projection.RuntimeEventID,
			"error", err,
		)
		return err
	}
	if ack {
		if err := w.store.redis.XAck(ctx, stream, runtimeProjectionGroup, message.ID).Err(); err != nil {
			w.metrics.projectionFailedTotal.Add(1)
			return err
		}
	}
	if projection.OccurredAtUnixMs > 0 {
		w.metrics.projectionLagMs.Store(time.Now().UnixMilli() - projection.OccurredAtUnixMs)
	}
	if !outcome.Projected || w.publisher == nil {
		return nil
	}
	for _, event := range events {
		if err := w.publisher.Publish(ctx, event); err != nil {
			w.metrics.projectionFailedTotal.Add(1)
			return err
		}
	}
	return nil
}

func (w *RuntimeProjectionWorker) decodeRuntimeProjection(ctx context.Context, message redis.XMessage) (auction.RuntimeProjectionEvent, error) {
	rawPayload, ok := message.Values["payload"]
	if !ok {
		return auction.RuntimeProjectionEvent{}, errors.New("runtime projection payload is missing")
	}
	payloadText := fmt.Sprint(rawPayload)
	var payload runtimeProjectionEventPayload
	if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
		return auction.RuntimeProjectionEvent{}, err
	}
	baseLot, err := w.store.FindCoreByID(ctx, payload.LotID)
	if err != nil {
		return auction.RuntimeProjectionEvent{}, err
	}
	lot := runtimeJSONToLot(baseLot, payload.UpdatedLot)
	bid := runtimeJSONToBid(payload.Bid)
	ranking := runtimeJSONToRanking(payload.RankingTop)
	return auction.RuntimeProjectionEvent{
		RuntimeEventID:     payload.EventID,
		RuntimeStreamID:    message.ID,
		RoomID:             payload.RoomID,
		LotID:              payload.LotID,
		EventType:          payload.EventType,
		IdempotencyKey:     payload.IdempotencyKey,
		Bid:                *bid,
		Lot:                lot,
		Ranking:            ranking,
		PreviousLeaderID:   payload.PreviousLeaderID,
		EndsBeforeBid:      payload.EndsBeforeBid,
		ExtendCountBefore:  payload.ExtendCountBefore,
		PreviousLotVersion: payload.PreviousLotVersion,
		LotVersion:         payload.LotVersion,
		OccurredAtUnixMs:   payload.OccurredAtUnixMs,
		OrderID:            payload.OrderID,
	}, nil
}

func RuntimeProjectionMetricsFromMap(snapshot map[string]any) map[string]string {
	out := make(map[string]string, len(snapshot))
	for key, value := range snapshot {
		switch typed := value.(type) {
		case int64:
			out[key] = strconv.FormatInt(typed, 10)
		default:
			out[key] = fmt.Sprint(value)
		}
	}
	return out
}
