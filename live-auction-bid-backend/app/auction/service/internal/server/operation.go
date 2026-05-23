package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
)

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type Readiness struct {
	Store        HealthChecker
	Outbox       HealthChecker
	AuctionClose HealthChecker
	Consul       HealthChecker
}

func (r Readiness) Ping(ctx context.Context) error {
	checks := []struct {
		name    string
		checker HealthChecker
	}{
		{name: "store", checker: r.Store},
		{name: "event_outbox_worker", checker: r.Outbox},
		{name: "auction_close_worker", checker: r.AuctionClose},
		{name: "consul_registration", checker: r.Consul},
	}
	for _, check := range checks {
		if check.checker == nil && check.name == "auction_close_worker" {
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

func registerOperationHTTP(srv *httptransport.Server, health HealthChecker) {
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
	srv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"service": "auction-backend", "transport": "kratos-http", "status": "ok"})
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
