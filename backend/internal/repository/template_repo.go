package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type TemplateRepo struct {
	db *sql.DB
}

func NewTemplateRepo(db *sql.DB) *TemplateRepo {
	return &TemplateRepo{db: db}
}

func (r *TemplateRepo) ListByUserID(userID string) ([]model.DisplayTemplate, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, hint_name, definition, created_at
		FROM display_templates
		WHERE user_id = $1
		ORDER BY hint_name`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query display_templates: %w", err)
	}
	defer rows.Close()

	templates := make([]model.DisplayTemplate, 0)
	for rows.Next() {
		var t model.DisplayTemplate
		var defRaw []byte
		if err := rows.Scan(&t.ID, &t.UserID, &t.HintName, &defRaw, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan display_template: %w", err)
		}
		t.Definition = json.RawMessage(defRaw)
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *TemplateRepo) GetByHint(userID, hintName string) (*model.DisplayTemplate, error) {
	var t model.DisplayTemplate
	var defRaw []byte

	err := r.db.QueryRow(`
		SELECT id, user_id, hint_name, definition, created_at
		FROM display_templates
		WHERE user_id = $1 AND hint_name = $2`, userID, hintName,
	).Scan(&t.ID, &t.UserID, &t.HintName, &defRaw, &t.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query display_template: %w", err)
	}
	t.Definition = json.RawMessage(defRaw)
	return &t, nil
}

func (r *TemplateRepo) Upsert(userID, hintName string, definition json.RawMessage) (*model.DisplayTemplate, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO display_templates (id, user_id, hint_name, definition)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(user_id, hint_name) DO UPDATE SET
			definition = excluded.definition`,
		id, userID, hintName, definition,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert display_template: %w", err)
	}
	return r.GetByHint(userID, hintName)
}

func (r *TemplateRepo) Delete(userID, hintName string) error {
	_, err := r.db.Exec(`
		DELETE FROM display_templates
		WHERE user_id = $1 AND hint_name = $2`, userID, hintName,
	)
	if err != nil {
		return fmt.Errorf("delete display_template: %w", err)
	}
	return nil
}
