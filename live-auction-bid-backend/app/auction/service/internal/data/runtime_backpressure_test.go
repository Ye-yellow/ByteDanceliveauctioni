package data

import (
	"os"
	"strings"
	"testing"
)

func TestRuntimeBidBackpressureDoesNotBlockOnStaleLagWhenShardCaughtUp(t *testing.T) {
	source, err := os.ReadFile("runtime_repo.go")
	if err != nil {
		t.Fatalf("read runtime repo source: %v", err)
	}
	script := string(source)
	oldLagOnlyGuard := "projection_lag_limit_ms > 0 and shard_lag_ms > projection_lag_limit_ms"
	pendingAwareGuard := "projection_lag_limit_ms > 0 and shard_pending > 0 and shard_lag_ms > projection_lag_limit_ms"
	if strings.Contains(script, oldLagOnlyGuard) {
		t.Fatalf("runtime Lua must not reject bids using stale lag alone")
	}
	if !strings.Contains(script, pendingAwareGuard) {
		t.Fatalf("runtime Lua should only apply lag backpressure while shard has pending events")
	}
}
