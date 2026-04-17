package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type Worker struct {
	postRepo *repository.PostRepo
	interval time.Duration
}

func NewWorker(postRepo *repository.PostRepo, interval time.Duration) *Worker {
	return &Worker{postRepo: postRepo, interval: interval}
}

func (w *Worker) Run(ctx context.Context) {
	slog.Info("scheduler worker started", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler worker stopped")
			return
		case <-ticker.C:
			n, err := w.postRepo.PublishScheduled()
			if err != nil {
				slog.Error("scheduler: publish failed", "error", err)
				continue
			}
			if n > 0 {
				slog.Info("scheduler: published posts", "count", n)
			}
		}
	}
}
