package wimp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
)

func TestCDXClient_LatestCapture_PicksNewestOKHTML(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "cdx_beatles_bloopers.json"))
	if err != nil {
		t.Fatalf("read cdx fixture: %v", err)
	}

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	client := wimp.NewCDXClient(srv.URL, srv.Client())

	cap, err := client.LatestCapture(context.Background(),
		"https://www.wimp.com/a-blooper-reel-of-beatles-recordings/")
	if err != nil {
		t.Fatalf("LatestCapture: %v", err)
	}
	if cap.Timestamp != "20250321105647" {
		t.Errorf("timestamp: got %q want newest 20250321105647", cap.Timestamp)
	}
	if cap.Original != "https://www.wimp.com/a-blooper-reel-of-beatles-recordings/" {
		t.Errorf("original url: got %q", cap.Original)
	}
	if want := "https://web.archive.org/web/20250321105647id_/https://www.wimp.com/a-blooper-reel-of-beatles-recordings/"; cap.IDURL() != want {
		t.Errorf("id-form url: got %q want %q", cap.IDURL(), want)
	}
	if gotQuery == "" {
		t.Errorf("expected client to send a query string to CDX API")
	}
}

func TestCDXClient_LatestCapture_NoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(srv.Close)

	client := wimp.NewCDXClient(srv.URL, srv.Client())
	_, err := client.LatestCapture(context.Background(), "https://www.wimp.com/nope/")
	if err == nil {
		t.Fatalf("expected ErrNoCapture, got nil")
	}
	if err != wimp.ErrNoCapture {
		t.Errorf("expected ErrNoCapture, got %v", err)
	}
}
