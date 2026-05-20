package data

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	MySQLDSN      string
	RedisAddr     string
	RedisPassword string
}

// Store is the single production data path for the auction service.
//
// Persistence rules:
// - GORM + MySQL owns authoritative lot and bid state.
// - MySQL keeps accepted bids and durable idempotency keys; Redis caches idempotency lookups.
// - There is intentionally no in-memory or database/sql fallback.
type Store struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewStore(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.MySQLDSN == "" {
		return nil, errors.New("mysql dsn is required")
	}
	if cfg.RedisAddr == "" {
		return nil, errors.New("redis addr is required")
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = sqlDB.Close()
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

func (s *Store) Ping(ctx context.Context) error {
	if s.db == nil || s.redis == nil {
		return errors.New("store is not initialized")
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return err
	}
	return s.redis.Ping(ctx).Err()
}

func (s *Store) Close() error {
	if s.redis != nil {
		_ = s.redis.Close()
	}
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(
		&AuctionLotModel{},
		&AuctionBidModel{},
		&AuctionEventModel{},
		&AuctionUserModel{},
		&AuctionUserSessionModel{},
	)
}
