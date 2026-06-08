package searchindex

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

const DefaultHiddenRetention = 7 * 24 * time.Hour

type DocumentSource interface {
	ListLotSearchDocuments(ctx context.Context, updatedAfterMs int64, limit int) ([]LotDocument, error)
}

type SyncWorkerConfig struct {
	Interval        time.Duration
	BatchLimit      int
	HiddenRetention time.Duration
}

type SyncWorker struct {
	source          DocumentSource
	index           *PGVectorIndex
	embedder        *EmbeddingClient
	interval        time.Duration
	batchLimit      int
	hiddenRetention time.Duration
	lastSeenMs      int64
}

type SyncSummary struct {
	Scanned  int
	Skipped  int
	Indexed  int
	Hidden   int
	Purged   int64
	Failures int
	LastSeen int64
}

func NewSyncWorker(source DocumentSource, index *PGVectorIndex, embedder *EmbeddingClient, cfg SyncWorkerConfig) *SyncWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Minute
	}
	if cfg.BatchLimit <= 0 {
		cfg.BatchLimit = 500
	}
	if cfg.HiddenRetention <= 0 {
		cfg.HiddenRetention = DefaultHiddenRetention
	}
	return &SyncWorker{
		source:          source,
		index:           index,
		embedder:        embedder,
		interval:        cfg.Interval,
		batchLimit:      cfg.BatchLimit,
		hiddenRetention: cfg.HiddenRetention,
	}
}

func (w *SyncWorker) Configured() bool {
	return w != nil && w.source != nil && w.index != nil && w.embedder != nil && w.embedder.Configured()
}

func (w *SyncWorker) Start(ctx context.Context) {
	if !w.Configured() {
		return
	}
	go w.run(ctx)
}

func (w *SyncWorker) run(ctx context.Context) {
	_, _ = w.SyncOnce(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = w.SyncOnce(ctx)
		}
	}
}

func (w *SyncWorker) SyncOnce(ctx context.Context) (SyncSummary, error) {
	var summary SyncSummary
	if !w.Configured() {
		return summary, nil
	}
	docs, err := w.source.ListLotSearchDocuments(ctx, w.lastSeenMs, w.batchLimit)
	if err != nil {
		slog.Warn("lot search sync source failed", "error", err)
		return summary, err
	}
	summary.Scanned = len(docs)
	w.purgeHidden(ctx, &summary)
	if len(docs) == 0 {
		if summary.Purged > 0 || summary.Failures > 0 {
			slog.Info("lot search sync finished",
				"scanned", summary.Scanned,
				"indexed", summary.Indexed,
				"hidden", summary.Hidden,
				"purged", summary.Purged,
				"skipped", summary.Skipped,
				"failures", summary.Failures,
				"last_seen_ms", w.lastSeenMs,
			)
		}
		return summary, nil
	}
	for _, doc := range docs {
		if doc.LotUpdatedAtUnixMs > summary.LastSeen {
			summary.LastSeen = doc.LotUpdatedAtUnixMs
		}
	}
	lotIDs := make([]string, 0, len(docs))
	for _, doc := range docs {
		if strings.TrimSpace(doc.LotID) != "" {
			lotIDs = append(lotIDs, doc.LotID)
		}
	}
	existing, err := w.index.ExistingHashes(ctx, lotIDs)
	if err != nil {
		slog.Warn("lot search sync hash lookup failed", "error", err)
		return summary, err
	}
	visible := make([]LotDocument, 0, len(docs))
	for _, doc := range docs {
		doc.EmbeddingModel = w.embedder.Model()
		doc.EmbeddingDimensions = w.embedder.Dimensions()
		doc.EmbeddingHash = doc.Hash(doc.EmbeddingModel, doc.EmbeddingDimensions)
		if existing[doc.LotID] == doc.EmbeddingHash {
			summary.Skipped++
			continue
		}
		if !doc.PublicVisible || strings.TrimSpace(doc.SearchText) == "" {
			if err := w.index.UpsertDocument(ctx, doc, nil); err != nil {
				summary.Failures++
				slog.Warn("lot search sync hide document failed", "lot_id", doc.LotID, "error", err)
				continue
			}
			summary.Hidden++
			continue
		}
		visible = append(visible, doc)
	}
	for start := 0; start < len(visible); start += w.embedder.BatchSize() {
		end := start + w.embedder.BatchSize()
		if end > len(visible) {
			end = len(visible)
		}
		chunk := visible[start:end]
		texts := make([]string, 0, len(chunk))
		for _, doc := range chunk {
			texts = append(texts, doc.SearchText)
		}
		embeddings, err := w.embedder.Embed(ctx, texts)
		if err != nil {
			summary.Failures += len(chunk)
			slog.Warn("lot search sync embedding failed", "count", len(chunk), "error", err)
			continue
		}
		for i, doc := range chunk {
			if err := w.index.UpsertDocument(ctx, doc, embeddings[i]); err != nil {
				summary.Failures++
				slog.Warn("lot search sync upsert failed", "lot_id", doc.LotID, "error", err)
				continue
			}
			summary.Indexed++
		}
	}
	if summary.LastSeen > 0 {
		w.lastSeenMs = summary.LastSeen
	}
	slog.Info("lot search sync finished",
		"scanned", summary.Scanned,
		"indexed", summary.Indexed,
		"hidden", summary.Hidden,
		"purged", summary.Purged,
		"skipped", summary.Skipped,
		"failures", summary.Failures,
		"last_seen_ms", w.lastSeenMs,
	)
	return summary, nil
}

func (w *SyncWorker) purgeHidden(ctx context.Context, summary *SyncSummary) {
	if w == nil || w.index == nil || summary == nil || w.hiddenRetention <= 0 {
		return
	}
	purged, err := w.index.PurgeHiddenOlderThan(ctx, w.hiddenRetention)
	if err != nil {
		summary.Failures++
		slog.Warn("lot search hidden purge failed", "retention", w.hiddenRetention.String(), "error", err)
		return
	}
	summary.Purged = purged
}
