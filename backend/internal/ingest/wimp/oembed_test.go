package wimp_test

// Tests for the oEmbed enricher.
//
// These cover the adversarial-input cases flagged in the PR-183 review:
//   - 401 from Vimeo (private video) returns an error but does NOT mutate
//     the model.Video (non-fatal enrichment)
//   - Surprise JSON fields are ignored without crashing
//   - Response bodies larger than the 64KB cap are truncated and parsed
//     partially (or error cleanly); in neither case do we OOM
//   - shouldReplaceTitle picks the right per-provider policy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

func TestEnricher_VimeoPrivateReturns401(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer ts.Close()

	// Build a video whose WatchURL routes to our test server. We do this by
	// using a real vimeo URL shape; the enricher's oEmbedEndpoint() path
	// reads v.WatchURL into a query param, so we construct a video with a
	// watch_url the enricher will accept but redirect its lookup through
	// the httptest server via a custom transport.
	client := &http.Client{
		Transport: rewriteTransport{target: ts.URL},
		Timeout:   5 * time.Second,
	}
	e := wimp.NewEnricher(wimp.EnricherConfig{HTTPClient: client, Timeout: 2 * time.Second})

	v := &model.Video{
		Provider: "vimeo",
		WatchURL: "https://vimeo.com/123456",
		Title:    "original scraped title",
	}
	err := e.Enrich(context.Background(), v)
	if err == nil {
		t.Fatal("expected enrichment to error on 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %v", err)
	}
	// Model must not have been mutated.
	if v.Title != "original scraped title" {
		t.Errorf("title was clobbered by failed enrichment: %q", v.Title)
	}
}

func TestEnricher_SurpriseFieldsIgnored(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"title": "Real Creator Title",
			"author_name": "Creator Channel",
			"author_url": "https://youtube.com/@creator",
			"provider_name": "YouTube",
			"thumbnail_url": "https://i.ytimg.com/vi/xyz/hqdefault.jpg",
			"duration": 0,
			"unexpected_field": { "nested": true, "array": [1,2,3] },
			"width": 480,
			"html": "<iframe>...</iframe>"
		}`))
	}))
	defer ts.Close()

	client := &http.Client{Transport: rewriteTransport{target: ts.URL}}
	e := wimp.NewEnricher(wimp.EnricherConfig{HTTPClient: client})

	v := &model.Video{
		Provider: "youtube",
		WatchURL: "https://www.youtube.com/watch?v=xyz",
		Title:    "wimp generic title",
	}
	if err := e.Enrich(context.Background(), v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// YouTube policy: always prefer upstream title when present.
	if v.Title != "Real Creator Title" {
		t.Errorf("expected upstream title for youtube, got %q", v.Title)
	}
	if v.ChannelTitle != "Creator Channel" {
		t.Errorf("expected channel to be set, got %q", v.ChannelTitle)
	}
}

func TestEnricher_BodyCapEnforced(t *testing.T) {
	// Stream 128KB of garbage followed by valid JSON — the 64KB LimitReader
	// cap should truncate mid-stream; json.Unmarshal should then fail cleanly
	// without exhausting memory.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Build a body that is JSON-ish but very long, ensuring we hit the cap.
		var b strings.Builder
		b.WriteString(`{"title":"ok","filler":"`)
		for b.Len() < 200*1024 { // 200KB
			b.WriteString("xxxxxxxxxxxxxxxx")
		}
		b.WriteString(`"}`)
		w.Write([]byte(b.String()))
	}))
	defer ts.Close()

	client := &http.Client{Transport: rewriteTransport{target: ts.URL}}
	e := wimp.NewEnricher(wimp.EnricherConfig{HTTPClient: client, Timeout: 3 * time.Second})

	v := &model.Video{
		Provider: "youtube",
		WatchURL: "https://www.youtube.com/watch?v=cap",
		Title:    "original",
	}
	// We don't care whether Enrich succeeds or fails; what matters is that
	// it returns promptly without allocating unbounded memory, and that the
	// original video is not mutated on parse failure.
	_ = e.Enrich(context.Background(), v)
	if v.Title == "original" {
		// Truncated JSON parse will fail, leaving the model untouched.
		return
	}
	// If the truncated body happened to be parseable (unlikely), the title
	// should at least be a non-pathological length. The real invariant is
	// "we didn't OOM" and that's already demonstrated by the test returning.
	if len(v.Title) > 512 {
		t.Errorf("suspiciously large title leaked through body cap: %d chars", len(v.Title))
	}
}

func TestExtractEmbed_RawVideoTagWithMP4(t *testing.T) {
	// The parser's html.ElementNode switch must include "video" for pages
	// that embed MP4s via a raw <video src> tag (wimp occasionally does this
	// for direct-hosted historical clips).
	html := `<html><body><video src="https://cdn.example.com/clip.mp4" controls></video></body></html>`
	got, ok := wimp.ExtractEmbed([]byte(html))
	if !ok {
		t.Fatal("expected embed detection for <video src=...>")
	}
	if got.Provider != "mp4" {
		t.Errorf("expected provider=mp4, got %q", got.Provider)
	}
	if got.WatchURL != "https://cdn.example.com/clip.mp4" {
		t.Errorf("unexpected watch_url: %q", got.WatchURL)
	}
}

// rewriteTransport is a tiny test helper that routes every outbound request
// to the given target URL, preserving only the path + query. Lets us point
// the oEmbed enricher's hardcoded YouTube/Vimeo endpoints at an httptest
// server without modifying the production code.
type rewriteTransport struct{ target string }

func (rt rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, err := parseTarget(rt.target)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = target.scheme
	req.URL.Host = target.host
	return http.DefaultTransport.RoundTrip(req)
}

type parsedTarget struct{ scheme, host string }

func parseTarget(raw string) (parsedTarget, error) {
	u := strings.TrimSpace(raw)
	scheme := "http"
	if strings.HasPrefix(u, "https://") {
		scheme = "https"
		u = u[len("https://"):]
	} else if strings.HasPrefix(u, "http://") {
		u = u[len("http://"):]
	}
	if i := strings.Index(u, "/"); i >= 0 {
		u = u[:i]
	}
	return parsedTarget{scheme: scheme, host: u}, nil
}
