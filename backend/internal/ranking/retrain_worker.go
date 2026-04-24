package ranking

import (
	"context"
	"log/slog"
	"time"
)

// RetrainWorker periodically checks whether enough new engagement data has
// accumulated since the last model training run to warrant a retrain cycle.
// When the threshold is met it emits a signal exactly once per readiness period
// (debounced) — the signal resets when the condition clears (e.g. after a new
// model is deployed and events drain below the threshold again).
// Actual retraining is handled out-of-band (Python script / GitHub Actions).
type RetrainWorker struct {
	versionRepo *ModelVersionRepo
	minNewPairs int
	interval    time.Duration
	notifyFn    func()
	signaled    bool
}

// NewRetrainWorker creates a worker that logs a structured message when ready.
func NewRetrainWorker(versionRepo *ModelVersionRepo, minNewPairs int, interval time.Duration) *RetrainWorker {
	return &RetrainWorker{
		versionRepo: versionRepo,
		minNewPairs: minNewPairs,
		interval:    interval,
	}
}

// NewRetrainWorkerWithNotify creates a worker that calls notifyFn (instead of
// logging) when the retrain condition is first met. Useful for testing.
func NewRetrainWorkerWithNotify(versionRepo *ModelVersionRepo, minNewPairs int, interval time.Duration, notifyFn func()) *RetrainWorker {
	return &RetrainWorker{
		versionRepo: versionRepo,
		minNewPairs: minNewPairs,
		interval:    interval,
		notifyFn:    notifyFn,
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

	if ready && !w.signaled {
		w.signaled = true
		if w.notifyFn != nil {
			w.notifyFn()
		} else {
			slog.Info("retrain worker: sufficient new engagement data — retraining recommended",
				"min_new_pairs", w.minNewPairs)
		}
	} else if !ready {
		// Reset so the signal fires again if the threshold is crossed after a new deploy.
		w.signaled = false
	}
}
