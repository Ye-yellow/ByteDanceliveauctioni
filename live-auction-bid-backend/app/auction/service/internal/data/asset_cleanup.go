package data

import (
	"context"
	"log/slog"
	"time"

	"live-auction-bid/backend/app/auction/service/internal/storage"
)

type AssetCleanupWorker struct {
	store    *Store
	storage  storage.StorageProvider
	interval time.Duration
	limit    int
}

func NewAssetCleanupWorker(store *Store, provider storage.StorageProvider, interval time.Duration, limit int) *AssetCleanupWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	if limit <= 0 {
		limit = 100
	}
	return &AssetCleanupWorker{store: store, storage: provider, interval: interval, limit: limit}
}

func (w *AssetCleanupWorker) Start(ctx context.Context) {
	if w == nil || w.store == nil || w.storage == nil {
		return
	}
	go func() {
		w.runOnce(ctx)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.runOnce(ctx)
			}
		}
	}()
}

func (w *AssetCleanupWorker) runOnce(ctx context.Context) {
	assets, err := w.store.ListExpiredTemporaryAssets(ctx, w.limit)
	if err != nil {
		slog.Error("asset cleanup list failed", "error", err)
		return
	}
	for _, asset := range assets {
		if err := w.storage.DeleteObject(ctx, asset.ObjectKey); err != nil {
			slog.Error("asset cleanup storage delete failed", "asset_id", asset.ID, "object_key", asset.ObjectKey, "error", err)
			continue
		}
		if err := w.store.MarkAssetDeletedByID(ctx, asset.ID); err != nil {
			slog.Error("asset cleanup mark deleted failed", "asset_id", asset.ID, "error", err)
			continue
		}
		slog.Info("asset cleanup deleted expired temporary asset", "asset_id", asset.ID, "object_key", asset.ObjectKey)
	}
}
