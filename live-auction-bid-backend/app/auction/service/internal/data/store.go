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
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS auction_lots (
  id VARCHAR(64) PRIMARY KEY,
  room_id VARCHAR(64) NOT NULL,
  status INT NOT NULL,
  payload JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_room_status (room_id, status),
  INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS auction_bids (
  id VARCHAR(64) PRIMARY KEY,
  lot_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  amount BIGINT NOT NULL,
  currency VARCHAR(16) NOT NULL,
  created_at_unix_ms BIGINT NOT NULL,
  payload JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_lot_created (lot_id, created_at_unix_ms),
  INDEX idx_lot_user (lot_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`)
	return err
}
