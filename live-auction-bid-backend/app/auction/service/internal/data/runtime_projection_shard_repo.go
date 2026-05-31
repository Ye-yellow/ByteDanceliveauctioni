package data

import (
	"context"
	"errors"
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

func (s *Store) saveRuntimeProjectionShardOffset(ctx context.Context, shard int, streamID string, projectedAtUnixMs int64) error {
	if projectedAtUnixMs <= 0 {
		projectedAtUnixMs = time.Now().UnixMilli()
	}
	return s.db.WithContext(ctx).
		Model(&AuctionRuntimeProjectionShardOffsetModel{}).
		Where("shard_id = ?", shard).
		Updates(map[string]any{
			"last_stream_id":            streamID,
			"last_projected_at_unix_ms": projectedAtUnixMs,
			"updated_at_unix_ms":        time.Now().UnixMilli(),
		}).Error
}
