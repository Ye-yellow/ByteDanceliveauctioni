package server

import (
	"context"
	"testing"

	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
)

func TestReadinessWorkerSnapshotDoesNotAffectPing(t *testing.T) {
	readiness := Readiness{
		Store:             okHealthChecker{},
		Outbox:            okHealthChecker{},
		Consul:            okHealthChecker{},
		WorkerStatuses:    []auction.WorkerStatusProvider{staticWorkerStatus{status: auction.WorkerStatus{Name: "event_outbox_worker", Mode: auction.WorkerModeStandby, LeaseOwner: "worker-a", Started: true}}},
		ProjectionMetrics: staticProjectionMetrics{},
	}
	if err := readiness.Ping(context.Background()); err != nil {
		t.Fatalf("standby worker status should not make readiness fail: %v", err)
	}
	snapshot := readiness.WorkerSnapshot(context.Background())
	workers, ok := snapshot["workers"].(map[string]auction.WorkerStatus)
	if !ok {
		t.Fatalf("workers snapshot type mismatch: %#v", snapshot["workers"])
	}
	if workers["event_outbox_worker"].Mode != auction.WorkerModeStandby {
		t.Fatalf("worker snapshot should expose standby mode: %+v", workers["event_outbox_worker"])
	}
}

type okHealthChecker struct{}

func (okHealthChecker) Ping(context.Context) error {
	return nil
}

type staticWorkerStatus struct {
	status auction.WorkerStatus
}

func (s staticWorkerStatus) WorkerStatus(context.Context) auction.WorkerStatus {
	return s.status
}

type staticProjectionMetrics struct{}

func (staticProjectionMetrics) MetricsSnapshot(context.Context) map[string]any {
	return map[string]any{}
}
