package auction

import (
	"context"
	"time"
)

type LeaseProvider interface {
	TryAcquire(ctx context.Context, key, owner string, ttl time.Duration) (Lease, bool, error)
	Owner(ctx context.Context, key string) (string, error)
}

type Lease interface {
	Key() string
	Owner() string
	Renew(ctx context.Context) (bool, error)
	Release(ctx context.Context) error
}

type WorkerMode string

const (
	WorkerModeLeader         WorkerMode = "leader"
	WorkerModeStandby        WorkerMode = "standby"
	WorkerModePartialOwner   WorkerMode = "partial_owner"
	WorkerModeAcquiring      WorkerMode = "acquiring"
	WorkerModeDisabled       WorkerMode = "disabled"
	WorkerModeLeaderDegraded WorkerMode = "leader_degraded"
	WorkerModeLostLease      WorkerMode = "lost_lease"
	WorkerModeMisconfigured  WorkerMode = "misconfigured"
)

type WorkerStatus struct {
	Name          string             `json:"name"`
	Mode          WorkerMode         `json:"mode"`
	InstanceID    string             `json:"instance_id"`
	LeaseKey      string             `json:"lease_key,omitempty"`
	LeaseOwner    string             `json:"lease_owner,omitempty"`
	Started       bool               `json:"started"`
	OwnedShards   []int              `json:"owned_shards,omitempty"`
	StandbyShards []int              `json:"standby_shards,omitempty"`
	FailedShards  []int              `json:"failed_shards,omitempty"`
	LastAttemptAt string             `json:"last_attempt_at,omitempty"`
	LastSuccessAt string             `json:"last_success_at,omitempty"`
	LastError     string             `json:"last_error,omitempty"`
	Extra         map[string]any     `json:"extra,omitempty"`
	Shards        []WorkerShardState `json:"shards,omitempty"`
}

type WorkerShardState struct {
	ShardID       int        `json:"shard_id"`
	Mode          WorkerMode `json:"mode"`
	LeaseKey      string     `json:"lease_key"`
	LeaseOwner    string     `json:"lease_owner,omitempty"`
	LastAttemptAt string     `json:"last_attempt_at,omitempty"`
	LastSuccessAt string     `json:"last_success_at,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
}

type WorkerStatusProvider interface {
	WorkerStatus(ctx context.Context) WorkerStatus
}
