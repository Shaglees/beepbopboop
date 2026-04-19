package embedding

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// halfLifeDays controls engagement recency decay. Interactions 7 days old
// count for 50% of same-day interactions.
const halfLifeDays = 7.0

// UserEmbedder computes per-user interest vectors from engagement history.
// The vector is a decay-weighted average of post embeddings the user engaged
// with, L2-normalised so it lives on the unit sphere alongside post embeddings.
type UserEmbedder struct {
	db          *sql.DB
	userEmbRepo *repository.UserEmbeddingRepo
}

func NewUserEmbedder(db *sql.DB, userEmbRepo *repository.UserEmbeddingRepo) *UserEmbedder {
	return &UserEmbedder{db: db, userEmbRepo: userEmbRepo}
}

// engagementRow holds per-post aggregated engagement data for one user.
type engagementRow struct {
	postID         string
	embedding      []float64
	saved          bool
	clicked        bool
	views          int
	maxDwellMs     float64
	reaction       sql.NullString
	lastInteractAt time.Time
}

// postWeight returns the signed engagement weight for a single post.
// Mirrors the formula in weights_repo.go (saves=5, clicks=2, views=0.3, dwell bonus).
// Reaction 'more' adds 3.0; 'less'/'not_for_me' subtracts 5.0 (hard negatives).
func postWeight(e engagementRow) float64 {
	base := 0.0
	if e.saved {
		base += 5.0
	}
	if e.clicked {
		base += 2.0
	}
	base += float64(e.views) * 0.3
	if e.maxDwellMs >= 5000 {
		base += math.Min(e.maxDwellMs/10000.0, 1.0)
	}
	if e.reaction.Valid {
		switch e.reaction.String {
		case "more":
			base += 3.0
		case "less", "not_for_me":
			base -= 5.0
		}
	}
	return base
}

// decayFactor returns exp(-λ·days) with half-life = halfLifeDays.
// Returns a value in (0, 1] — fresh interactions score 1.0, older ones less.
func decayFactor(lastInteractAt time.Time) float64 {
	days := time.Since(lastInteractAt).Hours() / 24.0
	lambda := math.Log(2) / halfLifeDays
	return math.Exp(-lambda * days)
}

// ComputeForUser computes the user's interest vector from their last 14 days
// of engagement. Returns (nil, 0, nil) when there is no eligible engagement —
// callers should fall through to the cold-start path.
func (ue *UserEmbedder) ComputeForUser(ctx context.Context, userID string) ([]float32, int, error) {
	rows, err := ue.db.QueryContext(ctx, `
		SELECT
			eng.post_id,
			emb.embedding,
			eng.saved,
			eng.clicked,
			eng.views,
			eng.max_dwell_ms,
			eng.last_interaction_at,
			eng.reaction
		FROM (
			SELECT
				pe.post_id,
				CASE
					WHEN MAX(pe.created_at) FILTER (WHERE pe.event_type = 'save') >
					     COALESCE(MAX(pe.created_at) FILTER (WHERE pe.event_type = 'unsave'), '-infinity'::timestamptz)
					THEN 1 ELSE 0
				END AS saved,
				MAX(CASE WHEN pe.event_type = 'click' THEN 1 ELSE 0 END) AS clicked,
				COUNT(*) FILTER (WHERE pe.event_type = 'view')            AS views,
				COALESCE(MAX(pe.dwell_ms) FILTER (WHERE pe.event_type = 'view'), 0) AS max_dwell_ms,
				COALESCE(
					MAX(pe.created_at) FILTER (WHERE pe.event_type IN ('save', 'unsave', 'click', 'share')),
					MAX(pe.created_at)
				) AS last_interaction_at,
				MAX(pr.reaction) AS reaction
			FROM post_events pe
			LEFT JOIN post_reactions pr
				ON pr.post_id = pe.post_id AND pr.user_id = $1
			WHERE pe.user_id = $1
			  AND pe.created_at > NOW() - INTERVAL '14 days'
			GROUP BY pe.post_id
		) eng
		JOIN post_embeddings emb ON emb.post_id = eng.post_id`,
		userID)
	if err != nil {
		return nil, 0, fmt.Errorf("query engagement: %w", err)
	}
	defer rows.Close()

	var weightedSum []float64
	var totalWeight float64
	var postCount int

	for rows.Next() {
		var e engagementRow
		var f64arr pq.Float64Array
		var savedInt, clickedInt int
		if err := rows.Scan(
			&e.postID, &f64arr,
			&savedInt, &clickedInt, &e.views,
			&e.maxDwellMs, &e.lastInteractAt, &e.reaction,
		); err != nil {
			return nil, 0, fmt.Errorf("scan engagement row: %w", err)
		}
		e.embedding = []float64(f64arr)
		e.saved = savedInt == 1
		e.clicked = clickedInt == 1

		w := postWeight(e) * decayFactor(e.lastInteractAt)
		if w <= 0 {
			continue
		}

		if weightedSum == nil {
			weightedSum = make([]float64, len(e.embedding))
		}
		for i, v := range e.embedding {
			weightedSum[i] += w * v
		}
		totalWeight += w
		postCount++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate engagement rows: %w", err)
	}

	if totalWeight <= 0 || weightedSum == nil {
		return nil, 0, nil
	}

	// L2-normalise so the user vector lives on the unit sphere alongside post vectors.
	var mag float64
	for _, v := range weightedSum {
		mag += v * v
	}
	mag = math.Sqrt(mag)
	if mag < 1e-10 {
		return nil, 0, nil
	}

	result := make([]float32, len(weightedSum))
	for i, v := range weightedSum {
		result[i] = float32(v / mag)
	}
	return result, postCount, nil
}

// ComputeAll recomputes embeddings for every user active in the last 14 days.
// Per-user errors are logged and skipped so a single bad user cannot abort the batch.
func (ue *UserEmbedder) ComputeAll(ctx context.Context) error {
	rows, err := ue.db.QueryContext(ctx, `
		SELECT DISTINCT user_id FROM post_events
		WHERE created_at > NOW() - INTERVAL '14 days'`)
	if err != nil {
		return fmt.Errorf("list active users: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan user_id: %w", err)
		}
		userIDs = append(userIDs, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate active users: %w", err)
	}
	rows.Close() // release connection before per-user queries

	var ok, failed int
	for _, uid := range userIDs {
		vec, postCount, err := ue.ComputeForUser(ctx, uid)
		if err != nil {
			slog.Warn("user embedder: compute failed", "user_id", uid, "error", err)
			failed++
			continue
		}
		if vec == nil {
			continue
		}
		if err := ue.userEmbRepo.Upsert(ctx, uid, vec, postCount); err != nil {
			slog.Warn("user embedder: upsert failed", "user_id", uid, "error", err)
			failed++
			continue
		}
		ok++
	}

	slog.Info("user embedder: batch complete",
		"users", len(userIDs), "ok", ok, "failed", failed)
	return nil
}
