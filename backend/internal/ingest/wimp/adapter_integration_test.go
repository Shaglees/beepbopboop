//go:build integration

package wimp_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
)

// TestAdapter_FromArchivedURL_RealWayback hits the real Wayback CDX + fetch
// endpoints against a known wimp.com page that embeds a YouTube iframe.
// Run with: go test -tags=integration ./internal/ingest/wimp/...
func TestAdapter_FromArchivedURL_RealWayback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	a := wimp.NewAdapter(wimp.Config{})
	const wimpURL = "https://www.wimp.com/a-blooper-reel-of-beatles-recordings/"
	v, err := a.FromArchivedURL(ctx, wimpURL)
	if err != nil {
		t.Fatalf("FromArchivedURL(real wayback): %v", err)
	}
	if v.Provider != "youtube" {
		t.Errorf("expected youtube provider, got %q", v.Provider)
	}
	if v.ProviderVideoID == "" {
		t.Errorf("expected a YouTube video id")
	}
	if v.Title == "" || !strings.Contains(strings.ToLower(v.Title), "beatles") {
		t.Errorf("expected title to mention beatles, got %q", v.Title)
	}
	if !strings.HasPrefix(v.SourceURL, "https://web.archive.org/web/") {
		t.Errorf("source_url should be Wayback permalink, got %q", v.SourceURL)
	}
	if !containsLabel(v.Labels, "wimp") {
		t.Errorf("missing 'wimp' label: %+v", v.Labels)
	}
}
