package wimp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
)

// fakeWayback stands in for both CDX search API and the web/{ts}id_ fetch endpoint.
func fakeWayback(t *testing.T, cdxTimestamp, cdxOriginal, htmlFixture string) *httptest.Server {
	t.Helper()
	html, err := os.ReadFile(filepath.Join("testdata", htmlFixture))
	if err != nil {
		t.Fatalf("read html fixture: %v", err)
	}
	cdxRows := [][]string{
		{"urlkey", "timestamp", "original", "mimetype", "statuscode", "digest", "length"},
		{"com,wimp)/x", cdxTimestamp, cdxOriginal, "text/html", "200", "X", "1234"},
	}
	cdxBody, err := json.Marshal(cdxRows)
	if err != nil {
		t.Fatalf("marshal cdx: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/cdx/search/cdx", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cdxBody)
	})
	mux.HandleFunc("/web/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestAdapter_FromArchivedURL_YouTubePage(t *testing.T) {
	srv := fakeWayback(t, "20190109001127",
		"https://www.wimp.com/a-blooper-reel-of-beatles-recordings/",
		"beatles_bloopers_2019_youtube.html")

	a := wimp.NewAdapter(wimp.Config{BaseURL: srv.URL, HTTPClient: srv.Client()})
	v, err := a.FromArchivedURL(context.Background(),
		"https://www.wimp.com/a-blooper-reel-of-beatles-recordings/")
	if err != nil {
		t.Fatalf("FromArchivedURL: %v", err)
	}

	if v.Provider != "youtube" || v.ProviderVideoID != "NZd3R2iw4cA" {
		t.Errorf("provider keys: %+v", v)
	}
	if v.WatchURL != "https://www.youtube.com/watch?v=NZd3R2iw4cA" {
		t.Errorf("watch_url: %q", v.WatchURL)
	}
	if v.EmbedURL != "https://www.youtube.com/embed/NZd3R2iw4cA" {
		t.Errorf("embed_url: %q", v.EmbedURL)
	}
	if !strings.Contains(strings.ToLower(v.Title), "beatles") {
		t.Errorf("title: %q", v.Title)
	}
	if v.ThumbnailURL == "" {
		t.Errorf("thumbnail should be populated from og:image")
	}
	if !strings.HasPrefix(v.SourceURL, "http") || !strings.Contains(v.SourceURL, "web.archive.org/web/20190109001127") {
		t.Errorf("source_url should be the Wayback permalink, got %q", v.SourceURL)
	}
	if v.SourceDesc == "" {
		t.Errorf("source_description should carry the archived og:description")
	}
	if !containsLabel(v.Labels, "wimp") {
		t.Errorf("expected 'wimp' label, got %+v", v.Labels)
	}
	if !containsLabel(v.Labels, "2019") {
		t.Errorf("expected '2019' capture-year label, got %+v", v.Labels)
	}
}

func TestAdapter_FromArchivedURL_FlashEra_ReturnsErrNoLiveEmbed(t *testing.T) {
	srv := fakeWayback(t, "20140109114206",
		"http://www.wimp.com/flyingbike/",
		"flyingbike_2014.html")

	a := wimp.NewAdapter(wimp.Config{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := a.FromArchivedURL(context.Background(), "http://www.wimp.com/flyingbike/")
	if err != wimp.ErrNoLiveEmbed {
		t.Fatalf("expected ErrNoLiveEmbed, got %v", err)
	}
}

func TestAdapter_FromArchivedURL_NoCapture(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cdx/search/cdx", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("[]"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	a := wimp.NewAdapter(wimp.Config{BaseURL: srv.URL, HTTPClient: srv.Client()})
	_, err := a.FromArchivedURL(context.Background(), "https://www.wimp.com/never-archived/")
	if err != wimp.ErrNoCapture {
		t.Fatalf("expected ErrNoCapture, got %v", err)
	}
}

func containsLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}
