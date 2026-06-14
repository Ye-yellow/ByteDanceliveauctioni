package realtime

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

const defaultRedisBusChannel = "auction:realtime:events"

var redisBusMarshal = protojson.MarshalOptions{
	UseEnumNumbers: false,
	UseProtoNames:  false,
}

var redisBusUnmarshal = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}

type RedisBusConfig struct {
	Addr     string
	Password string
	DB       int
	Channel  string
	Origin   string
}

type RedisPubSubBus struct {
	client  *redis.Client
	channel string
	origin  string

	mu     sync.Mutex
	pubsub *redis.PubSub
}

type redisBusEnvelope struct {
	Origin string `json:"origin"`
	Event  []byte `json:"event"`
}

func NewRedisPubSubBus(cfg RedisBusConfig) (*RedisPubSubBus, error) {
	cfg.Addr = strings.TrimSpace(cfg.Addr)
	cfg.Channel = strings.TrimSpace(cfg.Channel)
	cfg.Origin = strings.TrimSpace(cfg.Origin)
	if cfg.Addr == "" {
		return nil, errors.New("redis realtime bus addr is required")
	}
	if cfg.Channel == "" {
		cfg.Channel = defaultRedisBusChannel
	}
	if cfg.Origin == "" {
		cfg.Origin = "unknown"
	}
	return &RedisPubSubBus{
		client:  redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB}),
		channel: cfg.Channel,
		origin:  cfg.Origin,
	}, nil
}

func (b *RedisPubSubBus) Start(ctx context.Context, sink EventPublisher) error {
	if b == nil || b.client == nil {
		return errors.New("redis realtime bus is not initialized")
	}
	if sink == nil {
		return errors.New("redis realtime bus sink is required")
	}
	pubsub := b.client.Subscribe(ctx, b.channel)
	if err := pubsub.Ping(ctx); err != nil {
		_ = pubsub.Close()
		return err
	}
	b.mu.Lock()
	if b.pubsub != nil {
		_ = b.pubsub.Close()
	}
	b.pubsub = pubsub
	b.mu.Unlock()
	go b.receive(ctx, pubsub, sink)
	return nil
}

func (b *RedisPubSubBus) Publish(ctx context.Context, event v1.AuctionEvent) error {
	if b == nil || b.client == nil {
		return errors.New("redis realtime bus is not initialized")
	}
	payload, err := encodeRedisBusEnvelope(b.origin, event)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, b.channel, payload).Err()
}

func (b *RedisPubSubBus) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	pubsub := b.pubsub
	b.pubsub = nil
	b.mu.Unlock()
	if pubsub != nil {
		_ = pubsub.Close()
	}
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}

func (b *RedisPubSubBus) receive(ctx context.Context, pubsub *redis.PubSub, sink EventPublisher) {
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			delivered, err := b.dispatchPayload(ctx, sink, msg.Payload)
			if err != nil {
				slog.Warn("redis realtime bus dispatch failed", "channel", b.channel, "error", err)
				continue
			}
			if delivered {
				slog.Debug("redis realtime bus delivered event", "channel", b.channel)
			}
		}
	}
}

func (b *RedisPubSubBus) dispatchPayload(ctx context.Context, sink EventPublisher, payload string) (bool, error) {
	return dispatchRedisBusPayload(ctx, b.origin, sink, payload)
}

func encodeRedisBusEnvelope(origin string, event v1.AuctionEvent) (string, error) {
	raw, err := redisBusMarshal.Marshal(&event)
	if err != nil {
		return "", err
	}
	envelope := redisBusEnvelope{Origin: strings.TrimSpace(origin), Event: raw}
	payload, err := jsonMarshal(envelope)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeRedisBusEnvelope(payload string) (string, v1.AuctionEvent, error) {
	var envelope redisBusEnvelope
	if err := jsonUnmarshal([]byte(payload), &envelope); err != nil {
		return "", v1.AuctionEvent{}, err
	}
	var event v1.AuctionEvent
	if err := redisBusUnmarshal.Unmarshal(envelope.Event, &event); err != nil {
		return "", v1.AuctionEvent{}, err
	}
	return strings.TrimSpace(envelope.Origin), event, nil
}

func dispatchRedisBusPayload(ctx context.Context, ownOrigin string, sink EventPublisher, payload string) (bool, error) {
	origin, event, err := decodeRedisBusEnvelope(payload)
	if err != nil {
		return false, err
	}
	if origin == strings.TrimSpace(ownOrigin) {
		return false, nil
	}
	if err := sink.Publish(ctx, event); err != nil {
		return false, err
	}
	return true, nil
}
