package embedding

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

const (
	highDwellThresholdMs = 5000 // dwell events above this are "high engagement"
	coldStartMinPosts    = 3    // minimum high-dwell posts before early-signal update fires
)

// ColdStartUpdater recomputes a user's embedding from their first high-dwell
// post interactions. Intended to be called on every dwell event so new users
// get a meaningful vector within their first session.
type ColdStartUpdater struct {
	db          *sql.DB
	userEmbRepo *repository.UserEmbeddingRepo
}

func NewColdStartUpdater(db *sql.DB) *ColdStartUpdater {
	return &ColdStartUpdater{
		db:          db,
		userEmbRepo: repository.NewUserEmbeddingRepo(db),
	}
}

// MaybeRefresh recomputes and stores the user's embedding if they have at least
// coldStartMinPosts distinct high-dwell posts in the last 24 hours. A no-op when
// the threshold has not yet been reached.
func (cs *ColdStartUpdater) MaybeRefresh(ctx context.Context, userID string) error {
	rows, err := cs.db.QueryContext(ctx, `
		SELECT pe.embedding
		FROM post_events evt
		JOIN post_embeddings pe ON pe.post_id = evt.post_id
		WHERE evt.user_id = $1
		  AND evt.event_type = 'view'
		  AND evt.dwell_ms > $2
		  AND evt.created_at > NOW() - INTERVAL '24 hours'
		GROUP BY pe.post_id, pe.embedding`,
		userID, highDwellThresholdMs)
	if err != nil {
		return fmt.Errorf("cold start: query high-dwell posts: %w", err)
	}
	defer rows.Close()

	var sum []float64
	count := 0
	for rows.Next() {
		var f64 pq.Float64Array
		if err := rows.Scan(&f64); err != nil {
			return fmt.Errorf("cold start: scan embedding: %w", err)
		}
		if sum == nil {
			sum = make([]float64, len(f64))
		}
		for i, v := range f64 {
			sum[i] += v
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("cold start: iterate rows: %w", err)
	}

	if count < coldStartMinPosts {
		return nil
	}

	vec := l2normalize(sum)
	if err := cs.userEmbRepo.Upsert(ctx, userID, vec, count); err != nil {
		slog.Warn("cold start: upsert failed", "user_id", userID, "error", err)
		return fmt.Errorf("cold start: upsert embedding: %w", err)
	}
	slog.Info("cold start: early-signal embedding stored",
		"user_id", userID, "posts", count)
	return nil
}
