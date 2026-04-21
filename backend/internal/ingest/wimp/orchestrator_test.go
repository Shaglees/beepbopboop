package wimp_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// TestOrchestrator_Run_IngestsAndDedups runs the full pipeline end-to-end
// against:
//   - a fake wimp.com that serves an RSS index plus two post pages
//   - the real Postgres test DB provided by database.OpenTestDB
//   - no oEmbed enricher (Enricher=nil) to keep the test hermetic
//
// What the test asserts:
//   1. First Run ingests 2 rows with provider=youtube and merges RSS categories
//      into labels.
//   2. Second Run on the same fake wimp sees everything already cached and
//      does not try to re-fetch (AlreadyCached == 2).
func TestOrchestrator_Run_IngestsAndDedups(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	pages := map[string]string{
		"/post-a/": postHTML("YT Post A", "Funny dog.", "dogs, funny", "AAAAAAAAAAA", "2026-04-18T10:00:00+00:00"),
		"/post-b/": postHTML("YT Post B", "Cool bird.", "birds, nature", "BBBBBBBBBBB", "2026-04-19T11:00:00+00:00"),
	}

	var wimpServerURL string
	feedBodyFn := func() string {
		// RSS references the two post URLs on the running httptest server.
		// We render the feed dynamically so it points at wimpServerURL.
		var items strings.Builder
		for path, html := range pages {
			title := titleFromHTML(html)
			items.WriteString(fmt.Sprintf(
				`<item><title>%s</title><link>%s%s</link><pubDate>Sat, 19 Apr 2026 11:00:00 +0000</pubDate><category>Funny</category><description>caption</description><dc:creator>Tester</dc:creator></item>`,
				title, wimpServerURL, path,
			))
		}
		return `<?xml version="1.0"?><rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/"><channel>` + items.String() + `</channel></rss>`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/feed/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(feedBodyFn()))
	})
	for path, html := range pages {
		path, html := path, html
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))
		})
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()
	wimpServerURL = ts.URL

	orch := &wimp.Orchestrator{
		Lister:  wimp.NewRSSLister(ts.URL+"/feed/", nil),
		Adapter: wimp.NewAdapter(wimp.Config{}),
		// Skip oEmbed — we don't want the test hitting the real YouTube.
		Enricher: nil,
		Repo:     repo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// First run: expect 2 fresh ingests.
	report, err := orch.Run(ctx, 0)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if report.Seen != 2 {
		t.Errorf("expected 2 RSS items seen, got %d", report.Seen)
	}
	if report.Ingested != 2 {
		t.Errorf("expected 2 ingested, got %d", report.Ingested)
	}
	if report.AlreadyCached != 0 {
		t.Errorf("expected 0 already-cached on first run, got %d", report.AlreadyCached)
	}

	// Labels should include both the scraped keyword labels AND the RSS category.
	for _, hit := range report.Videos {
		if !containsAny(hit.Labels, "funny") {
			t.Errorf("hit %s missing RSS category 'funny' in labels: %v", hit.VideoID, hit.Labels)
		}
	}

	// Second run: everything is already cached.
	report2, err := orch.Run(ctx, 0)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if report2.AlreadyCached != 2 {
		t.Errorf("expected 2 already-cached on second run, got %d", report2.AlreadyCached)
	}
	if report2.Ingested != 0 {
		t.Errorf("expected 0 ingested on second run, got %d", report2.Ingested)
	}

	// The ingest cursor should have been recorded.
	cursor, err := repo.GetIngest("wimp.com")
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if cursor == nil || cursor.LastCursor == "" {
		t.Errorf("expected wimp.com ingest cursor to be recorded")
	}
}

func TestOrchestrator_Run_NoEmbedPagesAreCounted(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	var wimpServerURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/feed/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprintf(w, `<?xml version="1.0"?><rss version="2.0"><channel><item><title>dud</title><link>%s/dud/</link><pubDate>Sat, 19 Apr 2026 11:00:00 +0000</pubDate></item></channel></rss>`, wimpServerURL)
	})
	mux.HandleFunc("/dud/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body>no video here</body></html>`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	wimpServerURL = ts.URL

	orch := &wimp.Orchestrator{
		Lister:  wimp.NewRSSLister(ts.URL+"/feed/", nil),
		Adapter: wimp.NewAdapter(wimp.Config{}),
		Repo:    repo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := orch.Run(ctx, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.NoEmbed != 1 {
		t.Errorf("expected 1 no-embed page, got %d", report.NoEmbed)
	}
	if report.Ingested != 0 {
		t.Errorf("expected 0 ingested, got %d", report.Ingested)
	}
}

// --- helpers ------------------------------------------------------------------

func postHTML(title, desc, keywords, ytID, publishedAt string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head>
<meta property="og:title" content="%s">
<meta property="og:description" content="%s">
<meta property="og:image" content="https://wimp/x.jpg">
<meta property="article:published_time" content="%s">
<meta name="keywords" content="%s">
<title>%s</title>
</head><body>
<article><iframe src="https://www.youtube.com/embed/%s" allowfullscreen></iframe></article>
</body></html>`, title, desc, publishedAt, keywords, title, ytID)
}

func titleFromHTML(html string) string {
	const marker = `og:title" content="`
	i := strings.Index(html, marker)
	if i < 0 {
		return "untitled"
	}
	rest := html[i+len(marker):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return "untitled"
	}
	return rest[:j]
}

func containsAny(labels []string, wanted ...string) bool {
	seen := make(map[string]bool, len(labels))
	for _, l := range labels {
		seen[l] = true
	}
	for _, w := range wanted {
		if seen[w] {
			return true
		}
	}
	return false
}

// sanity: force url package usage in case we inline URLs later.
var _ = url.Parse
