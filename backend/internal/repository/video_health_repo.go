package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// ListForEmbedHealthCheck returns candidate videos in the order the
// reconciliation worker should check them:
//  1. embed_health = 'unknown'
//  2. stale rows (embed_checked_at older than staleAfter)
//  3. most recently surfaced videos (approximated via recent publications)
//
// Rows already marked blocked/gone are excluded because the worker's role is to
// validate still-eligible content, not resurrect terminal states.
func (r *VideoRepo) ListForEmbedHealthCheck(ctx context.Context, staleAfter time.Duration, limit int) ([]model.Video, error) {
	if limit <= 0 {
		limit = 50
	}
	staleBefore := time.Now().Add(-staleAfter)
	rows, err := r.db.QueryContext(ctx, `
		WITH recent_publications AS (
			SELECT video_id, MAX(published_at) AS last_published_at
			FROM video_post_history
			GROUP BY video_id
		)
		SELECT
			vc.id, vc.provider, vc.provider_video_id, vc.watch_url, vc.embed_url,
			vc.title, vc.description, vc.channel_title, vc.thumbnail_url,
			vc.duration_sec, vc.published_at, vc.source_url, vc.source_description,
			vc.labels, vc.supports_preview_cap, vc.embed_health, vc.embed_checked_at, vc.created_at
		FROM video_catalog vc
		LEFT JOIN recent_publications rp ON rp.video_id = vc.id
		WHERE vc.embed_health IN ('unknown', 'ok')
		ORDER BY
			CASE
				WHEN vc.embed_health = 'unknown' THEN 0
				WHEN vc.embed_checked_at IS NULL OR vc.embed_checked_at < $1 THEN 1
				ELSE 2
			END,
			COALESCE(rp.last_published_at, 'epoch'::timestamptz) DESC,
			COALESCE(vc.embed_checked_at, 'epoch'::timestamptz) ASC,
			COALESCE(vc.published_at, vc.created_at) DESC
		LIMIT $2`, staleBefore, limit)
	if err != nil {
		return nil, fmt.Errorf("list embed health candidates: %w", err)
	}
	defer rows.Close()
	return scanVideoRows(rows)
}
