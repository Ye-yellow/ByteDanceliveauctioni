package data

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) runtimeProjectionShardOffset(ctx context.Context, shard int) (AuctionRuntimeProjectionShardOffsetModel, error) {
	nowMs := time.Now().UnixMilli()
	var offset AuctionRuntimeProjectionShardOffsetModel
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("shard_id = ?", shard).
			First(&offset).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		offset = AuctionRuntimeProjectionShardOffsetModel{
			ShardID:               shard,
			LastStreamID:          "0-0",
			LastProjectedAtUnixMs: 0,
			UpdatedAtUnixMs:       nowMs,
		}
		return tx.Create(&offset).Error
	})
	return offset, err
}

func (s *Store) recordRuntimeProjectionShardProgress(ctx context.Context, shard int, processed int64, lastOccurredAtUnixMs int64) error {
	if processed <= 0 || s == nil || s.redis == nil {
		return nil
	}
	lagMs := int64(0)
	if lastOccurredAtUnixMs > 0 {
		lagMs = time.Now().UnixMilli() - lastOccurredAtUnixMs
		if lagMs < 0 {
			lagMs = 0
		}
	}
	pipe := s.redis.Pipeline()
	pipe.IncrBy(ctx, runtimeProjectionShardMetricKey("runtime_event_projected_total", shard), processed)
	pipe.Set(ctx, runtimeProjectionShardMetricKey("runtime_projection_lag_ms", shard), strconv.FormatInt(lagMs, 10), 0)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) saveRuntimeProjectionShardOffset(ctx context.Context, shard int, streamID string, projectedAtUnixMs int64) error {
	if projectedAtUnixMs <= 0 {
		projectedAtUnixMs = time.Now().UnixMilli()
	}
	nowMs := time.Now().UnixMilli()
	offset := AuctionRuntimeProjectionShardOffsetModel{
		ShardID:               shard,
		LastStreamID:          streamID,
		LastProjectedAtUnixMs: projectedAtUnixMs,
		UpdatedAtUnixMs:       nowMs,
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "shard_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"last_stream_id":            streamID,
				"last_projected_at_unix_ms": projectedAtUnixMs,
				"updated_at_unix_ms":        nowMs,
			}),
		}).
		Create(&offset).Error
}
