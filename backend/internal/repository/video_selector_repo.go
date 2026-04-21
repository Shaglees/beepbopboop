package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// VideoSelectionParams defines the hard filters and optional personalization
// inputs for selecting candidate videos from the catalog.
type VideoSelectionParams struct {
	UserID          string
	Limit           int
	DedupWindowDays int
	IncludeLabels   []string
	ExcludeLabels   []string
	Seed            *int64
	UserEmbedding   []float32
}

// SelectCandidates returns candidate videos ordered by a blended score:
// hard filters first (dedup / labels / embed health), then personalization via
// user embedding similarity when present, freshness, and deterministic or
// nondeterministic exploration.
func (r *VideoRepo) SelectCandidates(ctx context.Context, p VideoSelectionParams) ([]model.Video, error) {
	if p.UserID == "" {
		return nil, fmt.Errorf("select video candidates: userID is required")
	}
	if p.Limit <= 0 {
		p.Limit = 1
	}
	if p.DedupWindowDays <= 0 {
		p.DedupWindowDays = 180
	}

	args := []any{p.UserID, p.DedupWindowDays}
	argIdx := 3

	seedExpr := "random()"
	if p.Seed != nil {
		args = append(args, fmt.Sprintf("%d", *p.Seed))
		seedExpr = fmt.Sprintf("mod(abs(hashtext(vc.id || ':' || $%d))::bigint, 2147483647)::double precision / 2147483647.0", argIdx)
		argIdx++
	}

	whereClauses := []string{
		"r.video_id IS NULL",
		"vc.embed_health IN ('ok', 'unknown')",
	}
	if len(p.IncludeLabels) > 0 {
		includeJSON, err := json.Marshal(p.IncludeLabels)
		if err != nil {
			return nil, fmt.Errorf("marshal include labels: %w", err)
		}
		args = append(args, string(includeJSON))
		whereClauses = append(whereClauses, fmt.Sprintf("vc.labels @> $%d::jsonb", argIdx))
		argIdx++
	}
	if len(p.ExcludeLabels) > 0 {
		args = append(args, pq.Array(p.ExcludeLabels))
		whereClauses = append(whereClauses, fmt.Sprintf("NOT (vc.labels ?| $%d::text[])", argIdx))
		argIdx++
	}

	scoreExpr := fmt.Sprintf("(1.0 / (1.0 + EXTRACT(EPOCH FROM (NOW() - COALESCE(vc.published_at, vc.created_at))) / 86400.0 / 30.0) * 0.55) + (%s * 0.45)", seedExpr)
	joinEmbedding := ""
	if len(p.UserEmbedding) > 0 {
		args = append(args, vecToStringF32(p.UserEmbedding))
		embeddingArg := argIdx
		argIdx++
		joinEmbedding = "LEFT JOIN video_embeddings ve ON ve.video_id = vc.id"
		scoreExpr = fmt.Sprintf("(COALESCE(1.0 - (ve.embedding <=> $%d::vector), 0.0) * 0.55) + (1.0 / (1.0 + EXTRACT(EPOCH FROM (NOW() - COALESCE(vc.published_at, vc.created_at))) / 86400.0 / 30.0) * 0.25) + (%s * 0.20)", embeddingArg, seedExpr)
	}

	args = append(args, p.Limit)
	limitArg := argIdx

	query := fmt.Sprintf(`
		WITH recent AS (
			SELECT video_id
			FROM video_post_history
			WHERE user_id = $1
			  AND published_at >= CURRENT_TIMESTAMP - ('1 day'::interval * $2)
		)
		SELECT
			vc.id, vc.provider, vc.provider_video_id, vc.watch_url, vc.embed_url,
			vc.title, vc.description, vc.channel_title, vc.thumbnail_url,
			vc.duration_sec, vc.published_at, vc.source_url, vc.source_description,
			vc.labels, vc.supports_preview_cap, vc.embed_health, vc.embed_checked_at, vc.created_at
		FROM video_catalog vc
		LEFT JOIN recent r ON r.video_id = vc.id
		%s
		WHERE %s
		ORDER BY %s DESC, COALESCE(vc.published_at, vc.created_at) DESC, vc.id ASC
		LIMIT $%d`,
		joinEmbedding,
		strings.Join(whereClauses, " AND "),
		scoreExpr,
		limitArg,
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select video candidates: %w", err)
	}
	defer rows.Close()
	return scanVideoRows(rows)
}

func scanVideoRows(rows *sql.Rows) ([]model.Video, error) {
	out := make([]model.Video, 0)
	for rows.Next() {
		v, err := scanVideoFromScanner(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate video rows: %w", err)
	}
	return out, nil
}

func scanVideoFromScanner(scanner interface{ Scan(dest ...any) error }) (model.Video, error) {
	var v model.Video
	var title, desc, channel, thumb, sourceURL, sourceDesc sql.NullString
	var labelsJSON []byte
	var duration sql.NullInt64
	var publishedAt, embedCheckedAt sql.NullTime

	err := scanner.Scan(
		&v.ID, &v.Provider, &v.ProviderVideoID, &v.WatchURL, &v.EmbedURL,
		&title, &desc, &channel, &thumb,
		&duration, &publishedAt, &sourceURL, &sourceDesc,
		&labelsJSON, &v.SupportsPrevCap, &v.EmbedHealth, &embedCheckedAt, &v.CreatedAt,
	)
	if err != nil {
		return model.Video{}, fmt.Errorf("scan video: %w", err)
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
	return v, nil
}
