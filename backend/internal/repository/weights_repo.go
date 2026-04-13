package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type WeightsRepo struct {
	db *sql.DB
}

func NewWeightsRepo(db *sql.DB) *WeightsRepo {
	return &WeightsRepo{db: db}
}

// Get returns the user's weights, or nil when none exist.
func (r *WeightsRepo) Get(userID string) (*model.UserWeights, error) {
	var w model.UserWeights
	var weightsRaw []byte

	err := r.db.QueryRow(`
		SELECT user_id, weights, updated_at
		FROM user_weights WHERE user_id = $1`, userID,
	).Scan(&w.UserID, &weightsRaw, &w.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user_weights: %w", err)
	}
	w.Weights = json.RawMessage(weightsRaw)
	return &w, nil
}

// Upsert inserts or updates the user's weights.
func (r *WeightsRepo) Upsert(userID string, weights json.RawMessage) (*model.UserWeights, error) {
	_, err := r.db.Exec(`
		INSERT INTO user_weights (user_id, weights, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			weights = excluded.weights,
			updated_at = CURRENT_TIMESTAMP`,
		userID, weights,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user_weights: %w", err)
	}
	return r.Get(userID)
}
