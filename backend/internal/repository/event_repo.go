package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type EventRepo struct {
	db *sql.DB
}

func NewEventRepo(db *sql.DB) *EventRepo {
	return &EventRepo{db: db}
}

// syncSaveCount recomputes save_count for a post from the source of truth.
// It uses DISTINCT ON to find each user's latest save/unsave event, then counts
// those who ended on "save", so unsave events correctly decrement the count.
func (r *EventRepo) syncSaveCount(postID string) error {
	_, err := r.db.Exec(`
		UPDATE posts SET save_count = (
			SELECT COUNT(*)
			FROM (
				SELECT DISTINCT ON (user_id) event_type
				FROM post_events
				WHERE post_id = $1 AND event_type IN ('save', 'unsave')
				ORDER BY user_id, created_at DESC
			) latest
			WHERE event_type = 'save'
		) WHERE id = $1`, postID)
	if err != nil {
		return fmt.Errorf("sync save_count for post %s: %w", postID, err)
	}
	return nil
}

func (r *EventRepo) Create(postID, userID, eventType string, dwellMs *int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO post_events (post_id, user_id, event_type, dwell_ms)
		VALUES ($1, $2, $3, $4)`,
		postID, userID, eventType, dwellMs,
	)
	if err != nil {
		return fmt.Errorf("insert post_event: %w", err)
	}

	if eventType == "save" || eventType == "unsave" {
		_, err = tx.Exec(`
			UPDATE posts SET save_count = (
				SELECT COUNT(*)
				FROM (
					SELECT DISTINCT ON (user_id) event_type
					FROM post_events
					WHERE post_id = $1 AND event_type IN ('save', 'unsave')
					ORDER BY user_id, created_at DESC
				) latest
				WHERE event_type = 'save'
			) WHERE id = $1`, postID)
		if err != nil {
			return fmt.Errorf("sync save_count for post %s: %w", postID, err)
		}
	}

	return tx.Commit()
}

func (r *EventRepo) BatchCreate(userID string, events []model.EventInput) error {
	if len(events) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("INSERT INTO post_events (post_id, user_id, event_type, dwell_ms) VALUES ")

	args := make([]any, 0, len(events)*4)
	for i, e := range events {
		if i > 0 {
			b.WriteString(", ")
		}
		base := i*4 + 1
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d)", base, base+1, base+2, base+3)
		args = append(args, e.PostID, userID, e.EventType, e.DwellMs)
	}

	_, err := r.db.Exec(b.String(), args...)
	if err != nil {
		return fmt.Errorf("batch insert post_events: %w", err)
	}

	// Update denormalized save_count for any posts that received a save or unsave event.
	synced := map[string]bool{}
	for _, e := range events {
		if e.EventType == "save" || e.EventType == "unsave" {
			synced[e.PostID] = true
		}
	}
	for postID := range synced {
		if err := r.syncSaveCount(postID); err != nil {
			return err
		}
	}
	return nil
}

// Summary returns aggregated engagement stats for a user's posts over the last N days.
// This is what the agent reads to compute preference weights.
func (r *EventRepo) Summary(userID string, days int) (*model.EventSummary, error) {
	summary := &model.EventSummary{PeriodDays: days}

	// Total events
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM post_events
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 day' * $2`,
		userID, days,
	).Scan(&summary.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("count events: %w", err)
	}

	// Label engagement: join post_events → posts → unnest labels
	rows, err := r.db.Query(`
		SELECT label,
			COUNT(*) FILTER (WHERE pe.event_type = 'view') AS views,
			COUNT(*) FILTER (WHERE pe.event_type = 'save') AS saves,
			COUNT(*) FILTER (WHERE pe.event_type = 'click') AS clicks,
			COALESCE(AVG(pe.dwell_ms) FILTER (WHERE pe.event_type = 'view'), 0) AS avg_dwell
		FROM post_events pe
		JOIN posts p ON p.id = pe.post_id,
		LATERAL jsonb_array_elements_text(p.labels::jsonb) AS label
		WHERE pe.user_id = $1 AND pe.created_at > NOW() - INTERVAL '1 day' * $2
		GROUP BY label
		ORDER BY (COUNT(*) FILTER (WHERE pe.event_type = 'save'))::float / GREATEST(COUNT(*) FILTER (WHERE pe.event_type = 'view'), 1) DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query label engagement: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var le model.LabelEngagement
		if err := rows.Scan(&le.Label, &le.Views, &le.Saves, &le.Clicks, &le.AvgDwell); err != nil {
			return nil, fmt.Errorf("scan label engagement: %w", err)
		}
		summary.LabelEngagement = append(summary.LabelEngagement, le)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate label engagement: %w", err)
	}

	// Type engagement
	rows2, err := r.db.Query(`
		SELECT p.post_type,
			COUNT(*) FILTER (WHERE pe.event_type = 'view') AS views,
			COUNT(*) FILTER (WHERE pe.event_type = 'save') AS saves,
			COUNT(*) FILTER (WHERE pe.event_type = 'click') AS clicks,
			COALESCE(AVG(pe.dwell_ms) FILTER (WHERE pe.event_type = 'view'), 0) AS avg_dwell
		FROM post_events pe
		JOIN posts p ON p.id = pe.post_id
		WHERE pe.user_id = $1 AND pe.created_at > NOW() - INTERVAL '1 day' * $2
		GROUP BY p.post_type
		ORDER BY saves DESC`,
		userID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query type engagement: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var te model.TypeEngagement
		if err := rows2.Scan(&te.PostType, &te.Views, &te.Saves, &te.Clicks, &te.AvgDwell); err != nil {
			return nil, fmt.Errorf("scan type engagement: %w", err)
		}
		summary.TypeEngagement = append(summary.TypeEngagement, te)
	}
	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("iterate type engagement: %w", err)
	}

	return summary, nil
}

// CountsForPosts returns view and save counts for a list of post IDs.
func (r *EventRepo) CountsForPosts(postIDs []string) (map[string][2]int, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString(`
		SELECT post_id,
			COUNT(*) FILTER (WHERE event_type = 'view') AS views,
			COUNT(*) FILTER (WHERE event_type = 'save') AS saves
		FROM post_events
		WHERE post_id IN (`)

	args := make([]any, len(postIDs))
	for i, id := range postIDs {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "$%d", i+1)
		args[i] = id
	}
	b.WriteString(") GROUP BY post_id")

	rows, err := r.db.Query(b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query event counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string][2]int)
	for rows.Next() {
		var postID string
		var views, saves int
		if err := rows.Scan(&postID, &views, &saves); err != nil {
			return nil, fmt.Errorf("scan event counts: %w", err)
		}
		counts[postID] = [2]int{views, saves}
	}
	return counts, rows.Err()
}
