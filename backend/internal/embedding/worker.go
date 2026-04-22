package embedding

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Worker runs ComputeAll on a periodic schedule, keeping user embeddings
// fresh as new engagement data arrives.
type Worker struct {
	embedder *UserEmbedder
	interval time.Duration
}

func NewWorker(embedder *UserEmbedder, interval time.Duration) *Worker {
	return &Worker{embedder: embedder, interval: interval}
}

// Run starts the embedding worker loop. The first cycle fires immediately
// so embeddings are current from startup.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("user embedding worker started", "interval", w.interval)
	w.cycle(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("user embedding worker stopped")
			return
		case <-ticker.C:
			w.cycle(ctx)
		}
	}
}

func (w *Worker) cycle(ctx context.Context) {
	if err := w.embedder.ComputeAll(ctx); err != nil {
		slog.Error("user embedding worker: cycle failed", "error", err)
	}
}

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

		inputs := make([]EmbeddingInput, len(posts))
		for i, p := range posts {
			inputs[i] = BuildEmbeddingPayload(p)
		}

		vecs, modelVersion, err := EmbedBatchResolved(ctx, w.embedder, inputs)
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
		if err := w.repo.BatchStore(ids, vecs, modelVersion); err != nil {
			return fmt.Errorf("backfill: batch store: %w", err)
		}

		total += len(posts)
		slog.Info("embedding backfill progress", "stored", total)
	}
	slog.Info("embedding backfill complete", "total", total)
	return nil
}
