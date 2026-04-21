package wimp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// oEmbed enrichment.
//
// Wimp scrapes give us the first YouTube/Vimeo/etc URL on the page, plus the
// wimp post's own og:title (often generic: "Owner talks to dog through Ring
// camera.") and og:image. oEmbed lets us substitute the *upstream* title and
// channel name, which are almost always more accurate for ranking and
// dedup — e.g. "Owner Tells Dog To Go Back Inside Via Spotlight Cam | RingTV"
// authored by "Ring" versus wimp's anonymized version.
//
// Why not the YouTube Data API v3? It requires a billable API key, has quotas,
// and for our use case doesn't add much beyond what oEmbed already gives us.
// The one thing oEmbed does NOT return is duration — we leave DurationSec at
// 0 for YouTube, which is fine for now because ranking doesn't use it yet.

// oEmbedMaxBodyBytes caps oEmbed response reads. YouTube/Vimeo oEmbed JSON
// is consistently under 1KB; 64KB is the effective upper bound we'll tolerate
// before treating the response as hostile.
const oEmbedMaxBodyBytes = 64 * 1024

// oEmbedResult is the subset of fields we care about. Both YouTube and Vimeo
// use the same JSON oEmbed shape, so a single struct covers them.
type oEmbedResult struct {
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	ProviderName string `json:"provider_name"`
	ThumbnailURL string `json:"thumbnail_url"`
	// Vimeo-only: duration is exposed. YouTube oEmbed does not include it.
	Duration int `json:"duration"`
}

// EnricherConfig controls the oEmbed enricher. Zero value is fine for prod.
type EnricherConfig struct {
	HTTPClient *http.Client
	// Timeout applied to each upstream call. Set to 0 for no timeout (tests).
	Timeout time.Duration
}

// Enricher hits provider oEmbed endpoints and patches a model.Video in place
// with richer metadata. It's safe for concurrent use.
type Enricher struct {
	http    *http.Client
	timeout time.Duration
}

func NewEnricher(cfg EnricherConfig) *Enricher {
	c := cfg.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 8 * time.Second}
	}
	return &Enricher{http: c, timeout: cfg.Timeout}
}

// Enrich fills in v.Title (if empty or "too generic"), v.ChannelTitle, and
// v.ThumbnailURL (if upstream has a higher-quality one) for providers with a
// public oEmbed endpoint. Providers without an endpoint (Twitch, Dailymotion
// via anonymous HTTP, Streamable, raw mp4) are returned unchanged and no
// error is raised — the row is still useful without enrichment.
//
// Enrich never fails the overall ingest: on any oEmbed error we log (via the
// returned error) and leave v alone. Callers should treat the error as
// informational.
func (e *Enricher) Enrich(ctx context.Context, v *model.Video) error {
	endpoint, supported := oEmbedEndpoint(v)
	if !supported {
		return nil
	}

	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("oembed: build request: %w", err)
	}
	req.Header.Set("User-Agent", defaultWimpUserAgent)

	resp, err := e.http.Do(req)
	if err != nil {
		return fmt.Errorf("oembed: fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		// 401/404 from oEmbed typically means the upstream video is
		// private/deleted — that's useful to know but not fatal to ingest.
		return fmt.Errorf("oembed: upstream status %d", resp.StatusCode)
	}
	// oEmbed JSON is ~500 bytes in practice; cap at 64KB so a hostile or
	// broken upstream can't exhaust memory by streaming garbage.
	body, err := io.ReadAll(io.LimitReader(resp.Body, oEmbedMaxBodyBytes))
	if err != nil {
		return fmt.Errorf("oembed: read body: %w", err)
	}
	var r oEmbedResult
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("oembed: parse body: %w", err)
	}

	applyEnrichment(v, r)
	return nil
}

// oEmbedEndpoint returns the canonical oEmbed URL for providers we support.
// The second return value is false when v.Provider has no supported endpoint;
// callers short-circuit in that case.
func oEmbedEndpoint(v *model.Video) (string, bool) {
	switch v.Provider {
	case "youtube":
		if v.WatchURL == "" {
			return "", false
		}
		q := url.Values{}
		q.Set("url", v.WatchURL)
		q.Set("format", "json")
		return "https://www.youtube.com/oembed?" + q.Encode(), true
	case "vimeo":
		if v.WatchURL == "" {
			return "", false
		}
		q := url.Values{}
		q.Set("url", v.WatchURL)
		q.Set("format", "json")
		return "https://vimeo.com/api/oembed.json?" + q.Encode(), true
	default:
		// Dailymotion, Twitch, Streamable, direct mp4: no anonymous oEmbed
		// endpoint we can reliably hit without an app token. Leave as-is.
		return "", false
	}
}

// applyEnrichment patches fields conservatively — we don't clobber scraped
// metadata unless the upstream value is strictly better for this provider.
func applyEnrichment(v *model.Video, r oEmbedResult) {
	if r.Title != "" && shouldReplaceTitle(v.Provider, r.Title, v.Title) {
		v.Title = strings.TrimSpace(r.Title)
	}
	if r.AuthorName != "" && v.ChannelTitle == "" {
		v.ChannelTitle = strings.TrimSpace(r.AuthorName)
	}
	// Only replace thumbnail if the scraped one is empty — wimp's 1280x720 og:image is
	// generally higher quality than YouTube's 480x360 hqdefault.
	if r.ThumbnailURL != "" && v.ThumbnailURL == "" {
		v.ThumbnailURL = r.ThumbnailURL
	}
	// Vimeo duration is in whole seconds.
	if r.Duration > 0 && v.DurationSec == 0 {
		v.DurationSec = r.Duration
	}
}

// shouldReplaceTitle decides per-provider whether to clobber the scraped
// wimp title with the upstream oEmbed title.
//
// YouTube: the creator-authored title is canonical and the correct one for
// dedup and ranking ("Epic Dog Rescue | RingTV"). Wimp's retitled version
// loses information. Always prefer upstream when present.
//
// Vimeo: creator-set titles are often lazy placeholders ("Untitled", "final
// cut 3"). Wimp's editorial title is typically cleaner. Only fall back to
// upstream when wimp didn't give us one.
//
// Other providers don't reach here (no oEmbed endpoint), but for safety we
// default to scraped-wins.
func shouldReplaceTitle(provider, upstream, scraped string) bool {
	upstream = strings.TrimSpace(upstream)
	scraped = strings.TrimSpace(scraped)
	if upstream == "" {
		return false
	}
	if scraped == "" {
		return true
	}
	switch provider {
	case "youtube":
		return true
	case "vimeo":
		return false
	default:
		return false
	}
}
