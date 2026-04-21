package repository

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

type PostEmbeddingRepo struct {
	db *sql.DB
}

func NewPostEmbeddingRepo(db *sql.DB) *PostEmbeddingRepo {
	return &PostEmbeddingRepo{db: db}
}

// Upsert stores or replaces an embedding for a post.
func (r *PostEmbeddingRepo) Upsert(postID string, embedding []float32, modelVersion string) error {
	_, err := r.db.Exec(`
		INSERT INTO post_embeddings (post_id, embedding, model_version, created_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (post_id) DO UPDATE SET
			embedding     = excluded.embedding,
			model_version = excluded.model_version,
			created_at    = CURRENT_TIMESTAMP`,
		postID, pq.Array(toFloat64Slice(embedding)), modelVersion)
	if err != nil {
		return fmt.Errorf("upsert post_embedding: %w", err)
	}
	return nil
}

// Get returns the embedding for a post, or nil if none exists.
func (r *PostEmbeddingRepo) Get(postID string) (*model.PostEmbedding, error) {
	var pe model.PostEmbedding
	var f64 pq.Float64Array
	err := r.db.QueryRow(`
		SELECT post_id, embedding, model_version, created_at
		FROM post_embeddings WHERE post_id = $1`, postID).
		Scan(&pe.PostID, &f64, &pe.ModelVersion, &pe.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get post_embedding: %w", err)
	}
	pe.Embedding = toFloat32Slice([]float64(f64))
	return &pe, nil
}

// GetMany returns post_id → embedding for the given IDs. Missing IDs are omitted.
func (r *PostEmbeddingRepo) GetMany(postIDs []string) (map[string][]float32, error) {
	if len(postIDs) == 0 {
		return map[string][]float32{}, nil
	}
	rows, err := r.db.Query(`
		SELECT post_id, embedding FROM post_embeddings WHERE post_id = ANY($1)`,
		pq.Array(postIDs))
	if err != nil {
		return nil, fmt.Errorf("get post_embeddings batch: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]float32)
	for rows.Next() {
		var id string
		var f64 pq.Float64Array
		if err := rows.Scan(&id, &f64); err != nil {
			return nil, fmt.Errorf("scan post_embedding: %w", err)
		}
		out[id] = toFloat32Slice([]float64(f64))
	}
	return out, rows.Err()
}

// toFloat64Slice converts []float32 to []float64 for PostgreSQL array storage.
func toFloat64Slice(f32 []float32) []float64 {
	f64 := make([]float64, len(f32))
	for i, v := range f32 {
		f64[i] = float64(v)
	}
	return f64
}

// toFloat32Slice converts []float64 from PostgreSQL array storage to []float32.
func toFloat32Slice(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32
}
