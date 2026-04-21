package wimp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// FromLiveURL fetches a current wimp.com page (not Wayback), parses it, and
// returns a model.Video normalized candidate.
//
// Modern wimp.com (WordPress + YouTube/Vimeo iframes) is directly fetchable
// without the Wayback round-trip. The Wayback-backed `FromArchivedURL` path
// remains for historical catalog backfill; new ingest should use this.
//
// Errors:
//   - ErrNoLiveEmbed: the page parsed fine but no recognized provider URL was
//     present.
//   - Any other error: network / non-2xx status from wimp.com.
func (a *Adapter) FromLiveURL(ctx context.Context, wimpURL string) (model.Video, error) {
	md, embed, publishedAt, err := a.fetchLiveInspection(ctx, wimpURL)
	if err != nil {
		return model.Video{}, err
	}
	if embed == nil {
		return model.Video{}, ErrNoLiveEmbed
	}

	v := model.Video{
		Provider:        embed.Provider,
		ProviderVideoID: embed.VideoID,
		WatchURL:        embed.WatchURL,
		EmbedURL:        embed.EmbedURL,
		Title:           md.Title,
		Description:     md.Description,
		ThumbnailURL:    md.ThumbnailURL,
		// Per product call (review response): once we have the provider URL,
		// the catalog row tracks the YouTube/Vimeo video, not the wimp.com
		// page. Wimp attribution is dropped here by design. If provenance is
		// ever needed it can be reintroduced via video_source_pages.
		SourceURL:    embed.WatchURL,
		SourceDesc:   md.Description,
		Labels:       buildLiveLabels(md),
		EmbedHealth:  "unknown",
	}
	if publishedAt != nil {
		v.PublishedAt = publishedAt
	}
	return v, nil
}

// fetchLiveInspection fetches and parses the page. Isolated from FromLiveURL
// to make testing against an httptest.Server trivial.
func (a *Adapter) fetchLiveInspection(ctx context.Context, pageURL string) (Metadata, *Embed, *time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return Metadata{}, nil, nil, fmt.Errorf("wimp live: build request: %w", err)
	}
	req.Header.Set("User-Agent", a.cfg.UserAgent)
	resp, err := a.http.Do(req)
	if err != nil {
		return Metadata{}, nil, nil, fmt.Errorf("wimp live: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Metadata{}, nil, nil, fmt.Errorf("wimp live: upstream status %d for %s", resp.StatusCode, pageURL)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Metadata{}, nil, nil, fmt.Errorf("wimp live: read body: %w", err)
	}

	md := ExtractMetadata(body)
	var embedPtr *Embed
	if embed, ok := ExtractEmbed(body); ok {
		e := embed
		embedPtr = &e
	}
	publishedAt := parseArticlePublishedTime(body)
	return md, embedPtr, publishedAt, nil
}

// parseArticlePublishedTime extracts the WordPress
// <meta property="article:published_time" content="2026-04-21T14:00:07+00:00">
// value, which wimp.com exposes on every post.
//
// Returns nil on any parsing failure so callers can fall back to CreatedAt.
func parseArticlePublishedTime(htmlBytes []byte) *time.Time {
	const marker = `article:published_time`
	idx := strings.Index(string(htmlBytes), marker)
	if idx < 0 {
		return nil
	}
	// Scan forward for the next content=" attribute.
	rest := string(htmlBytes[idx:])
	const contentAttr = `content="`
	cidx := strings.Index(rest, contentAttr)
	if cidx < 0 {
		return nil
	}
	rest = rest[cidx+len(contentAttr):]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return nil
	}
	raw := rest[:end]
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &t
}

// buildLiveLabels derives catalog labels from the WordPress metadata keywords
// and the og:site_name signal. Wimp's RSS categories are passed in via the
// ingest orchestrator, not here, so this function stays self-contained.
func buildLiveLabels(md Metadata) []string {
	// Deliberately do NOT stamp "wimp" as a label — per product call, the
	// cached row tracks the YouTube/Vimeo video, not the wimp post. Consumers
	// filter on actual topical labels (e.g. "dogs", "music").
	out := make([]string, 0, len(md.Keywords))
	seen := make(map[string]bool)
	for _, k := range md.Keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || k == "videos" || k == "clips" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return out
}

// ErrLiveFetchFailed wraps any non-classified live fetch failure so callers
// can distinguish it from ErrNoLiveEmbed / ErrNoCapture. Tests and the ingest
// orchestrator switch on this.
var ErrLiveFetchFailed = errors.New("wimp live: fetch failed")
