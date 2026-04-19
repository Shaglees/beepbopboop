package embedding

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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

// StoreEmbedding writes a 1536-dim embedding for the given post.
// Calling it twice for the same post overwrites the first value (idempotent UPDATE).
func (r *EmbeddingRepo) StoreEmbedding(postID string, vec []float32) error {
	if len(vec) == 0 {
		return fmt.Errorf("embedding: empty vector for post %s", postID)
	}
	_, err := r.db.Exec(
		`UPDATE posts SET embedding = $1::vector WHERE id = $2`,
		vecToString(vec), postID,
	)
	return err
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
		ORDER BY p.embedding <=> $1::vector
		LIMIT $2`, vecToString(queryVec), limit)
	if err != nil {
		return nil, fmt.Errorf("query similar posts: %w", err)
	}
	defer rows.Close()
	return scanPosts(rows)
}

// BatchStore stores embeddings for multiple (postID, vec) pairs in a transaction.
func (r *EmbeddingRepo) BatchStore(postIDs []string, vecs [][]float32) error {
	if len(postIDs) != len(vecs) {
		return fmt.Errorf("embedding: postIDs and vecs length mismatch")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE posts SET embedding = $1::vector WHERE id = $2`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, id := range postIDs {
		if len(vecs[i]) == 0 {
			continue
		}
		if _, err := stmt.Exec(vecToString(vecs[i]), id); err != nil {
			return fmt.Errorf("batch store post %s: %w", id, err)
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
