package realtime

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

const (
	defaultNATSURL     = nats.DefaultURL
	defaultNATSSubject = "auction.realtime.events"
)

type NATSBusConfig struct {
	URL     string
	Subject string
	Queue   string
	Name    string
	Origin  string
}

type NATSBus struct {
	url     string
	subject string
	queue   string
	name    string
	origin  string

	mu   sync.Mutex
	conn *nats.Conn
	sub  *nats.Subscription
}

func NewNATSBus(cfg NATSBusConfig) (*NATSBus, error) {
	cfg.URL = strings.TrimSpace(cfg.URL)
	cfg.Subject = strings.TrimSpace(cfg.Subject)
	cfg.Queue = strings.TrimSpace(cfg.Queue)
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.Origin = strings.TrimSpace(cfg.Origin)
	if cfg.URL == "" {
		cfg.URL = defaultNATSURL
	}
	if cfg.Subject == "" {
		cfg.Subject = defaultNATSSubject
	}
	if cfg.Origin == "" {
		cfg.Origin = cfg.Name
	}
	if cfg.Origin == "" {
		cfg.Origin = "unknown"
	}
	if cfg.Name == "" {
		cfg.Name = "live-auction-" + cfg.Origin
	}
	return &NATSBus{
		url:     cfg.URL,
		subject: cfg.Subject,
		queue:   cfg.Queue,
		name:    cfg.Name,
		origin:  cfg.Origin,
	}, nil
}

func (b *NATSBus) Start(ctx context.Context, sink EventPublisher) error {
	if b == nil {
		return errors.New("nats realtime bus is not initialized")
	}
	if sink == nil {
		return errors.New("nats realtime bus sink is required")
	}
	conn, err := nats.Connect(b.url, nats.Name(b.name))
	if err != nil {
		return err
	}
	handler := func(msg *nats.Msg) {
		_, _ = b.dispatchPayload(ctx, sink, string(msg.Data))
	}
	var sub *nats.Subscription
	if b.queue != "" {
		sub, err = conn.QueueSubscribe(b.subject, b.queue, handler)
	} else {
		sub, err = conn.Subscribe(b.subject, handler)
	}
	if err != nil {
		conn.Close()
		return err
	}
	if err := conn.Flush(); err != nil {
		_ = sub.Unsubscribe()
		conn.Close()
		return err
	}
	b.mu.Lock()
	if b.sub != nil {
		_ = b.sub.Unsubscribe()
	}
	if b.conn != nil {
		b.conn.Close()
	}
	b.conn = conn
	b.sub = sub
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		_ = b.Close()
	}()
	return nil
}

func (b *NATSBus) Publish(_ context.Context, event v1.AuctionEvent) error {
	if b == nil {
		return errors.New("nats realtime bus is not initialized")
	}
	payload, err := encodeRedisBusEnvelope(b.origin, event)
	if err != nil {
		return err
	}
	b.mu.Lock()
	conn := b.conn
	b.mu.Unlock()
	if conn == nil || !conn.IsConnected() {
		return errors.New("nats realtime bus is not connected")
	}
	return conn.Publish(b.subject, []byte(payload))
}

func (b *NATSBus) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	sub := b.sub
	conn := b.conn
	b.sub = nil
	b.conn = nil
	b.mu.Unlock()
	if sub != nil {
		_ = sub.Unsubscribe()
	}
	if conn != nil {
		conn.Close()
	}
	return nil
}

func (b *NATSBus) dispatchPayload(ctx context.Context, sink EventPublisher, payload string) (bool, error) {
	return dispatchRedisBusPayload(ctx, b.origin, sink, payload)
}
