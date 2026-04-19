package repository

import (
	"database/sql"
	"fmt"
	"strings"
)

type SaveRepo struct {
	db *sql.DB
}

func NewSaveRepo(db *sql.DB) *SaveRepo {
	return &SaveRepo{db: db}
}

// Save records a save event for a user on a post and syncs save_count.
func (r *SaveRepo) Save(postID, userID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO post_events (post_id, user_id, event_type)
		VALUES ($1, $2, 'save')`,
		postID, userID,
	)
	if err != nil {
		return fmt.Errorf("insert save event: %w", err)
	}

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
		return fmt.Errorf("sync save_count: %w", err)
	}

	return tx.Commit()
}

// Unsave records an unsave event for a user on a post and syncs save_count.
func (r *SaveRepo) Unsave(postID, userID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO post_events (post_id, user_id, event_type)
		VALUES ($1, $2, 'unsave')`,
		postID, userID,
	)
	if err != nil {
		return fmt.Errorf("insert unsave event: %w", err)
	}

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
		return fmt.Errorf("sync save_count: %w", err)
	}

	return tx.Commit()
}

// IsSaved returns true if the user's most recent save/unsave event for the post is 'save'.
func (r *SaveRepo) IsSaved(postID, userID string) (bool, error) {
	var eventType string
	err := r.db.QueryRow(`
		SELECT event_type FROM post_events
		WHERE post_id = $1 AND user_id = $2 AND event_type IN ('save', 'unsave')
		ORDER BY created_at DESC
		LIMIT 1`,
		postID, userID,
	).Scan(&eventType)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query saved state: %w", err)
	}
	return eventType == "save", nil
}

// GetSavedForPosts returns a set of post IDs that the user has currently saved.
func (r *SaveRepo) GetSavedForPosts(postIDs []string, userID string) (map[string]bool, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}

	var b strings.Builder
	b.WriteString(`
		SELECT post_id FROM (
			SELECT DISTINCT ON (post_id) post_id, event_type
			FROM post_events
			WHERE user_id = $1
			  AND event_type IN ('save', 'unsave')
			  AND post_id IN (`)

	args := []any{userID}
	for i, id := range postIDs {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "$%d", i+2)
		args = append(args, id)
	}
	b.WriteString(`) ORDER BY post_id, created_at DESC
		) latest WHERE event_type = 'save'`)

	rows, err := r.db.Query(b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query saved posts: %w", err)
	}
	defer rows.Close()

	saved := make(map[string]bool)
	for rows.Next() {
		var postID string
		if err := rows.Scan(&postID); err != nil {
			return nil, fmt.Errorf("scan saved post: %w", err)
		}
		saved[postID] = true
	}
	return saved, rows.Err()
}
