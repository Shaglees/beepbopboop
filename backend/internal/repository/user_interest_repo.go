package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserInterestRepo struct {
	db *sql.DB
}

func NewUserInterestRepo(db *sql.DB) *UserInterestRepo {
	return &UserInterestRepo{db: db}
}

// BulkSetUser replaces all source='user' interests for the given user.
// Does NOT touch source='inferred' rows.
func (r *UserInterestRepo) BulkSetUser(userID string, interests []model.UserInterest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM user_interests WHERE user_id = $1 AND source = 'user'", userID)
	if err != nil {
		return fmt.Errorf("delete existing user interests: %w", err)
	}

	for _, i := range interests {
		id, err := generateID()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
		_, err = tx.Exec(`
			INSERT INTO user_interests (id, user_id, category, topic, source, confidence)
			VALUES ($1, $2, $3, $4, 'user', $5)`,
			id, userID, i.Category, i.Topic, i.Confidence,
		)
		if err != nil {
			return fmt.Errorf("insert interest: %w", err)
		}
	}

	return tx.Commit()
}

// UpsertInferred inserts or updates an inferred interest.
func (r *UserInterestRepo) UpsertInferred(userID, category, topic string, confidence float64) error {
	id, err := generateID()
	if err != nil {
		return fmt.Errorf("generate id: %w", err)
	}
	_, err = r.db.Exec(`
		INSERT INTO user_interests (id, user_id, category, topic, source, confidence)
		VALUES ($1, $2, $3, $4, 'inferred', $5)
		ON CONFLICT (user_id, category, topic) DO UPDATE SET
			confidence = EXCLUDED.confidence,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_interests.source = 'inferred'`,
		id, userID, category, topic, confidence,
	)
	if err != nil {
		return fmt.Errorf("upsert inferred interest: %w", err)
	}
	return nil
}

// ListActive returns non-dismissed, non-paused interests.
func (r *UserInterestRepo) ListActive(userID string) ([]model.UserInterest, error) {
	return r.list(userID, true)
}

// ListAll returns all interests including paused and dismissed.
func (r *UserInterestRepo) ListAll(userID string) ([]model.UserInterest, error) {
	return r.list(userID, false)
}

func (r *UserInterestRepo) list(userID string, activeOnly bool) ([]model.UserInterest, error) {
	query := `SELECT id, user_id, category, topic, source, confidence, dismissed,
		paused_until, last_asked_at, times_asked, created_at, updated_at
		FROM user_interests WHERE user_id = $1`
	if activeOnly {
		query += ` AND dismissed = FALSE AND (paused_until IS NULL OR paused_until < NOW())`
	}
	query += ` ORDER BY category, topic`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("list interests: %w", err)
	}
	defer rows.Close()

	var result []model.UserInterest
	for rows.Next() {
		var i model.UserInterest
		err := rows.Scan(
			&i.ID, &i.UserID, &i.Category, &i.Topic, &i.Source, &i.Confidence,
			&i.Dismissed, &i.PausedUntil, &i.LastAskedAt, &i.TimesAsked,
			&i.CreatedAt, &i.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan interest: %w", err)
		}
		result = append(result, i)
	}
	return result, rows.Err()
}

// Promote changes an inferred interest to user-declared.
// Returns the number of rows affected (0 if no matching inferred interest found).
func (r *UserInterestRepo) Promote(id, userID string) (int64, error) {
	res, err := r.db.Exec(`
		UPDATE user_interests SET source = 'user', confidence = 1.0, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2 AND source = 'inferred'`, id, userID)
	if err != nil {
		return 0, fmt.Errorf("promote interest: %w", err)
	}
	return res.RowsAffected()
}

// Dismiss marks an interest as dismissed. Only dismisses inferred interests.
// Returns the number of rows affected.
func (r *UserInterestRepo) Dismiss(id, userID string) (int64, error) {
	res, err := r.db.Exec(`
		UPDATE user_interests SET dismissed = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2 AND source = 'inferred'`, id, userID)
	if err != nil {
		return 0, fmt.Errorf("dismiss interest: %w", err)
	}
	return res.RowsAffected()
}

// Pause sets paused_until to N days from now.
// Returns the number of rows affected.
func (r *UserInterestRepo) Pause(id, userID string, days int) (int64, error) {
	pauseUntil := time.Now().AddDate(0, 0, days)
	res, err := r.db.Exec(`
		UPDATE user_interests SET paused_until = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $3`, id, pauseUntil, userID)
	if err != nil {
		return 0, fmt.Errorf("pause interest: %w", err)
	}
	return res.RowsAffected()
}

// MarkAsked records that the system asked the user about a declining interest.
func (r *UserInterestRepo) MarkAsked(id string) error {
	_, err := r.db.Exec(`
		UPDATE user_interests SET last_asked_at = CURRENT_TIMESTAMP,
			times_asked = times_asked + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark asked: %w", err)
	}
	return nil
}

// Delete removes an interest permanently.
func (r *UserInterestRepo) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM user_interests WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete interest: %w", err)
	}
	return nil
}

// UsersWithInterests returns distinct user IDs that have at least one active interest.
func (r *UserInterestRepo) UsersWithInterests() ([]string, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT user_id FROM user_interests
		WHERE dismissed = FALSE
		  AND (paused_until IS NULL OR paused_until < NOW())
		ORDER BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("users with interests: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan user_id: %w", err)
		}
		result = append(result, userID)
	}
	return result, rows.Err()
}
