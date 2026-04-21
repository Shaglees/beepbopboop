package wimp_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type fakeLister struct {
	pages      [][]string
	scanned    []int
	call       int
	lastOffset int
}

func (f *fakeLister) ListPageURLs(ctx context.Context, offset, limit int) (wimp.ListPageResult, error) {
	f.lastOffset = offset
	if f.call >= len(f.pages) {
		return wimp.ListPageResult{}, nil
	}
	page := f.pages[f.call]
	scanned := len(page)
	if f.call < len(f.scanned) && f.scanned[f.call] > 0 {
		scanned = f.scanned[f.call]
	}
	f.call++
	return wimp.ListPageResult{
		URLs:        page,
		ScannedRows: scanned,
	}, nil
}

type fakeInspector struct {
	inspections map[string]wimp.Inspection
	errs        map[string][]error
	attempts    map[string]int
}

func (f *fakeInspector) InspectArchivedURL(ctx context.Context, rawURL string) (wimp.Inspection, error) {
	if f.attempts == nil {
		f.attempts = map[string]int{}
	}
	f.attempts[rawURL]++
	if len(f.errs[rawURL]) > 0 {
		err := f.errs[rawURL][0]
		f.errs[rawURL] = f.errs[rawURL][1:]
		return wimp.Inspection{}, err
	}
	inspection, ok := f.inspections[rawURL]
	if !ok {
		return wimp.Inspection{}, fmt.Errorf("missing fake inspection for %s", rawURL)
	}
	return inspection, nil
}

func TestNormalizeWimpURL(t *testing.T) {
	cases := map[string]string{
		"http://wimp.com/flyingbike":          "https://www.wimp.com/flyingbike/",
		"https://wimp.com/flyingbike/":        "https://www.wimp.com/flyingbike/",
		"http://www.wimp.com/FlyingBike///":   "https://www.wimp.com/FlyingBike/",
		"https://www.wimp.com":                "https://www.wimp.com/",
		"https://www.wimp.com/search/?q=test": "https://www.wimp.com/search/",
	}
	for raw, want := range cases {
		got, err := wimp.NormalizeWimpURL(raw)
		if err != nil {
			t.Fatalf("NormalizeWimpURL(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("NormalizeWimpURL(%q): got %q want %q", raw, got, want)
		}
	}
}

func TestBackfiller_Run_TraversesPagesAndStoresNormalizedCandidates(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	backfiller := wimp.NewBackfiller(
		&fakeLister{pages: [][]string{
			{"http://wimp.com/flyingbike", "https://www.wimp.com/beatles/"},
			{"https://wimp.com/puppykitten/"},
		}},
		&fakeInspector{inspections: map[string]wimp.Inspection{
			"https://www.wimp.com/flyingbike/":  inspectionFixture("https://www.wimp.com/flyingbike/", "20140109114206", "Flying bike completes its first test flight.", "An early hoverbike prototype completes a test flight.", nil),
			"https://www.wimp.com/beatles/":     inspectionFixture("https://www.wimp.com/beatles/", "20190109001127", "A blooper reel of Beatles recordings", "A collection of studio chatter and rough takes from Beatles recording sessions.", embedFixture("youtube", "NZd3R2iw4cA")),
			"https://www.wimp.com/puppykitten/": inspectionFixture("https://www.wimp.com/puppykitten/", "20190109001127", "Amazing video clip of a puppy meeting a kitten", "An adorable canine and feline meet for the first time.", embedFixture("youtube", "jNQXAC9IVRw")),
		}},
		repo,
	)

	stats, err := backfiller.Run(context.Background(), wimp.BackfillOptions{CrawlBudget: 10, PageSize: 2, MaxRetries: 2})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.PagesStored != 3 {
		t.Fatalf("expected 3 raw crawl records, got %+v", stats)
	}
	if stats.CandidatesUpserted != 2 {
		t.Fatalf("expected 2 normalized candidates (flash page skipped), got %+v", stats)
	}

	beatles, err := repo.GetByProviderID("youtube", "NZd3R2iw4cA")
	if err != nil || beatles == nil {
		t.Fatalf("expected beatles candidate in catalog: %v %v", err, beatles)
	}
	if beatles.Title != "Beatles studio bloopers you probably haven't heard" {
		t.Fatalf("expected generated title, got %q", beatles.Title)
	}
	if strings.Join(beatles.Labels, ",") != "music,behind-the-scenes,nostalgia" {
		t.Fatalf("expected enriched labels, got %#v", beatles.Labels)
	}

	page, err := repo.GetSourcePage("https://www.wimp.com/flyingbike/")
	if err != nil || page == nil {
		t.Fatalf("expected raw crawl record for flyingbike: %v %v", err, page)
	}
	if page.LastError == "" || !strings.Contains(page.LastError, wimp.ErrNoLiveEmbed.Error()) {
		t.Fatalf("expected dead-letter style error for non-embeddable page, got %q", page.LastError)
	}
}

func TestBackfiller_Run_RerunsWithoutDuplicateRows(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	lister := &fakeLister{pages: [][]string{{"https://www.wimp.com/beatles/"}}}
	inspector := &fakeInspector{inspections: map[string]wimp.Inspection{
		"https://www.wimp.com/beatles/": inspectionFixture("https://www.wimp.com/beatles/", "20190109001127", "A blooper reel of Beatles recordings", "A collection of studio chatter and rough takes from Beatles recording sessions.", embedFixture("youtube", "NZd3R2iw4cA")),
	}}
	backfiller := wimp.NewBackfiller(lister, inspector, repo)

	for i := 0; i < 2; i++ {
		if _, err := backfiller.Run(context.Background(), wimp.BackfillOptions{CrawlBudget: 5, PageSize: 1, MaxRetries: 2}); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	assertCount(t, db, "video_catalog", 1)
	assertCount(t, db, "video_source_pages", 1)
}

func TestBackfiller_Run_RetriesTransientErrorsAndDeadLettersPermanentFailures(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	inspector := &fakeInspector{
		inspections: map[string]wimp.Inspection{
			"https://www.wimp.com/retry-success/": inspectionFixture("https://www.wimp.com/retry-success/", "20190109001127", "A blooper reel of Beatles recordings", "A collection of studio chatter and rough takes from Beatles recording sessions.", embedFixture("youtube", "retry-video")),
		},
		errs: map[string][]error{
			"https://www.wimp.com/retry-success/":  {wimp.RetryableError{Err: fmt.Errorf("503 once")}, wimp.RetryableError{Err: fmt.Errorf("503 twice")}},
			"https://www.wimp.com/permanent-fail/": {fmt.Errorf("permanent parse failure")},
		},
	}
	backfiller := wimp.NewBackfiller(
		&fakeLister{pages: [][]string{{"https://www.wimp.com/retry-success/", "https://www.wimp.com/permanent-fail/"}}},
		inspector,
		repo,
	)

	stats, err := backfiller.Run(context.Background(), wimp.BackfillOptions{CrawlBudget: 5, PageSize: 2, MaxRetries: 3})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if inspector.attempts["https://www.wimp.com/retry-success/"] != 3 {
		t.Fatalf("expected 3 attempts for retry-success, got %d", inspector.attempts["https://www.wimp.com/retry-success/"])
	}
	if stats.Retries < 2 || stats.DeadLetters != 1 {
		t.Fatalf("unexpected retry/dead-letter stats: %+v", stats)
	}
	page, err := repo.GetSourcePage("https://www.wimp.com/permanent-fail/")
	if err != nil || page == nil {
		t.Fatalf("expected dead-letter source page: %v %v", err, page)
	}
	if !strings.Contains(page.LastError, "permanent parse failure") {
		t.Fatalf("expected permanent failure message, got %q", page.LastError)
	}
}

func TestBackfiller_Run_AdvancesCursorByScannedRowsNotDedupedURLs(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	lister := &fakeLister{
		pages:   [][]string{{"https://www.wimp.com/beatles/"}},
		scanned: []int{10}, // Simulates 10 raw CDX rows collapsing to 1 URL.
	}
	backfiller := wimp.NewBackfiller(
		lister,
		&fakeInspector{inspections: map[string]wimp.Inspection{
			"https://www.wimp.com/beatles/": inspectionFixture("https://www.wimp.com/beatles/", "20190109001127", "A blooper reel of Beatles recordings", "A collection of studio chatter and rough takes from Beatles recording sessions.", embedFixture("youtube", "NZd3R2iw4cA")),
		}},
		repo,
	)

	if _, err := backfiller.Run(context.Background(), wimp.BackfillOptions{CrawlBudget: 5, PageSize: 5, MaxRetries: 2}); err != nil {
		t.Fatalf("first run: %v", err)
	}
	ingest, err := repo.GetIngest("wimp-cdx")
	if err != nil || ingest == nil {
		t.Fatalf("GetIngest after first run: %v %v", err, ingest)
	}
	if ingest.LastCursor != "10" {
		t.Fatalf("expected cursor=10 from scanned rows, got %q", ingest.LastCursor)
	}

	// Next run should resume from offset 10.
	lister.pages = [][]string{}
	lister.scanned = nil
	lister.call = 0
	if _, err := backfiller.Run(context.Background(), wimp.BackfillOptions{CrawlBudget: 5, PageSize: 5, MaxRetries: 2}); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if lister.lastOffset != 10 {
		t.Fatalf("expected resumed offset 10, got %d", lister.lastOffset)
	}
}

func inspectionFixture(sourceURL, timestamp, title, sourceDesc string, embed *wimp.Embed) wimp.Inspection {
	return wimp.Inspection{
		Capture: wimp.Capture{
			Timestamp: timestamp,
			Original:  sourceURL,
		},
		Metadata: wimp.Metadata{
			Title:        title,
			Description:  sourceDesc,
			ThumbnailURL: "https://example.com/thumb.jpg",
			CanonicalURL: sourceURL,
		},
		Embed: embed,
	}
}

func embedFixture(provider, id string) *wimp.Embed {
	switch provider {
	case "youtube":
		return &wimp.Embed{
			Provider: provider,
			VideoID:  id,
			WatchURL: "https://www.youtube.com/watch?v=" + id,
			EmbedURL: "https://www.youtube.com/embed/" + id,
		}
	default:
		return &wimp.Embed{
			Provider: provider,
			VideoID:  id,
			WatchURL: "https://vimeo.com/" + id,
			EmbedURL: "https://player.vimeo.com/video/" + id,
		}
	}
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count: got %d want %d", table, got, want)
	}
}
