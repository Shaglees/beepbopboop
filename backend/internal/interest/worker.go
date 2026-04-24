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

func (w *Worker) RunOnce(ctx context.Context) error {
	rows, err := w.db.QueryContext(ctx, `
		WITH engagement AS (
			SELECT pe.user_id,
				unnest(ARRAY(SELECT jsonb_array_elements_text(p.labels::jsonb))) AS label,
				CASE
					WHEN pe.event_type = 'save' THEN 3.0
					WHEN pe.event_type = 'dwell' THEN 1.0
					ELSE 0.0
				END AS weight
			FROM post_events pe
			JOIN posts p ON p.id = pe.post_id
			WHERE pe.created_at > NOW() - INTERVAL '30 days'
			  AND p.labels IS NOT NULL AND p.labels != ''

			UNION ALL

			SELECT pr.user_id,
				unnest(ARRAY(SELECT jsonb_array_elements_text(p.labels::jsonb))) AS label,
				CASE WHEN pr.reaction = 'more' THEN 5.0 ELSE 0.0 END AS weight
			FROM post_reactions pr
			JOIN posts p ON p.id = pr.post_id
			WHERE pr.reaction = 'more'
			  AND pr.created_at > NOW() - INTERVAL '30 days'
			  AND p.labels IS NOT NULL AND p.labels != ''
		)
		SELECT user_id, label, SUM(weight) AS score
		FROM engagement
		GROUP BY user_id, label
		HAVING SUM(weight) >= 5.0
		ORDER BY score DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var userID, label string
		var score float64
		if err := rows.Scan(&userID, &label, &score); err != nil {
			continue
		}
		confidence := score / 30.0 // 30 weighted points = 1.0 confidence
		if confidence > 1.0 {
			confidence = 1.0
		}
		if err := w.interestRepo.UpsertInferred(userID, label, label, confidence); err != nil {
			slog.Warn("failed to upsert inferred interest",
				"user_id", userID, "label", label, "error", err)
		}
		count++
	}

	slog.Info("interest inference complete", "interests_upserted", count)
	return rows.Err()
}
