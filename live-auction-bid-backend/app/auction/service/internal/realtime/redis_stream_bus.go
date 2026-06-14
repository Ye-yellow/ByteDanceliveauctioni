package realtime

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

const (
	defaultRedisBusStream      = "auction:realtime:events:stream"
	defaultRedisBusGroupPrefix = "auction-realtime"
	redisBusPayloadField       = "payload"
)

type RedisStreamBusConfig struct {
	Addr         string
	Password     string
	DB           int
	Stream       string
	Group        string
	Consumer     string
	Origin       string
	Block        time.Duration
	Count        int64
	MaxLenApprox int64
}

type RedisStreamBus struct {
	client       *redis.Client
	stream       string
	group        string
	consumer     string
	origin       string
	block        time.Duration
	count        int64
	maxLenApprox int64

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewRedisStreamBus(cfg RedisStreamBusConfig) (*RedisStreamBus, error) {
	cfg.Addr = strings.TrimSpace(cfg.Addr)
	cfg.Stream = strings.TrimSpace(cfg.Stream)
	cfg.Group = strings.TrimSpace(cfg.Group)
	cfg.Consumer = strings.TrimSpace(cfg.Consumer)
	cfg.Origin = strings.TrimSpace(cfg.Origin)
	if cfg.Addr == "" {
		return nil, errors.New("redis stream realtime bus addr is required")
	}
	if cfg.Stream == "" {
		cfg.Stream = defaultRedisBusStream
	}
	if cfg.Consumer == "" {
		cfg.Consumer = cfg.Origin
	}
	if cfg.Consumer == "" {
		cfg.Consumer = "unknown"
	}
	if cfg.Origin == "" {
		cfg.Origin = cfg.Consumer
	}
	if cfg.Group == "" {
		cfg.Group = defaultRedisBusGroupPrefix + "-" + cfg.Origin
	}
	if cfg.Block <= 0 {
		cfg.Block = 2 * time.Second
	}
	if cfg.Count <= 0 {
		cfg.Count = 100
	}
	return &RedisStreamBus{
		client:       redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB}),
		stream:       cfg.Stream,
		group:        cfg.Group,
		consumer:     cfg.Consumer,
		origin:       cfg.Origin,
		block:        cfg.Block,
		count:        cfg.Count,
		maxLenApprox: cfg.MaxLenApprox,
	}, nil
}

func (b *RedisStreamBus) Start(ctx context.Context, sink EventPublisher) error {
	if b == nil || b.client == nil {
		return errors.New("redis stream realtime bus is not initialized")
	}
	if sink == nil {
		return errors.New("redis stream realtime bus sink is required")
	}
	if err := b.ensureGroup(ctx); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	b.mu.Lock()
	if b.cancel != nil {
		b.cancel()
	}
	b.cancel = cancel
	b.mu.Unlock()
	go b.receive(runCtx, sink)
	return nil
}

func (b *RedisStreamBus) Publish(ctx context.Context, event v1.AuctionEvent) error {
	if b == nil || b.client == nil {
		return errors.New("redis stream realtime bus is not initialized")
	}
	payload, err := encodeRedisBusEnvelope(b.origin, event)
	if err != nil {
		return err
	}
	args := &redis.XAddArgs{
		Stream: b.stream,
		Values: map[string]any{redisBusPayloadField: payload},
	}
	if b.maxLenApprox > 0 {
		args.MaxLen = b.maxLenApprox
		args.Approx = true
	}
	return b.client.XAdd(ctx, args).Err()
}

func (b *RedisStreamBus) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	cancel := b.cancel
	b.cancel = nil
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}

func (b *RedisStreamBus) ensureGroup(ctx context.Context) error {
	err := b.client.XGroupCreateMkStream(ctx, b.stream, b.group, "0").Err()
	if err == nil || strings.Contains(err.Error(), "BUSYGROUP") {
		return nil
	}
	return err
}

func (b *RedisStreamBus) receive(ctx context.Context, sink EventPublisher) {
	for {
		streams, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    b.group,
			Consumer: b.consumer,
			Streams:  []string{b.stream, ">"},
			Count:    b.count,
			Block:    b.block,
		}).Result()
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				return
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			slog.Warn("redis stream realtime bus read failed", "stream", b.stream, "group", b.group, "error", err)
			continue
		}
		for _, stream := range streams {
			for _, message := range stream.Messages {
				if err := b.handleMessage(ctx, sink, message); err != nil {
					slog.Warn("redis stream realtime bus dispatch failed", "stream", b.stream, "message_id", message.ID, "error", err)
					continue
				}
			}
		}
	}
}

func (b *RedisStreamBus) handleMessage(ctx context.Context, sink EventPublisher, message redis.XMessage) error {
	payload, ok := stringValue(message.Values[redisBusPayloadField])
	if !ok {
		_ = b.client.XAck(ctx, b.stream, b.group, message.ID).Err()
		return errors.New("redis stream realtime message payload is missing")
	}
	_, err := dispatchRedisBusPayload(ctx, b.origin, sink, payload)
	if err != nil {
		return err
	}
	return b.client.XAck(ctx, b.stream, b.group, message.ID).Err()
}

func stringValue(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), strings.TrimSpace(v) != ""
	case []byte:
		s := strings.TrimSpace(string(v))
		return s, s != ""
	default:
		return "", false
	}
}
