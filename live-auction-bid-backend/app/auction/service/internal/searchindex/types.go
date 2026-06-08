package searchindex

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

const (
	DefaultEmbeddingModel      = "text-embedding-v4"
	DefaultEmbeddingDimensions = 1024
	DefaultEmbeddingBatchSize  = 10
	DefaultSearchLimit         = 20
)

type LotDocument struct {
	LotID               string
	RoomID              string
	MainAccountID       string
	Title               string
	SearchText          string
	Status              string
	CurrentPrice        *v1.Money
	Href                string
	PublicVisible       bool
	LotUpdatedAtUnixMs  int64
	EmbeddingModel      string
	EmbeddingDimensions int
	EmbeddingHash       string
}

type SearchQuery struct {
	Vector []float64
	RoomID string
	LotID  string
	Limit  int
}

func (d LotDocument) Hash(model string, dimensions int) string {
	parts := []string{
		strings.TrimSpace(model),
		strconv.Itoa(dimensions),
		strings.TrimSpace(d.LotID),
		strings.TrimSpace(d.RoomID),
		strings.TrimSpace(d.MainAccountID),
		strings.TrimSpace(d.Title),
		strings.TrimSpace(d.SearchText),
		strings.TrimSpace(d.Status),
		strconv.FormatBool(d.PublicVisible),
		strconv.FormatInt(d.LotUpdatedAtUnixMs, 10),
	}
	if d.CurrentPrice != nil {
		parts = append(parts, strconv.FormatInt(d.CurrentPrice.GetAmount(), 10), d.CurrentPrice.GetCurrency())
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

func CloneMoney(money *v1.Money) *v1.Money {
	if money == nil {
		return nil
	}
	return &v1.Money{Amount: money.GetAmount(), Currency: money.GetCurrency()}
}

func VectorLiteral(vector []float64) string {
	if len(vector) == 0 {
		return ""
	}
	parts := make([]string, 0, len(vector))
	for _, value := range vector {
		parts = append(parts, strconv.FormatFloat(value, 'g', -1, 64))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func NormalizeDimensions(value int) int {
	if value <= 0 {
		return DefaultEmbeddingDimensions
	}
	return value
}

func NormalizeLimit(value int, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	if value <= 0 {
		value = DefaultSearchLimit
	}
	if value > 100 {
		return 100
	}
	return value
}
