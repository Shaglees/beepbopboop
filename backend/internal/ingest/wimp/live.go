package wimp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// liveMaxBodyBytes caps the live-page body read so a hostile or broken
// upstream can't exhaust process memory.
const liveMaxBodyBytes = 2 * 1024 * 1024

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
	md, embed, err := a.fetchLiveInspection(ctx, wimpURL)
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
		SourceURL:   embed.WatchURL,
		SourceDesc:  md.Description,
		Labels:      buildLiveLabels(md),
		EmbedHealth: model.EmbedHealthUnknown,
	}
	if md.PublishedAt != nil {
		v.PublishedAt = md.PublishedAt
	}
	return v, nil
}

// fetchLiveInspection fetches and parses the page. Isolated from FromLiveURL
// to make testing against an httptest.Server trivial.
//
// PublishedAt is extracted by ExtractMetadata via the HTML walker rather than
// a raw string scan, so it's robust to unrelated `article:published_time`
// substrings in JSON-LD or comments.
func (a *Adapter) fetchLiveInspection(ctx context.Context, pageURL string) (Metadata, *Embed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return Metadata{}, nil, fmt.Errorf("wimp live: build request: %w", err)
	}
	req.Header.Set("User-Agent", a.cfg.UserAgent)
	resp, err := a.http.Do(req)
	if err != nil {
		return Metadata{}, nil, fmt.Errorf("wimp live: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Metadata{}, nil, fmt.Errorf("wimp live: upstream status %d for %s", resp.StatusCode, pageURL)
	}
	// Wimp post pages are ~200KB in the wild; 2MB cap bounds memory without
	// prematurely truncating unusually heavy pages.
	body, err := io.ReadAll(io.LimitReader(resp.Body, liveMaxBodyBytes))
	if err != nil {
		return Metadata{}, nil, fmt.Errorf("wimp live: read body: %w", err)
	}

	md := ExtractMetadata(body)
	var embedPtr *Embed
	if embed, ok := ExtractEmbed(body); ok {
		e := embed
		embedPtr = &e
	}
	return md, embedPtr, nil
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
		if k == "" || noiseLabels[k] || seen[k] {
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
