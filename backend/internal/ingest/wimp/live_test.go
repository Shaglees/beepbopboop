package wimp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
)

// Modern wimp.com HTML skeleton with the markers the adapter depends on:
//   - og:title, og:description, og:image
//   - article:published_time
//   - first YouTube iframe in the body
const modernWimpHTML = `<!DOCTYPE html>
<html><head>
<meta property="og:title" content="Owner talks to dog through Ring camera.">
<meta property="og:description" content="Big Brother is always watching.">
<meta property="og:image" content="https://www.wimp.com/wp-content/uploads/2021/03/owner-vfhqcx1pezy.jpg">
<meta property="og:url" content="https://www.wimp.com/owner-talks-to-dog-through-ring-camera/">
<meta property="article:published_time" content="2026-04-21T14:00:07+00:00">
<meta name="keywords" content="Dogs, Funny, Technology, Videos">
<title>Owner talks to dog through Ring camera. — Wimp.com</title>
</head><body>
<article class="entry-content">
<p>Big Brother is always watching.</p>
<iframe src="https://www.youtube.com/embed/VFHQCX1pezY" allowfullscreen></iframe>
</article>
</body></html>`

func TestAdapter_FromLiveURL_ExtractsYouTubeEmbed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(modernWimpHTML))
	}))
	defer ts.Close()

	a := wimp.NewAdapter(wimp.Config{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	v, err := a.FromLiveURL(ctx, ts.URL+"/owner-talks-to-dog-through-ring-camera/")
	if err != nil {
		t.Fatalf("FromLiveURL: %v", err)
	}

	if v.Provider != "youtube" || v.ProviderVideoID != "VFHQCX1pezY" {
		t.Errorf("bad provider/id: %+v", v)
	}
	if v.WatchURL != "https://www.youtube.com/watch?v=VFHQCX1pezY" {
		t.Errorf("unexpected watch_url: %q", v.WatchURL)
	}
	if v.Title != "Owner talks to dog through Ring camera." {
		t.Errorf("unexpected title: %q", v.Title)
	}
	if v.ThumbnailURL == "" {
		t.Errorf("expected og:image to be kept as thumbnail")
	}
	if v.PublishedAt == nil {
		t.Errorf("expected article:published_time to be parsed")
	} else if v.PublishedAt.Format(time.RFC3339) != "2026-04-21T14:00:07Z" {
		t.Errorf("unexpected published_at: %v", v.PublishedAt)
	}
	// Per the "drop wimp attribution" call, source_url should equal the watch URL,
	// not the wimp.com URL.
	if v.SourceURL != v.WatchURL {
		t.Errorf("source_url should track watch_url now that wimp attribution is dropped: got %q want %q", v.SourceURL, v.WatchURL)
	}
}

func TestAdapter_FromLiveURL_NoEmbed_ReturnsErrNoLiveEmbed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><head><title>x</title></head><body><p>no video here</p></body></html>`))
	}))
	defer ts.Close()

	a := wimp.NewAdapter(wimp.Config{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := a.FromLiveURL(ctx, ts.URL+"/empty")
	if !errors.Is(err, wimp.ErrNoLiveEmbed) {
		t.Fatalf("expected ErrNoLiveEmbed, got: %v", err)
	}
}

func TestAdapter_FromLiveURL_UpstreamError_PropagatesStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer ts.Close()

	a := wimp.NewAdapter(wimp.Config{})
	_, err := a.FromLiveURL(context.Background(), ts.URL+"/missing")
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected 404 in error, got: %v", err)
	}
}
