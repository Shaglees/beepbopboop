package embedding

import (
	"context"
	"fmt"
	"log/slog"
)

// BackfillWorker embeds all posts that have no embedding stored yet.
// It fetches posts in batches, calls EmbedBatch, then stores results.
type BackfillWorker struct {
	repo      *EmbeddingRepo
	embedder  Embedder
	batchSize int
}

func NewBackfillWorker(repo *EmbeddingRepo, embedder Embedder, batchSize int) *BackfillWorker {
	return &BackfillWorker{repo: repo, embedder: embedder, batchSize: batchSize}
}

// Run processes all unembedded posts until none remain or ctx is cancelled.
func (w *BackfillWorker) Run(ctx context.Context) error {
	total := 0
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		posts, err := w.repo.GetUnembedded(w.batchSize)
		if err != nil {
			return fmt.Errorf("backfill: get unembedded: %w", err)
		}
		if len(posts) == 0 {
			break
		}

		texts := make([]string, len(posts))
		for i, p := range posts {
			texts[i] = BuildEmbeddingInput(p)
		}

		vecs, err := w.embedder.EmbedBatch(ctx, texts)
		if err != nil {
			return fmt.Errorf("backfill: embed batch: %w", err)
		}
		if len(vecs) != len(posts) {
			return fmt.Errorf("backfill: embedder returned %d vecs for %d posts", len(vecs), len(posts))
		}

		ids := make([]string, len(posts))
		for i, p := range posts {
			ids[i] = p.ID
		}
		if err := w.repo.BatchStore(ids, vecs); err != nil {
			return fmt.Errorf("backfill: batch store: %w", err)
		}

		total += len(posts)
		slog.Info("embedding backfill progress", "stored", total)
	}
	slog.Info("embedding backfill complete", "total", total)
	return nil
}
