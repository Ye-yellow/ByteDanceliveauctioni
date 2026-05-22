package data

import (
	"context"
	"errors"
)

type AssetFile struct {
	ID              string
	OwnerUserID     string
	RoomID          string
	BizType         string
	StorageProvider string
	Bucket          string
	ObjectKey       string
	PublicURL       string
	OriginalName    string
	MimeType        string
	SizeBytes       int64
	SHA256          string
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
		ID:              asset.ID,
		OwnerUserID:     asset.OwnerUserID,
		RoomID:          asset.RoomID,
		BizType:         asset.BizType,
		StorageProvider: asset.StorageProvider,
		Bucket:          asset.Bucket,
		ObjectKey:       asset.ObjectKey,
		PublicURL:       asset.PublicURL,
		OriginalName:    asset.OriginalName,
		MimeType:        asset.MimeType,
		SizeBytes:       asset.SizeBytes,
		SHA256:          asset.SHA256,
	}
	return s.db.WithContext(ctx).Create(&model).Error
}
