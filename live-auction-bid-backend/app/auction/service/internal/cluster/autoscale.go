package cluster

import "fmt"

type ScaleAction string

const (
	ScaleActionNone            ScaleAction = "none"
	ScaleActionActivateStandby ScaleAction = "activate_standby"
	ScaleActionAddNode         ScaleAction = "add_node"
	ScaleActionDrainShard      ScaleAction = "drain_shard"
)

type ScalePolicy struct {
	MaxProjectionPending int64 `json:"maxProjectionPending"`
	MaxProjectionLagMs   int64 `json:"maxProjectionLagMs"`
	MaxRoomsPerShard     int   `json:"maxRoomsPerShard"`
	MinActiveShards      int   `json:"minActiveShards"`
	ScaleInPendingBelow  int64 `json:"scaleInPendingBelow"`
	ScaleInLagBelowMs    int64 `json:"scaleInLagBelowMs"`
}

type ScaleSignals struct {
	ProjectionPending int64 `json:"projectionPending"`
	ProjectionLagMs   int64 `json:"projectionLagMs"`
	ActiveRooms       int   `json:"activeRooms"`
}

type ScaleRecommendation struct {
	Action  ScaleAction `json:"action"`
	ShardID int         `json:"shardId,omitempty"`
	Reason  string      `json:"reason"`
	Safe    bool        `json:"safe"`
}

func DefaultScalePolicy() ScalePolicy {
	return ScalePolicy{
		MaxProjectionPending: 2000,
		MaxProjectionLagMs:   5000,
		MaxRoomsPerShard:     100,
		MinActiveShards:      1,
		ScaleInPendingBelow:  100,
		ScaleInLagBelowMs:    500,
	}
}

func EvaluateScale(policy ScalePolicy, signals ScaleSignals, snapshot Snapshot) ScaleRecommendation {
	policy = normalizeScalePolicy(policy)
	activeShards := shardsByStatus(snapshot, ShardStatusActive)
	if shouldScaleOut(policy, signals, len(activeShards)) {
		if shard, ok := firstShardByStatus(snapshot, ShardStatusDraining); ok {
			return ScaleRecommendation{Action: ScaleActionActivateStandby, ShardID: shard.ID, Reason: "load is above policy and a draining shard can return to active", Safe: true}
		}
		if shard, ok := firstShardByStatus(snapshot, ShardStatusOffline); ok {
			return ScaleRecommendation{Action: ScaleActionActivateStandby, ShardID: shard.ID, Reason: "load is above policy and a registered offline shard can be activated after readiness check", Safe: false}
		}
		return ScaleRecommendation{Action: ScaleActionAddNode, Reason: "load is above policy and no registered standby shard is available", Safe: false}
	}
	if shouldScaleIn(policy, signals, len(activeShards)) {
		if shard, ok := leastLoadedActiveShard(snapshot); ok {
			return ScaleRecommendation{Action: ScaleActionDrainShard, ShardID: shard.ID, Reason: "load is below policy and active shard count is above minimum", Safe: true}
		}
	}
	return ScaleRecommendation{Action: ScaleActionNone, Reason: "cluster is within scale policy", Safe: true}
}

func normalizeScalePolicy(policy ScalePolicy) ScalePolicy {
	defaults := DefaultScalePolicy()
	if policy.MaxProjectionPending <= 0 {
		policy.MaxProjectionPending = defaults.MaxProjectionPending
	}
	if policy.MaxProjectionLagMs <= 0 {
		policy.MaxProjectionLagMs = defaults.MaxProjectionLagMs
	}
	if policy.MaxRoomsPerShard <= 0 {
		policy.MaxRoomsPerShard = defaults.MaxRoomsPerShard
	}
	if policy.MinActiveShards <= 0 {
		policy.MinActiveShards = defaults.MinActiveShards
	}
	if policy.ScaleInPendingBelow < 0 {
		policy.ScaleInPendingBelow = defaults.ScaleInPendingBelow
	}
	if policy.ScaleInLagBelowMs < 0 {
		policy.ScaleInLagBelowMs = defaults.ScaleInLagBelowMs
	}
	return policy
}

func shouldScaleOut(policy ScalePolicy, signals ScaleSignals, activeShards int) bool {
	if signals.ProjectionPending > policy.MaxProjectionPending || signals.ProjectionLagMs > policy.MaxProjectionLagMs {
		return true
	}
	if activeShards <= 0 {
		return true
	}
	return signals.ActiveRooms > activeShards*policy.MaxRoomsPerShard
}

func shouldScaleIn(policy ScalePolicy, signals ScaleSignals, activeShards int) bool {
	if activeShards <= policy.MinActiveShards {
		return false
	}
	if signals.ProjectionPending > policy.ScaleInPendingBelow || signals.ProjectionLagMs > policy.ScaleInLagBelowMs {
		return false
	}
	roomFloor := (activeShards - 1) * policy.MaxRoomsPerShard
	return signals.ActiveRooms <= roomFloor
}

func shardsByStatus(snapshot Snapshot, status ShardStatus) []Shard {
	out := make([]Shard, 0)
	for _, shard := range snapshot.Shards {
		if shard.Status == status {
			out = append(out, shard)
		}
	}
	return out
}

func firstShardByStatus(snapshot Snapshot, status ShardStatus) (Shard, bool) {
	for _, shard := range snapshot.Shards {
		if shard.Status == status {
			return shard, true
		}
	}
	return Shard{}, false
}

func leastLoadedActiveShard(snapshot Snapshot) (Shard, bool) {
	counts := make(map[int]int, len(snapshot.Shards))
	for _, assignment := range snapshot.Assignments {
		counts[assignment.ShardID]++
	}
	var selected Shard
	found := false
	for _, shard := range snapshot.Shards {
		if shard.Status != ShardStatusActive || shard.ID == 0 {
			continue
		}
		if !found || counts[shard.ID] < counts[selected.ID] {
			selected = shard
			found = true
		}
	}
	return selected, found
}

func ApplyScaleRecommendation(registry *StaticRegistry, rec ScaleRecommendation) error {
	if registry == nil {
		return fmt.Errorf("registry is required")
	}
	switch rec.Action {
	case ScaleActionActivateStandby:
		return registry.SetShardStatus(rec.ShardID, ShardStatusActive)
	case ScaleActionDrainShard:
		return registry.SetShardStatus(rec.ShardID, ShardStatusDraining)
	case ScaleActionNone, ScaleActionAddNode:
		return nil
	default:
		return fmt.Errorf("unsupported scale action: %s", rec.Action)
	}
}
