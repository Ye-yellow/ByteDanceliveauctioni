package data

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"live-auction-bid/backend/app/auction/service/internal/observability"
)

type Config struct {
	MySQLDSN                string
	RedisAddr               string
	RedisPassword           string
	RuntimeProjectionShards int
	DBMaxOpenConns          int
	DBMaxIdleConns          int
	DBConnMaxLifetime       time.Duration
	DBConnMaxIdleTime       time.Duration
	RedisPoolSize           int
	RedisMinIdleConns       int
}

// Store is the single production data path for the auction service.
//
// Persistence rules:
// - GORM + MySQL owns authoritative lot and bid state.
// - MySQL keeps accepted bids and durable idempotency keys; Redis caches idempotency lookups.
// - There is intentionally no in-memory or database/sql fallback.
type Store struct {
	db                      *gorm.DB
	redis                   *redis.Client
	runtimeProjectionShards int
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
	if cfg.DBMaxOpenConns <= 0 {
		cfg.DBMaxOpenConns = 20
	}
	if cfg.DBMaxIdleConns <= 0 {
		cfg.DBMaxIdleConns = cfg.DBMaxOpenConns / 2
	}
	if cfg.DBConnMaxLifetime <= 0 {
		cfg.DBConnMaxLifetime = 30 * time.Minute
	}
	if cfg.DBConnMaxIdleTime <= 0 {
		cfg.DBConnMaxIdleTime = 2 * time.Minute
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	redisOptions := &redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword}
	if cfg.RedisPoolSize > 0 {
		redisOptions.PoolSize = cfg.RedisPoolSize
	}
	if cfg.RedisMinIdleConns > 0 {
		redisOptions.MinIdleConns = cfg.RedisMinIdleConns
	}
	rdb := redis.NewClient(redisOptions)
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = sqlDB.Close()
		_ = rdb.Close()
		return nil, err
	}

	if cfg.RuntimeProjectionShards <= 0 {
		cfg.RuntimeProjectionShards = defaultRuntimeProjectionShards
	}
	observability.BindDBStatsProvider(sqlDB.Stats)
	observability.BindRedisPoolStatsProvider(func() observability.RedisPoolStats {
		stats := rdb.PoolStats()
		return observability.RedisPoolStats{
			Hits:       stats.Hits,
			Misses:     stats.Misses,
			Timeouts:   stats.Timeouts,
			TotalConns: stats.TotalConns,
			IdleConns:  stats.IdleConns,
			StaleConns: stats.StaleConns,
		}
	})
	store := &Store{db: db, redis: rdb, runtimeProjectionShards: cfg.RuntimeProjectionShards}
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
	if err := s.db.WithContext(ctx).AutoMigrate(
		&AuctionRoomModel{},
		&AuctionRoomStateModel{},
		&AuctionLotModel{},
		&AuctionBidModel{},
		&AuctionLotStatsModel{},
		&AuctionLotParticipantModel{},
		&AuctionRuntimeProjectionOffsetModel{},
		&AuctionRuntimeProjectionShardOffsetModel{},
		&AuctionOrderModel{},
		&AuctionPaymentModel{},
		&AuctionEventModel{},
		&AssetFileModel{},
		&AuctionUserModel{},
		&AuctionRoleModel{},
		&AuctionPermissionModel{},
		&AuctionUserRoleModel{},
		&AuctionRolePermissionModel{},
		&AuctionUserPermissionModel{},
		&AuctionUserSessionModel{},
	); err != nil {
		return err
	}
	return s.EnsureRBACDefaults(ctx)
}
