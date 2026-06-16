package data

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/observability"
)

type Config struct {
	MySQLDSN                                  string
	RedisAddr                                 string
	RedisPassword                             string
	RuntimeProjectionShards                   int
	RuntimeProjectionBackpressurePendingLimit int64
	RuntimeProjectionBackpressureLag          time.Duration
	DBMaxOpenConns                            int
	DBMaxIdleConns                            int
	DBConnMaxLifetime                         time.Duration
	DBConnMaxIdleTime                         time.Duration
	RedisPoolSize                             int
	RedisMinIdleConns                         int
}

// Store is the single production data path for the auction service.
//
// Persistence rules:
// - GORM + MySQL owns authoritative lot and bid state.
// - MySQL keeps accepted bids and durable idempotency keys; Redis caches idempotency lookups.
// - There is intentionally no in-memory or database/sql fallback.
type Store struct {
	db                                        *gorm.DB
	redis                                     *redis.Client
	runtimeProjectionShards                   int
	runtimeProjectionBackpressurePendingLimit int64
	runtimeProjectionBackpressureLagMs        int64
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
	store := &Store{
		db:                      db,
		redis:                   rdb,
		runtimeProjectionShards: cfg.RuntimeProjectionShards,
		runtimeProjectionBackpressurePendingLimit: cfg.RuntimeProjectionBackpressurePendingLimit,
		runtimeProjectionBackpressureLagMs:        cfg.RuntimeProjectionBackpressureLag.Milliseconds(),
	}
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
		&UserOrderModel{},
		&UserOrderItemModel{},
		&UserOrderPaymentModel{},
		&AuctionDepositHoldModel{},
		&ShopProductModel{},
		&ShopSKUModel{},
		&UserDeliveryAddressModel{},
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
	if err := s.EnsureRBACDefaults(ctx); err != nil {
		return err
	}
	if err := s.ensureVisualDefaults(ctx); err != nil {
		return err
	}
	return s.EnsureShopSeeds(ctx)
}

func (s *Store) ensureVisualDefaults(ctx context.Context) error {
	var users []AuctionUserModel
	if err := s.db.WithContext(ctx).
		Select("id").
		Where("avatar_url = ?", "").
		Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if err := s.db.WithContext(ctx).Model(&AuctionUserModel{}).
			Where("id = ? AND avatar_url = ?", user.ID, "").
			Update("avatar_url", userbiz.AvatarURLForUserID(user.ID)).Error; err != nil {
			return err
		}
	}

	var rooms []AuctionRoomModel
	if err := s.db.WithContext(ctx).
		Select("id", "created_at_unix_ms").
		Where("live_source_url = ? OR live_started_at_unix_ms = 0", "").
		Find(&rooms).Error; err != nil {
		return err
	}
	nowMs := time.Now().UnixMilli()
	for _, room := range rooms {
		startedAt := room.CreatedAtUnixMs
		if startedAt <= 0 {
			startedAt = nowMs
		}
		if err := s.db.WithContext(ctx).Model(&AuctionRoomModel{}).
			Where("id = ?", room.ID).
			Updates(map[string]any{
				"live_source_url":         gorm.Expr("CASE WHEN live_source_url = '' THEN ? ELSE live_source_url END", auction.LiveSourceURLForRoomID(room.ID)),
				"live_started_at_unix_ms": gorm.Expr("CASE WHEN live_started_at_unix_ms = 0 THEN ? ELSE live_started_at_unix_ms END", startedAt),
			}).Error; err != nil {
			return err
		}
	}
	return nil
}
