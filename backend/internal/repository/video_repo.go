package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// VideoRepo persists the historical video catalog, pgvector embeddings,
// per-user post history, and per-source ingest cursors. See epic #159 and
// issue #161 for the schema it owns.
type VideoRepo struct {
	db *sql.DB
}

func NewVideoRepo(db *sql.DB) *VideoRepo { return &VideoRepo{db: db} }

// UpsertCatalog writes or updates a video row keyed by (provider, provider_video_id).
// Returns the persisted row with its stable `id` and `created_at`.
func (r *VideoRepo) UpsertCatalog(v model.Video) (model.Video, error) {
	if v.Provider == "" || v.ProviderVideoID == "" {
		return model.Video{}, fmt.Errorf("video: provider and provider_video_id are required")
	}
	if v.EmbedURL == "" || v.WatchURL == "" {
		return model.Video{}, fmt.Errorf("video: embed_url and watch_url are required")
	}
	if v.EmbedHealth == "" {
		v.EmbedHealth = "unknown"
	}
	if v.ID == "" {
		v.ID = newVideoID()
	}

	labelsJSON, err := json.Marshal(nonNilLabels(v.Labels))
	if err != nil {
		return model.Video{}, fmt.Errorf("marshal labels: %w", err)
	}

	row := r.db.QueryRow(`
		INSERT INTO video_catalog (
			id, provider, provider_video_id, watch_url, embed_url,
			title, description, channel_title, thumbnail_url,
			duration_sec, published_at, source_url, source_description,
			labels, supports_preview_cap, embed_health
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,
			$10,$11,$12,$13,
			$14::jsonb, $15, $16
		)
		ON CONFLICT (provider, provider_video_id) DO UPDATE SET
			watch_url            = EXCLUDED.watch_url,
			embed_url            = EXCLUDED.embed_url,
			title                = EXCLUDED.title,
			description          = EXCLUDED.description,
			channel_title        = EXCLUDED.channel_title,
			thumbnail_url        = EXCLUDED.thumbnail_url,
			duration_sec         = EXCLUDED.duration_sec,
			published_at         = EXCLUDED.published_at,
			source_url           = EXCLUDED.source_url,
			source_description   = EXCLUDED.source_description,
			labels               = EXCLUDED.labels,
			supports_preview_cap = EXCLUDED.supports_preview_cap
		RETURNING id, provider, provider_video_id, watch_url, embed_url,
			title, description, channel_title, thumbnail_url,
			duration_sec, published_at, source_url, source_description,
			labels, supports_preview_cap, embed_health, embed_checked_at, created_at`,
		v.ID, v.Provider, v.ProviderVideoID, v.WatchURL, v.EmbedURL,
		nullString(v.Title), nullString(v.Description), nullString(v.ChannelTitle), nullString(v.ThumbnailURL),
		nullInt(v.DurationSec), nullTime(v.PublishedAt), nullString(v.SourceURL), nullString(v.SourceDesc),
		string(labelsJSON), v.SupportsPrevCap, v.EmbedHealth,
	)
	return scanVideoRow(row)
}

// GetByID fetches a single catalog row. Returns nil, nil when not found.
func (r *VideoRepo) GetByID(id string) (*model.Video, error) {
	row := r.db.QueryRow(`
		SELECT id, provider, provider_video_id, watch_url, embed_url,
			title, description, channel_title, thumbnail_url,
			duration_sec, published_at, source_url, source_description,
			labels, supports_preview_cap, embed_health, embed_checked_at, created_at
		FROM video_catalog WHERE id = $1`, id)
	return scanVideoRowOptional(row)
}

// GetByProviderID fetches a catalog row by its natural key. Returns nil, nil when not found.
func (r *VideoRepo) GetByProviderID(provider, providerVideoID string) (*model.Video, error) {
	row := r.db.QueryRow(`
		SELECT id, provider, provider_video_id, watch_url, embed_url,
			title, description, channel_title, thumbnail_url,
			duration_sec, published_at, source_url, source_description,
			labels, supports_preview_cap, embed_health, embed_checked_at, created_at
		FROM video_catalog WHERE provider = $1 AND provider_video_id = $2`,
		provider, providerVideoID)
	return scanVideoRowOptional(row)
}

// UpdateEmbedHealth updates both `embed_health` and `embed_checked_at`.
// The health reconciliation worker (#167) is the primary caller.
func (r *VideoRepo) UpdateEmbedHealth(videoID, health string) error {
	res, err := r.db.Exec(`
		UPDATE video_catalog
		   SET embed_health = $2, embed_checked_at = CURRENT_TIMESTAMP
		 WHERE id = $1`, videoID, health)
	if err != nil {
		return fmt.Errorf("update embed_health: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("video %s not found", videoID)
	}
	return nil
}

// UpsertEmbedding stores a 1536-dim embedding for the video. Idempotent on video_id.
func (r *VideoRepo) UpsertEmbedding(videoID string, vec []float32, modelVersion string) error {
	if len(vec) == 0 {
		return fmt.Errorf("video_embedding: empty vector for %s", videoID)
	}
	_, err := r.db.Exec(`
		INSERT INTO video_embeddings (video_id, embedding, model_version, created_at)
		VALUES ($1, $2::vector, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (video_id) DO UPDATE SET
			embedding     = EXCLUDED.embedding,
			model_version = EXCLUDED.model_version,
			created_at    = CURRENT_TIMESTAMP`,
		videoID, vecToStringF32(vec), modelVersion)
	if err != nil {
		return fmt.Errorf("upsert video_embedding: %w", err)
	}
	return nil
}

// GetEmbedding returns the stored vector or nil when no row exists.
func (r *VideoRepo) GetEmbedding(videoID string) ([]float32, error) {
	var raw sql.NullString
	err := r.db.QueryRow(
		`SELECT embedding::text FROM video_embeddings WHERE video_id = $1`, videoID,
	).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get video_embedding: %w", err)
	}
	if !raw.Valid {
		return nil, nil
	}
	return parseVecF32(raw.String)
}

// InsertPostHistory records that a post using `videoID` was just published by `userID`.
// Called from the publish flow (#168) to drive the 180-day dedup window.
func (r *VideoRepo) InsertPostHistory(postID, videoID, userID string) error {
	_, err := r.db.Exec(`
		INSERT INTO video_post_history (post_id, video_id, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (post_id) DO UPDATE SET
			video_id     = EXCLUDED.video_id,
			user_id      = EXCLUDED.user_id,
			published_at = CURRENT_TIMESTAMP`,
		postID, videoID, userID)
	if err != nil {
		return fmt.Errorf("insert video_post_history: %w", err)
	}
	return nil
}

// ListPostHistoryForUserSince returns rows this user published at or after `since`,
// newest first. Selection (#162) uses the set of video_ids here to skip recent dupes.
func (r *VideoRepo) ListPostHistoryForUserSince(userID string, since time.Time) ([]model.VideoPostHistory, error) {
	rows, err := r.db.Query(`
		SELECT post_id, video_id, user_id, published_at
		  FROM video_post_history
		 WHERE user_id = $1 AND published_at >= $2
		 ORDER BY published_at DESC`, userID, since)
	if err != nil {
		return nil, fmt.Errorf("list video_post_history: %w", err)
	}
	defer rows.Close()

	out := make([]model.VideoPostHistory, 0)
	for rows.Next() {
		var h model.VideoPostHistory
		if err := rows.Scan(&h.PostID, &h.VideoID, &h.UserID, &h.PublishedAt); err != nil {
			return nil, fmt.Errorf("scan video_post_history: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// RecordIngest upserts the last cursor for a given ingest source.
func (r *VideoRepo) RecordIngest(source, cursor string) error {
	_, err := r.db.Exec(`
		INSERT INTO video_source_ingest (source, last_cursor, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (source) DO UPDATE SET
			last_cursor = EXCLUDED.last_cursor,
			updated_at  = CURRENT_TIMESTAMP`,
		source, cursor)
	if err != nil {
		return fmt.Errorf("record video_source_ingest: %w", err)
	}
	return nil
}

// GetIngest returns the ingest cursor row for a source. Returns nil, nil on miss.
func (r *VideoRepo) GetIngest(source string) (*model.VideoSourceIngest, error) {
	var s model.VideoSourceIngest
	err := r.db.QueryRow(`
		SELECT source, last_cursor, updated_at FROM video_source_ingest WHERE source = $1`,
		source).Scan(&s.Source, &s.LastCursor, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get video_source_ingest: %w", err)
	}
	return &s, nil
}

// --- internal helpers ---------------------------------------------------------

// scanVideoRow scans a RETURNING-style row that is guaranteed to exist.
func scanVideoRow(row *sql.Row) (model.Video, error) {
	v, err := scanVideoRowOptional(row)
	if err != nil {
		return model.Video{}, err
	}
	if v == nil {
		return model.Video{}, sql.ErrNoRows
	}
	return *v, nil
}

// scanVideoRowOptional scans a SELECT that may return zero rows.
func scanVideoRowOptional(row *sql.Row) (*model.Video, error) {
	var v model.Video
	var title, desc, channel, thumb, sourceURL, sourceDesc sql.NullString
	var labelsJSON []byte
	var duration sql.NullInt64
	var publishedAt, embedCheckedAt sql.NullTime

	err := row.Scan(
		&v.ID, &v.Provider, &v.ProviderVideoID, &v.WatchURL, &v.EmbedURL,
		&title, &desc, &channel, &thumb,
		&duration, &publishedAt, &sourceURL, &sourceDesc,
		&labelsJSON, &v.SupportsPrevCap, &v.EmbedHealth, &embedCheckedAt, &v.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan video_catalog: %w", err)
	}

	v.Title = title.String
	v.Description = desc.String
	v.ChannelTitle = channel.String
	v.ThumbnailURL = thumb.String
	v.SourceURL = sourceURL.String
	v.SourceDesc = sourceDesc.String
	if duration.Valid {
		v.DurationSec = int(duration.Int64)
	}
	if publishedAt.Valid {
		t := publishedAt.Time
		v.PublishedAt = &t
	}
	if embedCheckedAt.Valid {
		t := embedCheckedAt.Time
		v.EmbedCheckedAt = &t
	}
	v.Labels = nonNilLabels(nil)
	if len(labelsJSON) > 0 && string(labelsJSON) != "null" {
		_ = json.Unmarshal(labelsJSON, &v.Labels)
		if v.Labels == nil {
			v.Labels = nonNilLabels(nil)
		}
	}
	return &v, nil
}

func nonNilLabels(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func nullInt(i int) any {
	if i == 0 {
		return nil
	}
	return i
}

// newVideoID returns a short hex id unique within the catalog. Collision
// probability with 16 hex chars is negligible at any realistic scale.
func newVideoID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return "vid_" + hex.EncodeToString(buf[:])
}

// --- pgvector serialization ---------------------------------------------------
//
// Duplicated from internal/embedding/repo.go so the repository package doesn't
// depend on embedding. Keep the two implementations in sync.

func vecToStringF32(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', 6, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func parseVecF32(s string) ([]float32, error) {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil, nil
	}
	strs := strings.Split(s, ",")
	out := make([]float32, len(strs))
	for i, p := range strs {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, fmt.Errorf("parse vector component %d: %w", i, err)
		}
		out[i] = float32(f)
	}
	return out, nil
}
