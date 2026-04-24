package interest

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type Worker struct {
	db           *sql.DB
	interestRepo *repository.UserInterestRepo
}

func NewWorker(db *sql.DB, interestRepo *repository.UserInterestRepo) *Worker {
	return &Worker{db: db, interestRepo: interestRepo}
}

func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	slog.Info("interest inference worker started", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil {
				slog.Warn("interest inference failed", "error", err)
			}
		}
	}
}

type labelEngagement struct {
	UserID string
	Label  string
	Saves  int
}

func (w *Worker) RunOnce(ctx context.Context) error {
	// Find labels with high save engagement per user over last 30 days
	rows, err := w.db.QueryContext(ctx, `
		SELECT pe.user_id, unnest(string_to_array(trim(both '[]"' from p.labels), '","')) AS label,
			COUNT(*) AS saves
		FROM post_events pe
		JOIN posts p ON p.id = pe.post_id
		WHERE pe.event_type = 'save'
		  AND pe.created_at > NOW() - INTERVAL '30 days'
		  AND p.labels IS NOT NULL AND p.labels != ''
		GROUP BY pe.user_id, label
		HAVING COUNT(*) >= 3
		ORDER BY saves DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var engagements []labelEngagement
	for rows.Next() {
		var e labelEngagement
		if err := rows.Scan(&e.UserID, &e.Label, &e.Saves); err != nil {
			continue
		}
		engagements = append(engagements, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, e := range engagements {
		confidence := float64(e.Saves) / 20.0 // 20 saves = 1.0 confidence
		if confidence > 1.0 {
			confidence = 1.0
		}
		if err := w.interestRepo.UpsertInferred(e.UserID, e.Label, e.Label, confidence); err != nil {
			slog.Warn("failed to upsert inferred interest",
				"user_id", e.UserID, "label", e.Label, "error", err)
		}
	}

	slog.Info("interest inference complete", "engagements_processed", len(engagements))
	return nil
}
