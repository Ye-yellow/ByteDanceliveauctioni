package storage

import (
	"context"
	"io"
)

type PutObjectInput struct {
	ObjectKey   string
	Reader      io.Reader
	SizeBytes   int64
	ContentType string
}

type StoredObject struct {
	Provider  string
	Bucket    string
	ObjectKey string
	PublicURL string
}

type StorageProvider interface {
	PutObject(ctx context.Context, input PutObjectInput) (*StoredObject, error)
	DeleteObject(ctx context.Context, objectKey string) error
}
