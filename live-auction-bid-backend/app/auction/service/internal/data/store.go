package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	MySQLDSN      string
	RedisAddr     string
	RedisPassword string
}

type Store struct {
	db    *sql.DB
	redis *redis.Client
}

func NewStore(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.MySQLDSN == "" {
		return nil, errors.New("mysql dsn is required")
	}
	if cfg.RedisAddr == "" {
		return nil, errors.New("redis addr is required")
	}

	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = db.Close()
		_ = rdb.Close()
		return nil, err
	}

	store := &Store{db: db, redis: rdb}
	if err := store.migrate(ctx); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s.redis != nil {
		_ = s.redis.Close()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	for _, stmt := range []string{
		createAuctionLotsTable,
		createAuctionTrustCardsTable,
		createAuctionBidsTable,
		createAuctionEventsTable,
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
