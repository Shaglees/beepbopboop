package ranking

import (
	"context"
	"log/slog"
	"time"
)

// RetrainWorker periodically checks whether enough new engagement data has
// accumulated since the last model training run to warrant a retrain cycle.
// When the threshold is met it logs a signal; actual retraining is handled
// out-of-band (Python script / GitHub Actions).
type RetrainWorker struct {
	versionRepo *ModelVersionRepo
	minNewPairs int
	interval    time.Duration
}

func NewRetrainWorker(versionRepo *ModelVersionRepo, minNewPairs int, interval time.Duration) *RetrainWorker {
	return &RetrainWorker{
		versionRepo: versionRepo,
		minNewPairs: minNewPairs,
		interval:    interval,
	}
}

// Run starts the worker loop. The first check fires immediately on startup.
func (w *RetrainWorker) Run(ctx context.Context) {
	slog.Info("retrain worker started", "min_new_pairs", w.minNewPairs, "interval", w.interval)
	w.cycle(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("retrain worker stopped")
			return
		case <-ticker.C:
			w.cycle(ctx)
		}
	}
}

func (w *RetrainWorker) cycle(ctx context.Context) {
	ready, err := w.versionRepo.ReadyToRetrain(ctx, w.minNewPairs)
	if err != nil {
		slog.Warn("retrain worker: readiness check failed", "error", err)
		return
	}
	if ready {
		slog.Info("retrain worker: sufficient new engagement data — retraining recommended",
			"min_new_pairs", w.minNewPairs)
	}
}
