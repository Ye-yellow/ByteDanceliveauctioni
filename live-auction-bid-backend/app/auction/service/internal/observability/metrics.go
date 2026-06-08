package observability

import (
	"database/sql"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	projectionPendingCount atomic.Int64
	projectionLagMs        atomic.Int64
	projectionFailedTotal  atomic.Int64
	projectionGapTotal     atomic.Int64
	runtimeEventXAddTotal  atomic.Int64
	activeLotMu            sync.Mutex
	activeLotCounts        = make(map[string]int64)
	statsMu                sync.RWMutex
	dbStatsProvider        func() sql.DBStats
	redisStatsProvider     func() RedisPoolStats

	bidRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_bid_requests_total",
		Help: "Total auction bid requests by result and stable reason.",
	}, []string{"result", "reason"})
	bidLatencyMs = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "auction_bid_latency_ms",
		Help:    "Auction bid request latency in milliseconds.",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"result"})
	bidAccepted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auction_bid_accepted_total",
		Help: "Total accepted auction bids.",
	})
	bidRejected = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_bid_rejected_total",
		Help: "Total rejected auction bids by stable reason.",
	}, []string{"reason"})
	wsConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "auction_ws_connections",
		Help: "Current websocket connections by room and scope.",
	}, []string{"room_id", "scope"})
	wsEventsSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_ws_events_sent_total",
		Help: "Total websocket auction events sent by type.",
	}, []string{"type"})
	wsEventsCoalesced = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_ws_events_coalesced_total",
		Help: "Total websocket auction events replaced by room-level coalescing.",
	}, []string{"type"})
	wsEventsDropped = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_ws_events_dropped_total",
		Help: "Total websocket auction events dropped because a client send queue was full.",
	}, []string{"type"})
	outboxPendingCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "auction_outbox_pending_count",
		Help: "Current number of persisted auction events pending Redis stream delivery.",
	})
	activeLots = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "auction_active_lots",
		Help: "Event-derived count of currently active lots by room.",
	}, []string{"room_id"})
	workerLeaseActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "auction_worker_lease_active",
		Help: "Whether this instance currently owns a worker lease.",
	}, []string{"worker"})
	orderCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auction_order_created_total",
		Help: "Total order-created auction events broadcast after settlement.",
	})
	projectionBatchDurationMs = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "auction_projection_batch_duration_ms",
		Help:    "Runtime projection shard batch duration in milliseconds.",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"shard"})
)

type RedisPoolStats struct {
	Hits       uint32
	Misses     uint32
	Timeouts   uint32
	TotalConns uint32
	IdleConns  uint32
	StaleConns uint32
}

func init() {
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "auction_projection_lag_ms",
		Help: "Latest runtime projection lag in milliseconds.",
	}, func() float64 {
		return float64(projectionLagMs.Load())
	}))
	prometheus.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: "auction_projection_failed_total",
		Help: "Total runtime projection failures.",
	}, func() float64 {
		return float64(projectionFailedTotal.Load())
	}))
	prometheus.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: "auction_projection_gap_total",
		Help: "Total runtime projection version gaps.",
	}, func() float64 {
		return float64(projectionGapTotal.Load())
	}))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "auction_projection_pending_count",
		Help: "Current runtime projection pending count.",
	}, func() float64 {
		return float64(projectionPendingCount.Load())
	}))
	prometheus.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: "auction_runtime_event_xadd_total",
		Help: "Total runtime events atomically appended to Redis streams.",
	}, func() float64 {
		return float64(runtimeEventXAddTotal.Load())
	}))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "auction_db_pool_open_connections",
		Help: "Current open MySQL connections.",
	}, func() float64 { return float64(currentDBStats().OpenConnections) }))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "auction_db_pool_in_use",
		Help: "Current in-use MySQL connections.",
	}, func() float64 { return float64(currentDBStats().InUse) }))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "auction_db_pool_idle",
		Help: "Current idle MySQL connections.",
	}, func() float64 { return float64(currentDBStats().Idle) }))
	prometheus.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: "auction_db_pool_wait_count_total",
		Help: "Total MySQL connection pool waits.",
	}, func() float64 { return float64(currentDBStats().WaitCount) }))
	prometheus.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: "auction_db_pool_wait_duration_ms_total",
		Help: "Total MySQL connection pool wait duration in milliseconds.",
	}, func() float64 { return float64(currentDBStats().WaitDuration.Milliseconds()) }))
	prometheus.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: "auction_redis_pool_hits_total",
		Help: "Total Redis pool hits.",
	}, func() float64 { return float64(currentRedisStats().Hits) }))
	prometheus.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: "auction_redis_pool_misses_total",
		Help: "Total Redis pool misses.",
	}, func() float64 { return float64(currentRedisStats().Misses) }))
	prometheus.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{
		Name: "auction_redis_pool_timeouts_total",
		Help: "Total Redis pool timeouts.",
	}, func() float64 { return float64(currentRedisStats().Timeouts) }))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "auction_redis_pool_total_conns",
		Help: "Current Redis pool total connections.",
	}, func() float64 { return float64(currentRedisStats().TotalConns) }))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "auction_redis_pool_idle_conns",
		Help: "Current Redis pool idle connections.",
	}, func() float64 { return float64(currentRedisStats().IdleConns) }))
}

func BindDBStatsProvider(provider func() sql.DBStats) {
	statsMu.Lock()
	defer statsMu.Unlock()
	dbStatsProvider = provider
}

func BindRedisPoolStatsProvider(provider func() RedisPoolStats) {
	statsMu.Lock()
	defer statsMu.Unlock()
	redisStatsProvider = provider
}

func currentDBStats() sql.DBStats {
	statsMu.RLock()
	provider := dbStatsProvider
	statsMu.RUnlock()
	if provider == nil {
		return sql.DBStats{}
	}
	return provider()
}

func currentRedisStats() RedisPoolStats {
	statsMu.RLock()
	provider := redisStatsProvider
	statsMu.RUnlock()
	if provider == nil {
		return RedisPoolStats{}
	}
	return provider()
}

func RecordBid(result, reason string, duration time.Duration) {
	result = cleanLabel(result, "unknown")
	reason = cleanLabel(reason, "unknown")
	bidRequests.WithLabelValues(result, reason).Inc()
	bidLatencyMs.WithLabelValues(result).Observe(float64(duration.Milliseconds()))
	switch result {
	case "accepted":
		bidAccepted.Inc()
	case "rejected":
		bidRejected.WithLabelValues(reason).Inc()
	}
}

func IncWSConnection(roomID, scope string) {
	wsConnections.WithLabelValues(cleanLabel(roomID, "unknown"), cleanLabel(scope, "unknown")).Inc()
}

func DecWSConnection(roomID, scope string) {
	wsConnections.WithLabelValues(cleanLabel(roomID, "unknown"), cleanLabel(scope, "unknown")).Dec()
}

func RecordWSEventSent(eventType string) {
	wsEventsSent.WithLabelValues(cleanLabel(eventType, "unknown")).Inc()
}

func RecordWSEventCoalesced(eventType string) {
	wsEventsCoalesced.WithLabelValues(cleanLabel(eventType, "unknown")).Inc()
}

func RecordWSEventDropped(eventType string) {
	wsEventsDropped.WithLabelValues(cleanLabel(eventType, "unknown")).Inc()
}

func SetRuntimeProjectionMetrics(xaddTotal, pending, lagMs, failedTotal, gapTotal int64) {
	runtimeEventXAddTotal.Store(nonNegative(xaddTotal))
	projectionPendingCount.Store(nonNegative(pending))
	projectionLagMs.Store(nonNegative(lagMs))
	projectionFailedTotal.Store(nonNegative(failedTotal))
	projectionGapTotal.Store(nonNegative(gapTotal))
}

func RecordProjectionBatchDuration(shard int, duration time.Duration) {
	projectionBatchDurationMs.WithLabelValues(strconv.Itoa(shard)).Observe(float64(duration.Milliseconds()))
}

func SetOutboxPendingCount(count int64) {
	outboxPendingCount.Set(float64(nonNegative(count)))
}

func AddActiveLots(roomID string, delta int) {
	roomID = cleanLabel(roomID, "unknown")
	activeLotMu.Lock()
	defer activeLotMu.Unlock()
	next := activeLotCounts[roomID] + int64(delta)
	if next < 0 {
		next = 0
	}
	activeLotCounts[roomID] = next
	activeLots.WithLabelValues(roomID).Set(float64(next))
}

func SetWorkerLeaseActive(worker string, active bool) {
	value := 0.0
	if active {
		value = 1
	}
	workerLeaseActive.WithLabelValues(cleanLabel(worker, "unknown")).Set(value)
}

func IncOrderCreated() {
	orderCreated.Inc()
}

func cleanLabel(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if len(value) > 96 {
		return value[:96]
	}
	return value
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
