package embedding

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// EmbeddingRepo handles vector embedding storage and retrieval on top of the
// posts table. Vectors are stored in the pgvector `embedding` column.
type EmbeddingRepo struct {
	db *sql.DB
}

func NewEmbeddingRepo(db *sql.DB) *EmbeddingRepo {
	return &EmbeddingRepo{db: db}
}

// StoreEmbedding writes an embedding for the given post.
// Calling it twice for the same post overwrites the first value (idempotent UPDATE).
// Returns an error if postID does not exist.
func (r *EmbeddingRepo) StoreEmbedding(postID string, vec []float32) error {
	return r.StoreEmbeddingWithModel(postID, vec, "")
}

// StoreEmbeddingWithModel writes an embedding and captures model metadata in
// post_embeddings for migration-safe re-indexing.
func (r *EmbeddingRepo) StoreEmbeddingWithModel(postID string, vec []float32, modelVersion string) error {
	if len(vec) == 0 {
		return fmt.Errorf("embedding: empty vector for post %s", postID)
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`UPDATE posts SET embedding = $1::vector WHERE id = $2`,
		vecToString(vec), postID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("embedding: post %s not found", postID)
	}

	if _, err := tx.Exec(`
		INSERT INTO post_embeddings (post_id, embedding, model_version, created_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (post_id) DO UPDATE SET
			embedding = excluded.embedding,
			model_version = excluded.model_version,
			created_at = CURRENT_TIMESTAMP`,
		postID, pq.Array(toFloat64Slice(vec)), modelVersion); err != nil {
		return fmt.Errorf("embedding: upsert post_embeddings: %w", err)
	}

	return tx.Commit()
}

// GetEmbedding retrieves the stored embedding for a post. Returns nil if not set.
func (r *EmbeddingRepo) GetEmbedding(postID string) ([]float32, error) {
	var raw sql.NullString
	err := r.db.QueryRow(
		`SELECT embedding::text FROM posts WHERE id = $1`, postID,
	).Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid {
		return nil, nil
	}
	return parseVec(raw.String)
}

// GetUnembedded returns up to limit posts that have no embedding stored yet.
func (r *EmbeddingRepo) GetUnembedded(limit int) ([]model.Post, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.agent_id, a.name, p.user_id, p.title, p.body,
			p.image_url, p.external_url, p.locality, p.latitude, p.longitude,
			p.post_type, p.visibility, p.display_hint, p.labels, p.images,
			p.status, p.scheduled_at, p.source_published_at, p.created_at,
			p.view_count, p.save_count, p.reaction_count, p.seq
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.embedding IS NULL
		  AND p.status = 'published'
		ORDER BY p.seq DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query unembedded posts: %w", err)
	}
	defer rows.Close()
	return scanPosts(rows)
}

// FindSimilar returns up to limit posts ordered by cosine distance to queryVec.
func (r *EmbeddingRepo) FindSimilar(queryVec []float32, limit int) ([]model.Post, error) {
	if len(queryVec) == 0 {
		return []model.Post{}, nil
	}
	rows, err := r.db.Query(`
		SELECT p.id, p.agent_id, a.name, p.user_id, p.title, p.body,
			p.image_url, p.external_url, p.locality, p.latitude, p.longitude,
			p.post_type, p.visibility, p.display_hint, p.labels, p.images,
			p.status, p.scheduled_at, p.source_published_at, p.created_at,
			p.view_count, p.save_count, p.reaction_count, p.seq
		FROM posts p
		JOIN agents a ON a.id = p.agent_id
		WHERE p.embedding IS NOT NULL
		  AND p.status = 'published'
		ORDER BY p.embedding <=> $1::vector
		LIMIT $2`, vecToString(queryVec), limit)
	if err != nil {
		return nil, fmt.Errorf("query similar posts: %w", err)
	}
	defer rows.Close()
	return scanPosts(rows)
}

// BatchStore stores embeddings for multiple (postID, vec) pairs in a transaction.
func (r *EmbeddingRepo) BatchStore(postIDs []string, vecs [][]float32, modelVersion string) error {
	if len(postIDs) != len(vecs) {
		return fmt.Errorf("embedding: postIDs and vecs length mismatch")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmtPosts, err := tx.Prepare(`UPDATE posts SET embedding = $1::vector WHERE id = $2`)
	if err != nil {
		return err
	}
	defer stmtPosts.Close()

	stmtMeta, err := tx.Prepare(`
		INSERT INTO post_embeddings (post_id, embedding, model_version, created_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (post_id) DO UPDATE SET
			embedding = excluded.embedding,
			model_version = excluded.model_version,
			created_at = CURRENT_TIMESTAMP`)
	if err != nil {
		return err
	}
	defer stmtMeta.Close()

	for i, id := range postIDs {
		if len(vecs[i]) == 0 {
			continue
		}
		if _, err := stmtPosts.Exec(vecToString(vecs[i]), id); err != nil {
			return fmt.Errorf("batch store post %s: %w", id, err)
		}
		if _, err := stmtMeta.Exec(id, pq.Array(toFloat64Slice(vecs[i])), modelVersion); err != nil {
			return fmt.Errorf("batch store metadata for post %s: %w", id, err)
		}
	}
	return tx.Commit()
}

// --- vector serialization helpers ---

func vecToString(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', 6, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func parseVec(s string) ([]float32, error) {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil, nil
	}
	strs := strings.Split(s, ",")
	v := make([]float32, len(strs))
	for i, p := range strs {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, fmt.Errorf("parse vector component %d: %w", i, err)
		}
		v[i] = float32(f)
	}
	return v, nil
}

func toFloat64Slice(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = float64(x)
	}
	return out
}

// --- post row scanner (mirrors repository.scanPost) ---

func scanPosts(rows *sql.Rows) ([]model.Post, error) {
	posts := make([]model.Post, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func scanPost(scanner interface{ Scan(dest ...any) error }) (model.Post, error) {
	var p model.Post
	var imageURL, externalURL, locality, postType, labelsJSON, imagesJSON sql.NullString
	var latitude, longitude sql.NullFloat64
	var scheduledAt, sourcePublishedAt sql.NullTime
	var seq int64

	err := scanner.Scan(
		&p.ID, &p.AgentID, &p.AgentName, &p.UserID,
		&p.Title, &p.Body,
		&imageURL, &externalURL, &locality, &latitude, &longitude,
		&postType, &p.Visibility, &p.DisplayHint, &labelsJSON, &imagesJSON,
		&p.Status, &scheduledAt, &sourcePublishedAt, &p.CreatedAt,
		&p.ViewCount, &p.SaveCount, &p.ReactionCount, &seq,
	)
	if err != nil {
		return p, err
	}
	p.ImageURL = imageURL.String
	p.ExternalURL = externalURL.String
	p.Locality = locality.String
	if latitude.Valid {
		p.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		p.Longitude = &longitude.Float64
	}
	p.PostType = postType.String
	if labelsJSON.Valid {
		json.Unmarshal([]byte(labelsJSON.String), &p.Labels)
	}
	if imagesJSON.Valid {
		p.Images = json.RawMessage(imagesJSON.String)
	}
	if scheduledAt.Valid {
		p.ScheduledAt = &scheduledAt.Time
	}
	if sourcePublishedAt.Valid {
		t := sourcePublishedAt.Time
		p.SourcePublishedAt = &t
	}
	_ = seq
	_ = time.Time{} // used indirectly
	return p, nil
}
