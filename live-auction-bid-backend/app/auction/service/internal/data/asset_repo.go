package data

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	AssetStatusTemporary = "temporary"
	AssetStatusAttached  = "attached"
	AssetStatusDeleted   = "deleted"
)

type AssetFile struct {
	ID               string
	MainAccountID    string
	OwnerUserID      string
	RoomID           string
	BizType          string
	Status           string
	AttachedLotID    string
	StorageProvider  string
	Bucket           string
	ObjectKey        string
	PublicURL        string
	OriginalName     string
	MimeType         string
	SizeBytes        int64
	SHA256           string
	AttachedAtUnixMs int64
	DeletedAtUnixMs  int64
	ExpiresAtUnixMs  int64
}

func (s *Store) SaveAssetFile(ctx context.Context, asset AssetFile) error {
	if asset.ID == "" {
		return errors.New("asset id is required")
	}
	if asset.OwnerUserID == "" {
		return errors.New("asset owner user id is required")
	}
	if asset.StorageProvider == "" || asset.Bucket == "" || asset.ObjectKey == "" || asset.PublicURL == "" {
		return errors.New("asset storage location is required")
	}
	if asset.MimeType == "" || asset.SizeBytes <= 0 || asset.SHA256 == "" {
		return errors.New("asset file metadata is required")
	}
	model := AssetFileModel{
		ID:               asset.ID,
		MainAccountID:    asset.MainAccountID,
		OwnerUserID:      asset.OwnerUserID,
		RoomID:           asset.RoomID,
		BizType:          asset.BizType,
		Status:           assetStatusOrDefault(asset.Status),
		AttachedLotID:    asset.AttachedLotID,
		StorageProvider:  asset.StorageProvider,
		Bucket:           asset.Bucket,
		ObjectKey:        asset.ObjectKey,
		PublicURL:        asset.PublicURL,
		OriginalName:     asset.OriginalName,
		MimeType:         asset.MimeType,
		SizeBytes:        asset.SizeBytes,
		SHA256:           asset.SHA256,
		AttachedAtUnixMs: asset.AttachedAtUnixMs,
		DeletedAtUnixMs:  asset.DeletedAtUnixMs,
		ExpiresAtUnixMs:  asset.ExpiresAtUnixMs,
	}
	return s.db.WithContext(ctx).Create(&model).Error
}

func assetStatusOrDefault(status string) string {
	if status == "" {
		return AssetStatusTemporary
	}
	return status
}

func (s *Store) AttachAssetFilesByURL(ctx context.Context, ownerUserID, lotID string, publicURLs []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return attachAssetFilesByURL(tx, ownerUserID, lotID, publicURLs)
	})
}

func attachAssetFilesByURL(tx *gorm.DB, ownerUserID, lotID string, publicURLs []string) error {
	if ownerUserID == "" || lotID == "" || len(publicURLs) == 0 {
		return nil
	}
	urls := compactStrings(publicURLs)
	if len(urls) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	return tx.Model(&AssetFileModel{}).
		Where("owner_user_id = ? AND public_url IN ? AND status = ? AND deleted_at_unix_ms = 0", ownerUserID, urls, AssetStatusTemporary).
		Updates(map[string]any{
			"status":              AssetStatusAttached,
			"attached_lot_id":     lotID,
			"attached_at_unix_ms": now,
			"expires_at_unix_ms":  int64(0),
		}).Error
}

func compactStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (s *Store) FindTemporaryAssetForDelete(ctx context.Context, assetID, ownerUserID string) (*AssetFile, error) {
	if assetID == "" || ownerUserID == "" {
		return nil, errors.New("asset id and owner user id are required")
	}
	var model AssetFileModel
	err := s.db.WithContext(ctx).
		Where("id = ? AND owner_user_id = ? AND status = ? AND deleted_at_unix_ms = 0", assetID, ownerUserID, AssetStatusTemporary).
		First(&model).Error
	if err != nil {
		return nil, err
	}
	return assetFileFromModel(&model), nil
}

func (s *Store) MarkAssetDeleted(ctx context.Context, assetID, ownerUserID string) error {
	if assetID == "" || ownerUserID == "" {
		return errors.New("asset id and owner user id are required")
	}
	now := time.Now().UnixMilli()
	return s.db.WithContext(ctx).Model(&AssetFileModel{}).
		Where("id = ? AND owner_user_id = ? AND status = ? AND deleted_at_unix_ms = 0", assetID, ownerUserID, AssetStatusTemporary).
		Updates(map[string]any{"status": AssetStatusDeleted, "deleted_at_unix_ms": now}).Error
}

func (s *Store) ListExpiredTemporaryAssets(ctx context.Context, limit int) ([]AssetFile, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var models []AssetFileModel
	now := time.Now().UnixMilli()
	if err := s.db.WithContext(ctx).
		Where("status = ? AND deleted_at_unix_ms = 0 AND expires_at_unix_ms > 0 AND expires_at_unix_ms <= ?", AssetStatusTemporary, now).
		Order("expires_at_unix_ms ASC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, err
	}
	assets := make([]AssetFile, 0, len(models))
	for i := range models {
		assets = append(assets, *assetFileFromModel(&models[i]))
	}
	return assets, nil
}

func (s *Store) MarkAssetDeletedByID(ctx context.Context, assetID string) error {
	if assetID == "" {
		return errors.New("asset id is required")
	}
	now := time.Now().UnixMilli()
	return s.db.WithContext(ctx).Model(&AssetFileModel{}).
		Where("id = ? AND status = ? AND deleted_at_unix_ms = 0", assetID, AssetStatusTemporary).
		Updates(map[string]any{"status": AssetStatusDeleted, "deleted_at_unix_ms": now}).Error
}

func assetFileFromModel(model *AssetFileModel) *AssetFile {
	return &AssetFile{
		ID:               model.ID,
		MainAccountID:    model.MainAccountID,
		OwnerUserID:      model.OwnerUserID,
		RoomID:           model.RoomID,
		BizType:          model.BizType,
		Status:           model.Status,
		AttachedLotID:    model.AttachedLotID,
		StorageProvider:  model.StorageProvider,
		Bucket:           model.Bucket,
		ObjectKey:        model.ObjectKey,
		PublicURL:        model.PublicURL,
		OriginalName:     model.OriginalName,
		MimeType:         model.MimeType,
		SizeBytes:        model.SizeBytes,
		SHA256:           model.SHA256,
		AttachedAtUnixMs: model.AttachedAtUnixMs,
		DeletedAtUnixMs:  model.DeletedAtUnixMs,
		ExpiresAtUnixMs:  model.ExpiresAtUnixMs,
	}
}
