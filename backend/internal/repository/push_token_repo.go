package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type PushTokenRepo struct {
	db *sql.DB
}

func NewPushTokenRepo(db *sql.DB) *PushTokenRepo {
	return &PushTokenRepo{db: db}
}

func (r *PushTokenRepo) Upsert(userID, token, platform string) error {
	_, err := r.db.Exec(`
		INSERT INTO push_tokens (user_id, token, platform, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, token) DO UPDATE SET
			platform   = excluded.platform,
			updated_at = CURRENT_TIMESTAMP`,
		userID, token, platform,
	)
	if err != nil {
		return fmt.Errorf("upsert push_token: %w", err)
	}
	return nil
}

// TopUnseenPosts returns up to limit posts scored by composite engagement
// in the last 24 hours that the given user has not yet viewed.
// Score: save=3, click=2, view=1.
func (r *PushTokenRepo) TopUnseenPosts(userID string, limit int) ([]model.DigestPost, error) {
	rows, err := r.db.Query(`
		WITH engagement AS (
		    SELECT post_id,
		           COALESCE(SUM(CASE
		               WHEN event_type = 'save'  THEN 3
		               WHEN event_type = 'click' THEN 2
		               WHEN event_type = 'view'  THEN 1
		               ELSE 0
		           END), 0) AS score
		    FROM post_events
		    GROUP BY post_id
		)
		SELECT p.id, p.title, p.body
		FROM posts p
		LEFT JOIN engagement e ON e.post_id = p.id
		WHERE p.created_at > NOW() - INTERVAL '24 hours'
		  AND p.status = 'published'
		  AND NOT EXISTS (
		      SELECT 1 FROM post_events pe
		      WHERE pe.post_id = p.id
		        AND pe.user_id = $1
		        AND pe.event_type = 'view'
		  )
		ORDER BY COALESCE(e.score, 0) DESC
		LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query digest posts: %w", err)
	}
	defer rows.Close()

	var posts []model.DigestPost
	for rows.Next() {
		var dp model.DigestPost
		if err := rows.Scan(&dp.ID, &dp.Title, &dp.Body); err != nil {
			return nil, fmt.Errorf("scan digest post: %w", err)
		}
		posts = append(posts, dp)
	}
	return posts, rows.Err()
}
