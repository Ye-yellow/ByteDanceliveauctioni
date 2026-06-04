package searchindex

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

type PGVectorConfig struct {
	DSN                 string
	EmbeddingModel      string
	EmbeddingDimensions int
	MaxOpenConns        int
	MaxIdleConns        int
	ConnMaxLifetime     time.Duration
	ConnMaxIdleTime     time.Duration
}

type PGVectorIndex struct {
	db         *sql.DB
	model      string
	dimensions int
}

func NewPGVectorIndex(ctx context.Context, cfg PGVectorConfig) (*PGVectorIndex, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, errors.New("pgvector dsn is required")
	}
	cfg.EmbeddingModel = strings.TrimSpace(cfg.EmbeddingModel)
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = DefaultEmbeddingModel
	}
	cfg.EmbeddingDimensions = NormalizeDimensions(cfg.EmbeddingDimensions)
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 5
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 2
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = 30 * time.Minute
	}
	if cfg.ConnMaxIdleTime <= 0 {
		cfg.ConnMaxIdleTime = 2 * time.Minute
	}
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	index := &PGVectorIndex{db: db, model: cfg.EmbeddingModel, dimensions: cfg.EmbeddingDimensions}
	if err := index.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return index, nil
}

func (i *PGVectorIndex) Close() error {
	if i == nil || i.db == nil {
		return nil
	}
	return i.db.Close()
}

func (i *PGVectorIndex) Model() string {
	if i == nil {
		return DefaultEmbeddingModel
	}
	return i.model
}

func (i *PGVectorIndex) Dimensions() int {
	if i == nil {
		return DefaultEmbeddingDimensions
	}
	return i.dimensions
}

func (i *PGVectorIndex) migrate(ctx context.Context) error {
	if i == nil || i.db == nil {
		return errors.New("pgvector index is not initialized")
	}
	if _, err := i.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return err
	}
	tableSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS auction_lot_search_docs (
  lot_id VARCHAR(64) PRIMARY KEY,
  room_id VARCHAR(64) NOT NULL,
  main_account_id VARCHAR(64) NOT NULL,
  title TEXT NOT NULL,
  search_text TEXT NOT NULL,
  status VARCHAR(64) NOT NULL,
  current_price_amount BIGINT NOT NULL DEFAULT 0,
  current_price_currency VARCHAR(16) NOT NULL DEFAULT '',
  href TEXT NOT NULL,
  public_visible BOOLEAN NOT NULL DEFAULT FALSE,
  lot_updated_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  embedding_model VARCHAR(128) NOT NULL,
  embedding_dimensions INTEGER NOT NULL,
  embedding vector(%d),
  embedding_hash VARCHAR(64) NOT NULL,
  indexed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`, i.dimensions)
	if _, err := i.db.ExecContext(ctx, tableSQL); err != nil {
		return err
	}
	for _, stmt := range []string{
		"CREATE INDEX IF NOT EXISTS idx_lot_search_public_status ON auction_lot_search_docs (public_visible, status)",
		"CREATE INDEX IF NOT EXISTS idx_lot_search_room ON auction_lot_search_docs (room_id)",
		"CREATE INDEX IF NOT EXISTS idx_lot_search_updated ON auction_lot_search_docs (lot_updated_at_unix_ms)",
		"CREATE INDEX IF NOT EXISTS idx_lot_search_hidden_indexed_at ON auction_lot_search_docs (indexed_at) WHERE public_visible = FALSE",
	} {
		if _, err := i.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (i *PGVectorIndex) ExistingHashes(ctx context.Context, lotIDs []string) (map[string]string, error) {
	out := make(map[string]string)
	if i == nil || i.db == nil || len(lotIDs) == 0 {
		return out, nil
	}
	rows, err := i.db.QueryContext(ctx, "SELECT lot_id, embedding_hash FROM auction_lot_search_docs WHERE lot_id = ANY($1)", pq.Array(lotIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var lotID, hash string
		if err := rows.Scan(&lotID, &hash); err != nil {
			return nil, err
		}
		out[lotID] = hash
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (i *PGVectorIndex) UpsertDocument(ctx context.Context, doc LotDocument, embedding []float64) error {
	if i == nil || i.db == nil {
		return errors.New("pgvector index is not initialized")
	}
	doc.EmbeddingModel = strings.TrimSpace(doc.EmbeddingModel)
	if doc.EmbeddingModel == "" {
		doc.EmbeddingModel = i.model
	}
	if doc.EmbeddingDimensions <= 0 {
		doc.EmbeddingDimensions = i.dimensions
	}
	price := CloneMoney(doc.CurrentPrice)
	if price == nil {
		price = &v1.Money{}
	}
	if len(embedding) == 0 {
		return i.upsertDocumentWithoutEmbedding(ctx, doc, price)
	}
	if len(embedding) != i.dimensions {
		return fmt.Errorf("embedding dimensions mismatch: got %d want %d", len(embedding), i.dimensions)
	}
	return i.upsertDocumentWithEmbedding(ctx, doc, price, VectorLiteral(embedding))
}

func (i *PGVectorIndex) PurgeHiddenOlderThan(ctx context.Context, retention time.Duration) (int64, error) {
	if i == nil || i.db == nil {
		return 0, errors.New("pgvector index is not initialized")
	}
	if retention <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-retention)
	result, err := i.db.ExecContext(ctx, `
DELETE FROM auction_lot_search_docs
WHERE public_visible = FALSE
  AND embedding IS NULL
  AND indexed_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (i *PGVectorIndex) upsertDocumentWithEmbedding(ctx context.Context, doc LotDocument, price *v1.Money, vector string) error {
	_, err := i.db.ExecContext(ctx, `
INSERT INTO auction_lot_search_docs (
  lot_id, room_id, main_account_id, title, search_text, status,
  current_price_amount, current_price_currency, href, public_visible,
  lot_updated_at_unix_ms, embedding_model, embedding_dimensions, embedding, embedding_hash, indexed_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10,
  $11, $12, $13, $14::vector, $15, NOW()
)
ON CONFLICT (lot_id) DO UPDATE SET
  room_id = EXCLUDED.room_id,
  main_account_id = EXCLUDED.main_account_id,
  title = EXCLUDED.title,
  search_text = EXCLUDED.search_text,
  status = EXCLUDED.status,
  current_price_amount = EXCLUDED.current_price_amount,
  current_price_currency = EXCLUDED.current_price_currency,
  href = EXCLUDED.href,
  public_visible = EXCLUDED.public_visible,
  lot_updated_at_unix_ms = EXCLUDED.lot_updated_at_unix_ms,
  embedding_model = EXCLUDED.embedding_model,
  embedding_dimensions = EXCLUDED.embedding_dimensions,
  embedding = EXCLUDED.embedding,
  embedding_hash = EXCLUDED.embedding_hash,
  indexed_at = EXCLUDED.indexed_at`,
		doc.LotID, doc.RoomID, doc.MainAccountID, doc.Title, doc.SearchText, doc.Status,
		price.GetAmount(), price.GetCurrency(), doc.Href, doc.PublicVisible,
		doc.LotUpdatedAtUnixMs, doc.EmbeddingModel, doc.EmbeddingDimensions, vector, doc.EmbeddingHash,
	)
	return err
}

func (i *PGVectorIndex) upsertDocumentWithoutEmbedding(ctx context.Context, doc LotDocument, price *v1.Money) error {
	_, err := i.db.ExecContext(ctx, `
INSERT INTO auction_lot_search_docs (
  lot_id, room_id, main_account_id, title, search_text, status,
  current_price_amount, current_price_currency, href, public_visible,
  lot_updated_at_unix_ms, embedding_model, embedding_dimensions, embedding, embedding_hash, indexed_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10,
  $11, $12, $13, NULL, $14, NOW()
)
ON CONFLICT (lot_id) DO UPDATE SET
  room_id = EXCLUDED.room_id,
  main_account_id = EXCLUDED.main_account_id,
  title = EXCLUDED.title,
  search_text = EXCLUDED.search_text,
  status = EXCLUDED.status,
  current_price_amount = EXCLUDED.current_price_amount,
  current_price_currency = EXCLUDED.current_price_currency,
  href = EXCLUDED.href,
  public_visible = EXCLUDED.public_visible,
  lot_updated_at_unix_ms = EXCLUDED.lot_updated_at_unix_ms,
  embedding_model = EXCLUDED.embedding_model,
  embedding_dimensions = EXCLUDED.embedding_dimensions,
  embedding = NULL,
  embedding_hash = EXCLUDED.embedding_hash,
  indexed_at = EXCLUDED.indexed_at`,
		doc.LotID, doc.RoomID, doc.MainAccountID, doc.Title, doc.SearchText, doc.Status,
		price.GetAmount(), price.GetCurrency(), doc.Href, doc.PublicVisible,
		doc.LotUpdatedAtUnixMs, doc.EmbeddingModel, doc.EmbeddingDimensions, doc.EmbeddingHash,
	)
	return err
}

func (i *PGVectorIndex) Search(ctx context.Context, query SearchQuery) ([]LotDocument, error) {
	if i == nil || i.db == nil {
		return nil, errors.New("pgvector index is not initialized")
	}
	vector := VectorLiteral(query.Vector)
	if vector == "" {
		return nil, errors.New("query vector is required")
	}
	limit := NormalizeLimit(query.Limit, DefaultSearchLimit)
	rows, err := i.db.QueryContext(ctx, `
SELECT lot_id, room_id, main_account_id, title, search_text, status,
       current_price_amount, current_price_currency, href, public_visible,
       lot_updated_at_unix_ms, embedding_model, embedding_dimensions, embedding_hash
FROM auction_lot_search_docs
WHERE public_visible = TRUE
  AND embedding IS NOT NULL
  AND ($2 = '' OR room_id = $2)
  AND ($3 = '' OR lot_id = $3)
ORDER BY embedding <=> $1::vector
LIMIT $4`,
		vector, strings.TrimSpace(query.RoomID), strings.TrimSpace(query.LotID), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LotDocument, 0)
	for rows.Next() {
		var doc LotDocument
		var amount int64
		var currency string
		if err := rows.Scan(
			&doc.LotID, &doc.RoomID, &doc.MainAccountID, &doc.Title, &doc.SearchText, &doc.Status,
			&amount, &currency, &doc.Href, &doc.PublicVisible,
			&doc.LotUpdatedAtUnixMs, &doc.EmbeddingModel, &doc.EmbeddingDimensions, &doc.EmbeddingHash,
		); err != nil {
			return nil, err
		}
		doc.CurrentPrice = &v1.Money{Amount: amount, Currency: currency}
		out = append(out, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
