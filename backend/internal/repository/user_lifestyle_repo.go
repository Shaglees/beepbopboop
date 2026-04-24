package repository

import (
	"database/sql"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserLifestyleRepo struct {
	db *sql.DB
}

func NewUserLifestyleRepo(db *sql.DB) *UserLifestyleRepo {
	return &UserLifestyleRepo{db: db}
}

func (r *UserLifestyleRepo) BulkSet(userID string, tags []model.LifestyleTag) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM user_lifestyle_tags WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("delete existing tags: %w", err)
	}

	for _, tag := range tags {
		id, err := generateID()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
		_, err = tx.Exec(`
			INSERT INTO user_lifestyle_tags (id, user_id, tag_category, tag_value)
			VALUES ($1, $2, $3, $4)`,
			id, userID, tag.Category, tag.Value,
		)
		if err != nil {
			return fmt.Errorf("insert tag: %w", err)
		}
	}

	return tx.Commit()
}

func (r *UserLifestyleRepo) List(userID string) ([]model.LifestyleTag, error) {
	rows, err := r.db.Query(`
		SELECT id, tag_category, tag_value FROM user_lifestyle_tags
		WHERE user_id = $1 ORDER BY tag_category, tag_value`, userID)
	if err != nil {
		return nil, fmt.Errorf("list lifestyle tags: %w", err)
	}
	defer rows.Close()

	var result []model.LifestyleTag
	for rows.Next() {
		var tag model.LifestyleTag
		if err := rows.Scan(&tag.ID, &tag.Category, &tag.Value); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}
