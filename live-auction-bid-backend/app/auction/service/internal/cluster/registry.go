package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/url"
	"sort"
	"strings"
	"sync"
)

type ShardStatus string

const (
	ShardStatusActive   ShardStatus = "active"
	ShardStatusDraining ShardStatus = "draining"
	ShardStatusOffline  ShardStatus = "offline"
)

type Shard struct {
	ID            int         `json:"id"`
	Name          string      `json:"name"`
	BackendURL    string      `json:"backendUrl"`
	WebSocketURL  string      `json:"webSocketUrl,omitempty"`
	MySQLDSN      string      `json:"mysqlDsn,omitempty"`
	RedisAddr     string      `json:"redisAddr,omitempty"`
	Weight        int         `json:"weight,omitempty"`
	Status        ShardStatus `json:"status,omitempty"`
	HotDedicated  bool        `json:"hotDedicated,omitempty"`
	MaxActiveRoom int         `json:"maxActiveRoom,omitempty"`
}

func (s Shard) normalized() Shard {
	s.Name = strings.TrimSpace(s.Name)
	s.BackendURL = strings.TrimRight(strings.TrimSpace(s.BackendURL), "/")
	s.WebSocketURL = strings.TrimRight(strings.TrimSpace(s.WebSocketURL), "/")
	s.MySQLDSN = strings.TrimSpace(s.MySQLDSN)
	s.RedisAddr = strings.TrimSpace(s.RedisAddr)
	if s.Name == "" {
		s.Name = fmt.Sprintf("shard-%d", s.ID)
	}
	if s.Weight <= 0 {
		s.Weight = 1
	}
	if s.Status == "" {
		s.Status = ShardStatusActive
	}
	return s
}

func (s Shard) AvailableForNewRooms() bool {
	return s.Status == ShardStatusActive && s.Weight > 0
}

func (s Shard) ServesExistingRooms() bool {
	return s.Status == ShardStatusActive || s.Status == ShardStatusDraining
}

type RoomAssignment struct {
	RoomID  string `json:"roomId"`
	ShardID int    `json:"shardId"`
}

type Snapshot struct {
	Shards      []Shard          `json:"shards"`
	Assignments []RoomAssignment `json:"assignments,omitempty"`
}

type StaticRegistry struct {
	mu          sync.RWMutex
	shards      map[int]Shard
	orderedIDs  []int
	assignments map[string]int
}

func NewStaticRegistry(shards []Shard, assignments map[string]int) (*StaticRegistry, error) {
	if len(shards) == 0 {
		return nil, errors.New("at least one shard is required")
	}
	r := &StaticRegistry{
		shards:      make(map[int]Shard, len(shards)),
		assignments: make(map[string]int, len(assignments)),
	}
	for _, shard := range shards {
		normalized := shard.normalized()
		if normalized.ID < 0 {
			return nil, fmt.Errorf("shard id must be non-negative: %d", normalized.ID)
		}
		if normalized.BackendURL == "" {
			return nil, fmt.Errorf("shard %d backend url is required", normalized.ID)
		}
		if _, err := url.ParseRequestURI(normalized.BackendURL); err != nil {
			return nil, fmt.Errorf("shard %d backend url is invalid: %w", normalized.ID, err)
		}
		if _, exists := r.shards[normalized.ID]; exists {
			return nil, fmt.Errorf("duplicate shard id: %d", normalized.ID)
		}
		r.shards[normalized.ID] = normalized
		r.orderedIDs = append(r.orderedIDs, normalized.ID)
	}
	sort.Ints(r.orderedIDs)
	for roomID, shardID := range assignments {
		roomID = strings.TrimSpace(roomID)
		if roomID == "" {
			continue
		}
		shard, ok := r.shards[shardID]
		if !ok {
			return nil, fmt.Errorf("assignment room %s references unknown shard %d", roomID, shardID)
		}
		if !shard.ServesExistingRooms() {
			return nil, fmt.Errorf("assignment room %s references unavailable shard %d", roomID, shardID)
		}
		r.assignments[roomID] = shardID
	}
	return r, nil
}

func validateShard(shard Shard) (Shard, error) {
	normalized := shard.normalized()
	if normalized.ID < 0 {
		return Shard{}, fmt.Errorf("shard id must be non-negative: %d", normalized.ID)
	}
	if normalized.BackendURL == "" {
		return Shard{}, fmt.Errorf("shard %d backend url is required", normalized.ID)
	}
	if _, err := url.ParseRequestURI(normalized.BackendURL); err != nil {
		return Shard{}, fmt.Errorf("shard %d backend url is invalid: %w", normalized.ID, err)
	}
	return normalized, nil
}

func ParseStaticRegistryJSON(raw string) (*StaticRegistry, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("cluster registry json is empty")
	}
	var payload struct {
		Shards      []Shard          `json:"shards"`
		Assignments []RoomAssignment `json:"assignments"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	assignments := make(map[string]int, len(payload.Assignments))
	for _, item := range payload.Assignments {
		assignments[strings.TrimSpace(item.RoomID)] = item.ShardID
	}
	return NewStaticRegistry(payload.Shards, assignments)
}

func (r *StaticRegistry) UpsertShard(shard Shard) error {
	normalized, err := validateShard(shard)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.shards[normalized.ID]; !exists {
		r.orderedIDs = append(r.orderedIDs, normalized.ID)
		sort.Ints(r.orderedIDs)
	}
	r.shards[normalized.ID] = normalized
	return nil
}

func (r *StaticRegistry) RemoveShard(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.shards[id]; !ok {
		return fmt.Errorf("unknown shard: %d", id)
	}
	for roomID, shardID := range r.assignments {
		if shardID == id {
			return fmt.Errorf("shard %d still owns room %s", id, roomID)
		}
	}
	delete(r.shards, id)
	for i, currentID := range r.orderedIDs {
		if currentID == id {
			r.orderedIDs = append(r.orderedIDs[:i], r.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

func (r *StaticRegistry) AssignRoomToShard(roomID string, shardID int) error {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return errors.New("room id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	shard, ok := r.shards[shardID]
	if !ok {
		return fmt.Errorf("unknown shard: %d", shardID)
	}
	if !shard.ServesExistingRooms() {
		return fmt.Errorf("shard %d is unavailable", shardID)
	}
	r.assignments[roomID] = shardID
	return nil
}

func (r *StaticRegistry) ClearRoomAssignment(roomID string) error {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return errors.New("room id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.assignments, roomID)
	return nil
}

func (r *StaticRegistry) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := Snapshot{
		Shards:      make([]Shard, 0, len(r.orderedIDs)),
		Assignments: make([]RoomAssignment, 0, len(r.assignments)),
	}
	for _, id := range r.orderedIDs {
		out.Shards = append(out.Shards, r.shards[id])
	}
	for roomID, shardID := range r.assignments {
		out.Assignments = append(out.Assignments, RoomAssignment{RoomID: roomID, ShardID: shardID})
	}
	sort.Slice(out.Assignments, func(i, j int) bool {
		return out.Assignments[i].RoomID < out.Assignments[j].RoomID
	})
	return out
}

func (r *StaticRegistry) LookupShard(id int) (Shard, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	shard, ok := r.shards[id]
	return shard, ok
}

func (r *StaticRegistry) RouteExistingRoom(roomID string) (Shard, bool) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return Shard{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	shardID, ok := r.assignments[roomID]
	if !ok {
		return Shard{}, false
	}
	shard, ok := r.shards[shardID]
	if !ok || !shard.ServesExistingRooms() {
		return Shard{}, false
	}
	return shard, true
}

func (r *StaticRegistry) AssignRoom(roomID string) (Shard, error) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return Shard{}, errors.New("room id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if shardID, ok := r.assignments[roomID]; ok {
		shard, ok := r.shards[shardID]
		if !ok || !shard.ServesExistingRooms() {
			return Shard{}, fmt.Errorf("assigned shard %d is unavailable", shardID)
		}
		return shard, nil
	}
	candidates := r.activeCandidatesLocked()
	if len(candidates) == 0 {
		return Shard{}, errors.New("no active shard is available")
	}
	shard := candidates[int(hashString(roomID)%uint32(len(candidates)))]
	r.assignments[roomID] = shard.ID
	return shard, nil
}

func (r *StaticRegistry) SetShardStatus(id int, status ShardStatus) error {
	if status != ShardStatusActive && status != ShardStatusDraining && status != ShardStatusOffline {
		return fmt.Errorf("unsupported shard status: %s", status)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	shard, ok := r.shards[id]
	if !ok {
		return fmt.Errorf("unknown shard: %d", id)
	}
	shard.Status = status
	r.shards[id] = shard
	return nil
}

func (r *StaticRegistry) RoomsOnShard(shardID int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rooms := make([]string, 0)
	for roomID, assignedShardID := range r.assignments {
		if assignedShardID == shardID {
			rooms = append(rooms, roomID)
		}
	}
	sort.Strings(rooms)
	return rooms
}

func (r *StaticRegistry) PickFailoverShard(sourceShardID int, includeHotDedicated bool) (Shard, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, id := range r.orderedIDs {
		if id == sourceShardID {
			continue
		}
		shard := r.shards[id]
		if shard.Status != ShardStatusActive {
			continue
		}
		if shard.HotDedicated && !includeHotDedicated {
			continue
		}
		return shard, true
	}
	return Shard{}, false
}

func (r *StaticRegistry) activeCandidatesLocked() []Shard {
	candidates := make([]Shard, 0, len(r.shards))
	activeRoomCounts := make(map[int]int, len(r.shards))
	for _, shardID := range r.assignments {
		activeRoomCounts[shardID]++
	}
	for _, id := range r.orderedIDs {
		shard := r.shards[id]
		if !shard.AvailableForNewRooms() {
			continue
		}
		if shard.HotDedicated {
			continue
		}
		if shard.MaxActiveRoom > 0 && activeRoomCounts[shard.ID] >= shard.MaxActiveRoom {
			continue
		}
		for i := 0; i < shard.Weight; i++ {
			candidates = append(candidates, shard)
		}
	}
	return candidates
}

func hashString(value string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return h.Sum32()
}
