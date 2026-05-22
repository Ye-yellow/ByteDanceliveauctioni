package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/data"
	"live-auction-bid/backend/app/auction/service/internal/pkg/auth"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
	"live-auction-bid/backend/app/auction/service/internal/storage"
)

const maxImageUploadBytes = 5 << 20

type assetStore interface {
	SaveAssetFile(ctx context.Context, asset data.AssetFile) error
	FindTemporaryAssetForDelete(ctx context.Context, assetID, ownerUserID string) (*data.AssetFile, error)
	MarkAssetDeleted(ctx context.Context, assetID, ownerUserID string) error
}

type uploadHandler struct {
	auth    *auth.Manager
	store   assetStore
	storage storage.StorageProvider
}

type uploadImageResponse struct {
	Code             int              `json:"code"`
	Message          string           `json:"message"`
	RequestID        string           `json:"requestId"`
	ServerTimeUnixMs int64            `json:"serverTimeUnixMs"`
	Data             *uploadImageData `json:"data,omitempty"`
	// Asset is kept temporarily for old clients. New clients should read data.asset.
	Asset *uploadedAsset `json:"asset,omitempty"`
}

type uploadImageData struct {
	Asset *uploadedAsset `json:"asset"`
}

type uploadedAsset struct {
	ID              string `json:"id"`
	ImageURL        string `json:"imageUrl"`
	Bucket          string `json:"bucket"`
	ObjectKey       string `json:"objectKey"`
	MimeType        string `json:"mimeType"`
	SizeBytes       int64  `json:"sizeBytes"`
	Status          string `json:"status"`
	ExpiresAtUnixMs int64  `json:"expiresAtUnixMs"`
}

func registerUploadHTTP(srv interface {
	HandleFunc(string, http.HandlerFunc)
}, authManager *auth.Manager, store assetStore, provider storage.StorageProvider) {
	h := &uploadHandler{auth: authManager, store: store, storage: provider}
	srv.HandleFunc("/api/uploads/images", h.handleImageUpload)
	srv.HandleFunc("/api/uploads/images/", h.handleImageDelete)
}

func (h *uploadHandler) handleImageUpload(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	requestID := uploadRequestID(r)
	w.Header().Set("X-Request-Id", requestID)
	logUploadInfo("upload_image.request", requestID, "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	if r.Method != http.MethodPost {
		writeUploadError(w, http.StatusMethodNotAllowed, requestID, "method not allowed")
		return
	}
	claims, err := h.authenticate(r)
	if err != nil {
		writeUploadError(w, http.StatusUnauthorized, requestID, err.Error())
		return
	}
	if claims.Role != v1.UserRole_USER_ROLE_ANCHOR && claims.Role != v1.UserRole_USER_ROLE_OPERATOR && claims.Role != v1.UserRole_USER_ROLE_ADMIN {
		writeUploadError(w, http.StatusForbidden, requestID, "permission denied")
		return
	}
	if h.storage == nil {
		writeUploadError(w, http.StatusServiceUnavailable, requestID, "image storage provider is not configured")
		return
	}
	if h.store == nil {
		writeUploadError(w, http.StatusServiceUnavailable, requestID, "asset store is not configured")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageUploadBytes+1024)
	if err := r.ParseMultipartForm(maxImageUploadBytes + 1024); err != nil {
		writeUploadError(w, http.StatusBadRequest, requestID, "invalid multipart image upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeUploadError(w, http.StatusBadRequest, requestID, "file is required")
		return
	}
	defer file.Close()
	logUploadInfo("upload_image.file_received", requestID, "file_name", filepath.Base(header.Filename), "declared_size", header.Size, "content_type", header.Header.Get("Content-Type"))

	dataBytes, err := io.ReadAll(io.LimitReader(file, maxImageUploadBytes+1))
	if err != nil {
		writeUploadError(w, http.StatusBadRequest, requestID, "failed to read upload file")
		return
	}
	if len(dataBytes) == 0 {
		writeUploadError(w, http.StatusBadRequest, requestID, "file is empty")
		return
	}
	if len(dataBytes) > maxImageUploadBytes {
		writeUploadError(w, http.StatusBadRequest, requestID, "image file must be <= 5MB")
		return
	}

	mimeType, ext, err := validateImageBytes(dataBytes)
	if err != nil {
		writeUploadError(w, http.StatusBadRequest, requestID, err.Error())
		return
	}
	assetID := idgen.New("asset")
	bizType := sanitizeBizType(r.FormValue("bizType"))
	if bizType == "" {
		bizType = "lot_image"
	}
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour).UnixMilli()
	objectKey := fmt.Sprintf("temp/%s/%04d/%02d/%s.%s", bizType, now.Year(), int(now.Month()), assetID, ext)
	sum := sha256.Sum256(dataBytes)
	sha := hex.EncodeToString(sum[:])

	stored, err := h.storage.PutObject(r.Context(), storage.PutObjectInput{
		ObjectKey:   objectKey,
		Reader:      bytes.NewReader(dataBytes),
		SizeBytes:   int64(len(dataBytes)),
		ContentType: mimeType,
	})
	if err != nil {
		writeUploadError(w, http.StatusBadGateway, requestID, "upload image to object storage failed: "+err.Error())
		return
	}
	logUploadInfo("upload_image.storage_put", requestID, "provider", stored.Provider, "bucket", stored.Bucket, "object_key", stored.ObjectKey, "duration_ms", time.Since(startedAt).Milliseconds())
	asset := data.AssetFile{
		ID:              assetID,
		OwnerUserID:     claims.UserID,
		RoomID:          strings.TrimSpace(r.FormValue("roomId")),
		BizType:         bizType,
		Status:          data.AssetStatusTemporary,
		StorageProvider: stored.Provider,
		Bucket:          stored.Bucket,
		ObjectKey:       stored.ObjectKey,
		PublicURL:       stored.PublicURL,
		OriginalName:    filepath.Base(header.Filename),
		MimeType:        mimeType,
		SizeBytes:       int64(len(dataBytes)),
		SHA256:          sha,
		ExpiresAtUnixMs: expiresAt,
	}
	if err := h.store.SaveAssetFile(r.Context(), asset); err != nil {
		_ = h.storage.DeleteObject(context.Background(), stored.ObjectKey)
		writeUploadError(w, http.StatusInternalServerError, requestID, "save asset file failed: "+err.Error())
		return
	}
	responseAsset := &uploadedAsset{ID: asset.ID, ImageURL: asset.PublicURL, Bucket: asset.Bucket, ObjectKey: asset.ObjectKey, MimeType: asset.MimeType, SizeBytes: asset.SizeBytes, Status: asset.Status, ExpiresAtUnixMs: asset.ExpiresAtUnixMs}
	writeJSON(w, http.StatusOK, uploadImageResponse{
		Code:             0,
		Message:          "success",
		RequestID:        requestID,
		ServerTimeUnixMs: time.Now().UnixMilli(),
		Data:             &uploadImageData{Asset: responseAsset},
		Asset:            responseAsset,
	})
	logUploadInfo("upload_image.success", requestID, "asset_id", asset.ID, "image_url", asset.PublicURL, "size_bytes", asset.SizeBytes, "mime_type", asset.MimeType, "duration_ms", time.Since(startedAt).Milliseconds())
}

func (h *uploadHandler) handleImageDelete(w http.ResponseWriter, r *http.Request) {
	requestID := uploadRequestID(r)
	w.Header().Set("X-Request-Id", requestID)
	if r.Method != http.MethodDelete {
		writeUploadError(w, http.StatusMethodNotAllowed, requestID, "method not allowed")
		return
	}
	claims, err := h.authenticate(r)
	if err != nil {
		writeUploadError(w, http.StatusUnauthorized, requestID, err.Error())
		return
	}
	assetID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/uploads/images/"), "/")
	if assetID == "" {
		writeUploadError(w, http.StatusBadRequest, requestID, "asset id is required")
		return
	}
	asset, err := h.store.FindTemporaryAssetForDelete(r.Context(), assetID, claims.UserID)
	if err != nil {
		writeUploadError(w, http.StatusNotFound, requestID, "temporary asset not found or already attached")
		return
	}
	if h.storage != nil {
		if err := h.storage.DeleteObject(r.Context(), asset.ObjectKey); err != nil {
			writeUploadError(w, http.StatusBadGateway, requestID, "delete image from object storage failed: "+err.Error())
			return
		}
	}
	if err := h.store.MarkAssetDeleted(r.Context(), asset.ID, claims.UserID); err != nil {
		writeUploadError(w, http.StatusInternalServerError, requestID, "mark asset deleted failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, uploadImageResponse{Code: 0, Message: "success", RequestID: requestID, ServerTimeUnixMs: time.Now().UnixMilli()})
	logUploadInfo("upload_image.deleted", requestID, "asset_id", asset.ID, "object_key", asset.ObjectKey)
}

func (h *uploadHandler) authenticate(r *http.Request) (*auth.Claims, error) {
	if h.auth == nil {
		return nil, errors.New("auth manager is not configured")
	}
	value := r.Header.Get("Authorization")
	if value == "" {
		value = r.Header.Get("authorization")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return nil, errors.New("unauthenticated")
	}
	return h.auth.ParseAccessToken(strings.TrimSpace(strings.TrimPrefix(value, prefix)))
}

func writeUploadError(w http.ResponseWriter, status int, requestID, message string) {
	code := uploadErrorCode(status)
	logUploadError("upload_image.failure", requestID, "status", status, "code", code, "message", message)
	writeJSON(w, status, uploadImageResponse{
		Code:             code,
		Message:          message,
		RequestID:        requestID,
		ServerTimeUnixMs: time.Now().UnixMilli(),
	})
}

func logUploadInfo(event, requestID string, attrs ...any) {
	args := append([]any{"event", event, "request_id", requestID}, attrs...)
	slog.Info("auction upload", args...)
}

func logUploadError(event, requestID string, attrs ...any) {
	args := append([]any{"event", event, "request_id", requestID}, attrs...)
	slog.Error("auction upload", args...)
}

func uploadRequestID(r *http.Request) string {
	for _, key := range []string{"X-Request-Id", "X-Request-ID", "X-Trace-Id"} {
		if value := strings.TrimSpace(r.Header.Get(key)); value != "" {
			return value
		}
	}
	return idgen.New("req")
}

func uploadErrorCode(status int) int {
	switch status {
	case http.StatusBadRequest:
		return 400001
	case http.StatusUnauthorized:
		return 401001
	case http.StatusForbidden:
		return 403001
	case http.StatusMethodNotAllowed:
		return 405001
	case http.StatusNotFound:
		return 404001
	case http.StatusBadGateway:
		return 502001
	case http.StatusServiceUnavailable:
		return 503001
	default:
		return status*1000 + 1
	}
}

func validateImageBytes(data []byte) (string, string, error) {
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "image/jpeg", "jpg", nil
	}
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "image/png", "png", nil
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp", "webp", nil
	}
	return "", "", errors.New("only jpeg, png and webp images are allowed")
}

var bizTypeRe = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)

func sanitizeBizType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = bizTypeRe.ReplaceAllString(value, "_")
	if len(value) > 64 {
		value = value[:64]
	}
	return value
}
