package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type UserEmbeddingRepo struct {
	db *sql.DB
}

func NewUserEmbeddingRepo(db *sql.DB) *UserEmbeddingRepo {
	return &UserEmbeddingRepo{db: db}
}

// Get returns the stored embedding for a user, or nil if none exists.
func (r *UserEmbeddingRepo) Get(ctx context.Context, userID string) (*model.UserEmbedding, error) {
	var ue model.UserEmbedding
	var f64 pq.Float64Array
	err := r.db.QueryRowContext(ctx, `
		SELECT user_id, embedding, post_count, computed_at
		FROM user_embeddings WHERE user_id = $1`, userID).
		Scan(&ue.UserID, &f64, &ue.PostCount, &ue.ComputedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user_embedding: %w", err)
	}
	ue.Embedding = toFloat32Slice([]float64(f64))
	return &ue, nil
}

// Upsert stores or replaces a user's embedding and the post count that produced it.
func (r *UserEmbeddingRepo) Upsert(ctx context.Context, userID string, embedding []float32, postCount int) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_embeddings (user_id, embedding, post_count, computed_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			embedding   = excluded.embedding,
			post_count  = excluded.post_count,
			computed_at = CURRENT_TIMESTAMP`,
		userID, pq.Array(toFloat64Slice(embedding)), postCount)
	if err != nil {
		return fmt.Errorf("upsert user_embedding: %w", err)
	}
	return nil
}

// GetAll returns all user embeddings as a userID → vector map (used for batch training).
func (r *UserEmbeddingRepo) GetAll(ctx context.Context) (map[string][]float32, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, embedding FROM user_embeddings`)
	if err != nil {
		return nil, fmt.Errorf("get all user_embeddings: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]float32)
	for rows.Next() {
		var userID string
		var f64 pq.Float64Array
		if err := rows.Scan(&userID, &f64); err != nil {
			return nil, fmt.Errorf("scan user_embedding: %w", err)
		}
		result[userID] = toFloat32Slice([]float64(f64))
	}
	return result, rows.Err()
}
