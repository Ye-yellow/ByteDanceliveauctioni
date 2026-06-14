package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
)

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type RuntimeProjectionMetricsProvider interface {
	MetricsSnapshot(ctx context.Context) map[string]any
}

type WorkerSnapshotProvider interface {
	WorkerSnapshot(ctx context.Context) map[string]any
}

type ClusterSnapshotProvider interface {
	ClusterSnapshot(ctx context.Context) map[string]any
}

type Readiness struct {
	Store             HealthChecker
	Outbox            HealthChecker
	AuctionClose      HealthChecker
	RuntimeProjection HealthChecker
	ProjectionMetrics RuntimeProjectionMetricsProvider
	WorkerStatuses    []auction.WorkerStatusProvider
	Cluster           ClusterSnapshotProvider
	Consul            HealthChecker
}

func (r Readiness) Ping(ctx context.Context) error {
	checks := []struct {
		name    string
		checker HealthChecker
	}{
		{name: "store", checker: r.Store},
		{name: "event_outbox_worker", checker: r.Outbox},
		{name: "auction_close_worker", checker: r.AuctionClose},
		{name: "runtime_projection_worker", checker: r.RuntimeProjection},
		{name: "consul_registration", checker: r.Consul},
	}
	for _, check := range checks {
		if check.checker == nil && (check.name == "auction_close_worker" || check.name == "runtime_projection_worker") {
			continue
		}
		if check.checker == nil {
			return fmt.Errorf("%s health checker is missing", check.name)
		}
		if err := check.checker.Ping(ctx); err != nil {
			return fmt.Errorf("%s not ready: %w", check.name, err)
		}
	}
	return nil
}

func (r Readiness) MetricsSnapshot(ctx context.Context) map[string]any {
	if r.ProjectionMetrics == nil {
		return map[string]any{"status": "runtime projection metrics unavailable"}
	}
	return r.ProjectionMetrics.MetricsSnapshot(ctx)
}

func (r Readiness) WorkerSnapshot(ctx context.Context) map[string]any {
	workers := make(map[string]auction.WorkerStatus, len(r.WorkerStatuses))
	for _, provider := range r.WorkerStatuses {
		if provider == nil {
			continue
		}
		status := provider.WorkerStatus(ctx)
		workers[status.Name] = status
	}
	return map[string]any{
		"ok":      true,
		"workers": workers,
	}
}

func (r Readiness) ClusterSnapshot(ctx context.Context) map[string]any {
	if r.Cluster == nil {
		return map[string]any{"ok": true, "mode": "single"}
	}
	return r.Cluster.ClusterSnapshot(ctx)
}

func registerOperationHTTP(srv *httptransport.Server, health HealthChecker) {
	srv.Handle("/metrics", promhttp.Handler())
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	srv.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if health == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "missing health checker"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := health.Ping(ctx); err != nil {
			slog.Error("auction readiness check failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready", "error": "dependency not ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	srv.HandleFunc("/metrics/runtime-projection", func(w http.ResponseWriter, r *http.Request) {
		provider, ok := health.(RuntimeProjectionMetricsProvider)
		if !ok || provider == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"status": "runtime projection metrics unavailable"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		writeJSON(w, http.StatusOK, provider.MetricsSnapshot(ctx))
	})
	srv.HandleFunc("/workerz", func(w http.ResponseWriter, r *http.Request) {
		provider, ok := health.(WorkerSnapshotProvider)
		if !ok || provider == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"status": "worker status unavailable"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		writeJSON(w, http.StatusOK, provider.WorkerSnapshot(ctx))
	})
	srv.HandleFunc("/clusterz", func(w http.ResponseWriter, r *http.Request) {
		provider, ok := health.(ClusterSnapshotProvider)
		if !ok || provider == nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "mode": "single"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		writeJSON(w, http.StatusOK, provider.ClusterSnapshot(ctx))
	})
	srv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"service": "auction-backend", "transport": "kratos-http", "status": "ok"})
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
