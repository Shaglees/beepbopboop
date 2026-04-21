package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// VideoCatalogListParams filters and paginates the public video catalog list.
// This is the backing query for GET /videos (simple, non-personalized).
// For the personalized variant, agents use the existing VideoSelector.
type VideoCatalogListParams struct {
	// Limit caps the number of rows; bounded to [1, 100] at the handler.
	Limit int
	// IncludeLabels: at least one must match (ANY-match). Case-insensitive.
	IncludeLabels []string
	// ExcludeLabels: none may match. Case-insensitive.
	ExcludeLabels []string
	// HealthyOnly: when true, restrict to embed_health='ok' so the caller
	// doesn't waste a post slot on a deleted video.
	HealthyOnly bool
	// Providers: optional whitelist (e.g. ["youtube","vimeo"]).
	Providers []string
}

// ListCatalog returns catalog rows ordered by published_at desc (falling back
// to created_at when published_at is null), filterable by the usual signals
// an agent cares about when picking a video.
func (r *VideoRepo) ListCatalog(ctx context.Context, p VideoCatalogListParams) ([]model.Video, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}

	var (
		where []string
		args  []any
	)
	argn := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if p.HealthyOnly {
		where = append(where, "embed_health = 'ok'")
	} else {
		// Even when the caller didn't ask for healthy-only, skip known-dead rows.
		// Dead rows are noise that nothing should pick as a post.
		where = append(where, "embed_health <> 'dead'")
	}

	if len(p.Providers) > 0 {
		normalized := make([]string, 0, len(p.Providers))
		for _, pr := range p.Providers {
			pr = strings.ToLower(strings.TrimSpace(pr))
			if pr != "" {
				normalized = append(normalized, pr)
			}
		}
		if len(normalized) > 0 {
			placeholders := make([]string, len(normalized))
			for i, pr := range normalized {
				placeholders[i] = argn(pr)
			}
			where = append(where, "provider IN ("+strings.Join(placeholders, ",")+")")
		}
	}

	if len(p.IncludeLabels) > 0 {
		labelsJSON, err := json.Marshal(lowerSet(p.IncludeLabels))
		if err != nil {
			return nil, fmt.Errorf("marshal include_labels: %w", err)
		}
		// JSONB ?| text[] checks "any of these labels present"; we wrap with a
		// jsonb->text cast because our labels column stores lowercased strings.
		where = append(where, "labels ?| ARRAY(SELECT jsonb_array_elements_text("+argn(string(labelsJSON))+"::jsonb))")
	}

	if len(p.ExcludeLabels) > 0 {
		labelsJSON, err := json.Marshal(lowerSet(p.ExcludeLabels))
		if err != nil {
			return nil, fmt.Errorf("marshal exclude_labels: %w", err)
		}
		where = append(where, "NOT (labels ?| ARRAY(SELECT jsonb_array_elements_text("+argn(string(labelsJSON))+"::jsonb)))")
	}

	q := `
		SELECT id, provider, provider_video_id, watch_url, embed_url,
			title, description, channel_title, thumbnail_url,
			duration_sec, published_at, source_url, source_description,
			labels, supports_preview_cap, embed_health, embed_checked_at, created_at
		FROM video_catalog`
	if len(where) > 0 {
		q += "\n\tWHERE " + strings.Join(where, " AND ")
	}
	q += "\n\tORDER BY COALESCE(published_at, created_at) DESC\n\tLIMIT " + argn(p.Limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list video_catalog: %w", err)
	}
	defer rows.Close()

	out := make([]model.Video, 0, p.Limit)
	for rows.Next() {
		var v model.Video
		var title, desc, channel, thumb, sourceURL, sourceDesc sql.NullString
		var labelsJSON []byte
		var duration sql.NullInt64
		var publishedAt, embedCheckedAt sql.NullTime

		if err := rows.Scan(
			&v.ID, &v.Provider, &v.ProviderVideoID, &v.WatchURL, &v.EmbedURL,
			&title, &desc, &channel, &thumb,
			&duration, &publishedAt, &sourceURL, &sourceDesc,
			&labelsJSON, &v.SupportsPrevCap, &v.EmbedHealth, &embedCheckedAt, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan video_catalog row: %w", err)
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
		v.Labels = []string{}
		if len(labelsJSON) > 0 && string(labelsJSON) != "null" {
			_ = json.Unmarshal(labelsJSON, &v.Labels)
			if v.Labels == nil {
				v.Labels = []string{}
			}
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func lowerSet(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
