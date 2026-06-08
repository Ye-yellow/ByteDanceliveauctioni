package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

type TOSConfig struct {
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PublicBaseURL string
	UseSSL        bool
}

type TOSStorage struct {
	client *tos.ClientV2
	cfg    TOSConfig
}

func NewTOSStorage(cfg TOSConfig) (*TOSStorage, error) {
	if cfg.Endpoint == "" || cfg.Region == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("tos endpoint, region, bucket, access key and secret key are required")
	}
	if cfg.PublicBaseURL == "" {
		scheme := "https"
		if !cfg.UseSSL {
			scheme = "http"
		}
		cfg.PublicBaseURL = fmt.Sprintf("%s://%s.%s", scheme, cfg.Bucket, strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "https://"), "http://"))
	}
	client, err := tos.NewClientV2(cfg.Endpoint, tos.WithRegion(cfg.Region), tos.WithCredentials(tos.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey)))
	if err != nil {
		return nil, err
	}
	return &TOSStorage{client: client, cfg: cfg}, nil
}

func (s *TOSStorage) PutObject(ctx context.Context, input PutObjectInput) (*StoredObject, error) {
	if input.ObjectKey == "" {
		return nil, errors.New("object key is required")
	}
	if input.Reader == nil {
		return nil, errors.New("object reader is required")
	}
	_, err := s.client.PutObjectV2(ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket:        s.cfg.Bucket,
			Key:           input.ObjectKey,
			ContentLength: input.SizeBytes,
			ContentType:   input.ContentType,
			ACL:           enum.ACLPublicRead,
		},
		Content: input.Reader,
	})
	if err != nil {
		return nil, err
	}
	return &StoredObject{
		Provider:  "tos",
		Bucket:    s.cfg.Bucket,
		ObjectKey: input.ObjectKey,
		PublicURL: strings.TrimRight(s.cfg.PublicBaseURL, "/") + "/" + input.ObjectKey,
	}, nil
}

func (s *TOSStorage) DeleteObject(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return errors.New("object key is required")
	}
	_, err := s.client.DeleteObjectV2(ctx, &tos.DeleteObjectV2Input{Bucket: s.cfg.Bucket, Key: objectKey})
	return err
}
