package wimp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// Config controls the Wayback-based Wimp ingest adapter.
//
// BaseURL is typically https://web.archive.org in production; tests override
// it with an httptest.Server URL. HTTPClient is optional.
// UserAgent is sent on all requests; default is conservative.
type Config struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

// Adapter converts an archived wimp.com page into a model.Video candidate.
// It is safe for concurrent use.
type Adapter struct {
	cfg  Config
	cdx  *CDXClient
	http *http.Client
}

// NewAdapter builds an Adapter with the given config.
func NewAdapter(cfg Config) *Adapter {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultCDXBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "beepbopboop-wimp-ingest/1.0 (+https://github.com/Shaglees/beepbopboop)"
	}
	return &Adapter{
		cfg:  cfg,
		cdx:  NewCDXClient(cfg.BaseURL, cfg.HTTPClient),
		http: cfg.HTTPClient,
	}
}

// FromArchivedURL does the full pipeline: CDX lookup, fetch the archived
// HTML in id_-form, parse metadata + embed, populate a model.Video.
//
// Errors:
//   - ErrNoCapture: the archive has no HTTP-200 HTML capture for wimpURL.
//   - ErrNoLiveEmbed: captured HTML has no YouTube/Vimeo reference.
func (a *Adapter) FromArchivedURL(ctx context.Context, wimpURL string) (model.Video, error) {
	cap, err := a.cdx.LatestCapture(ctx, wimpURL)
	if err != nil {
		return model.Video{}, err
	}

	htmlBytes, err := a.fetchArchived(ctx, cap)
	if err != nil {
		return model.Video{}, err
	}

	embed, ok := ExtractEmbed(htmlBytes)
	if !ok {
		return model.Video{}, ErrNoLiveEmbed
	}
	md := ExtractMetadata(htmlBytes)

	v := model.Video{
		Provider:        embed.Provider,
		ProviderVideoID: embed.VideoID,
		WatchURL:        embed.WatchURL,
		EmbedURL:        embed.EmbedURL,
		Title:           md.Title,
		Description:     md.Description,
		ThumbnailURL:    md.ThumbnailURL,
		SourceURL:       cap.IDURL(), // canonical Wayback permalink, not adapter's BaseURL.
		SourceDesc:      md.Description,
		Labels:          buildLabels(cap, md),
		EmbedHealth:     "unknown",
	}
	if t := cap.CaptureTime(); !t.IsZero() {
		// CaptureTime is an UPPER BOUND on the page's publish date, but it's
		// the best we have without hitting the third-party provider.
		tt := t
		v.PublishedAt = &tt
	}
	return v, nil
}

func (a *Adapter) fetchArchived(ctx context.Context, cap Capture) ([]byte, error) {
	// Build the fetch URL against the adapter's BaseURL so tests can point
	// at httptest.Server. The Capture itself retains the real archive.org
	// permalink for the DB row.
	reqURL := a.cfg.BaseURL + "/web/" + cap.Timestamp + "id_/" + cap.Original
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("wimp: build fetch request: %w", err)
	}
	req.Header.Set("User-Agent", a.cfg.UserAgent)
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wimp: fetch archived html: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("wimp: archived fetch returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func buildLabels(cap Capture, md Metadata) []string {
	labels := []string{"wimp"}
	if y := cap.CaptureYear(); y != "" {
		labels = append(labels, y)
	}
	for _, k := range md.Keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || k == "videos" || k == "clips" {
			continue
		}
		labels = append(labels, k)
	}
	return labels
}
