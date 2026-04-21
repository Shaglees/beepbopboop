// Command wimpingest runs a single live ingest pass against wimp.com.
//
// It is the one-shot CLI counterpart to the orchestrator: pull the current
// RSS feed, visit each post page, parse + oEmbed-enrich, and upsert rows into
// video_catalog. Use this to populate the catalog today; a scheduled worker
// can be added later by importing the same orchestrator.
//
// Required environment:
//
//	DATABASE_URL   Postgres DSN (same one the server uses).
//
// Flags:
//
//	--limit=N      Cap how many RSS items to ingest. 0 = all (feed has ~10).
//	--feed=URL     Override the feed URL (default https://www.wimp.com/feed/).
//	--no-oembed    Skip oEmbed enrichment (fast path, blank channel titles).
//
// Exit codes: 0 on success, 1 on fatal error (DB open, RSS fetch, etc).
//
// Partial failures (individual post pages failing to parse) are NOT fatal —
// they're counted in the report and logged to stderr.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func main() {
	var (
		limit    int
		feedURL  string
		noEnrich bool
		dbURL    string
	)
	flag.IntVar(&limit, "limit", 0, "Max RSS items to ingest (0 = all)")
	flag.StringVar(&feedURL, "feed", "", "Override feed URL")
	flag.BoolVar(&noEnrich, "no-oembed", false, "Skip oEmbed enrichment")
	flag.StringVar(&dbURL, "db", os.Getenv("DATABASE_URL"), "Postgres DSN (or DATABASE_URL env)")
	flag.Parse()

	if dbURL == "" {
		fatalf("DATABASE_URL not set (or pass --db)")
	}

	// Generous overall timeout: 5s RSS + (10 items * ~2s fetch + 1s oEmbed)
	// is well under 30s in practice, but leave headroom for slow upstreams.
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	db, err := database.Open(dbURL)
	if err != nil {
		fatalf("open database: %v", err)
	}
	defer db.Close()

	videoRepo := repository.NewVideoRepo(db)
	adapter := wimp.NewAdapter(wimp.Config{})
	lister := wimp.NewRSSLister(feedURL, nil)

	var enricher *wimp.Enricher
	if !noEnrich {
		enricher = wimp.NewEnricher(wimp.EnricherConfig{Timeout: 10 * time.Second})
	}

	orch := &wimp.Orchestrator{
		Lister:   lister,
		Adapter:  adapter,
		Enricher: enricher,
		Repo:     videoRepo,
	}

	report, err := orch.Run(ctx, limit)
	if err != nil {
		fatalf("ingest run: %v", err)
	}

	summary := map[string]any{
		"seen":           report.Seen,
		"ingested":       report.Ingested,
		"already_cached": report.AlreadyCached,
		"no_embed":       report.NoEmbed,
		"errored":        report.Errored,
		"videos":         report.Videos,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		fatalf("encode summary: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "wimpingest: "+format+"\n", args...)
	os.Exit(1)
}
