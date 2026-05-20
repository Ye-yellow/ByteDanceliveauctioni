package idgen

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	epochMs int64 = 1704067200000 // 2024-01-01 00:00:00 UTC

	workerIDBits uint = 10
	sequenceBits uint = 12

	maxWorkerID  int64 = (1 << workerIDBits) - 1
	sequenceMask int64 = (1 << sequenceBits) - 1

	workerIDShift  = sequenceBits
	timestampShift = sequenceBits + workerIDBits
)

type Generator struct {
	mu              sync.Mutex
	workerID        int64
	lastTimestampMs int64
	sequence        int64
	now             func() int64
}

type Option func(*Generator)

func WithNowFunc(now func() int64) Option {
	return func(g *Generator) {
		if now != nil {
			g.now = now
		}
	}
}

func NewGenerator(workerID int64, opts ...Option) (*Generator, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, fmt.Errorf("snowflake worker id must be between 0 and %d", maxWorkerID)
	}

	g := &Generator{
		workerID:        workerID,
		lastTimestampMs: -1,
		now: func() int64 {
			return time.Now().UnixMilli()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(g)
		}
	}
	return g, nil
}

func (g *Generator) NextID() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	nowMs := g.now()
	if nowMs < epochMs {
		nowMs = epochMs
	}
	if nowMs < g.lastTimestampMs {
		nowMs = g.lastTimestampMs
	}

	if nowMs == g.lastTimestampMs {
		g.sequence = (g.sequence + 1) & sequenceMask
		if g.sequence == 0 {
			nowMs = g.nextTimestampAfter(g.lastTimestampMs)
		}
	} else {
		g.sequence = 0
	}

	g.lastTimestampMs = nowMs
	return ((nowMs - epochMs) << timestampShift) | (g.workerID << workerIDShift) | g.sequence
}

func (g *Generator) NextString() string {
	return strconv.FormatInt(g.NextID(), 10)
}

func (g *Generator) nextTimestampAfter(lastTimestampMs int64) int64 {
	nowMs := g.now()
	if nowMs <= lastTimestampMs {
		return lastTimestampMs + 1
	}
	return nowMs
}

var defaultGenerator = mustNewDefaultGenerator()

func mustNewDefaultGenerator() *Generator {
	g, err := NewGenerator(workerIDFromEnv())
	if err != nil {
		panic(err)
	}
	return g
}

func workerIDFromEnv() int64 {
	raw := os.Getenv("SNOWFLAKE_WORKER_ID")
	if raw == "" {
		return 0
	}
	workerID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || workerID < 0 || workerID > maxWorkerID {
		return 0
	}
	return workerID
}

func New(prefix string) string {
	return defaultGenerator.NextString()
}

func NewUserID() string {
	return defaultGenerator.NextString()
}
