package embedding

import (
	"context"
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
