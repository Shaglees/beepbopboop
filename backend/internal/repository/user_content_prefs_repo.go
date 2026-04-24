package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserContentPrefsRepo struct {
	db *sql.DB
}

func NewUserContentPrefsRepo(db *sql.DB) *UserContentPrefsRepo {
	return &UserContentPrefsRepo{db: db}
}

func (r *UserContentPrefsRepo) BulkSet(userID string, prefs []model.ContentPref) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM user_content_prefs WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("delete existing prefs: %w", err)
	}

	for _, p := range prefs {
		id, err := generateID()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
		_, err = tx.Exec(`
			INSERT INTO user_content_prefs (id, user_id, category, depth, tone, max_per_day)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			id, userID, p.Category, p.Depth, p.Tone, p.MaxPerDay,
		)
		if err != nil {
			return fmt.Errorf("insert pref: %w", err)
		}
	}

	return tx.Commit()
}

func (r *UserContentPrefsRepo) List(userID string) ([]model.ContentPref, error) {
	rows, err := r.db.Query(`
		SELECT id, category, depth, tone, max_per_day FROM user_content_prefs
		WHERE user_id = $1 ORDER BY category NULLS FIRST`, userID)
	if err != nil {
		return nil, fmt.Errorf("list content prefs: %w", err)
	}
	defer rows.Close()

	var result []model.ContentPref
	for rows.Next() {
		var p model.ContentPref
		if err := rows.Scan(&p.ID, &p.Category, &p.Depth, &p.Tone, &p.MaxPerDay); err != nil {
			return nil, fmt.Errorf("scan pref: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
