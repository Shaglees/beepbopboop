package wimp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// Orchestrator runs a full live ingest cycle:
//
//  1. Fetch the RSS feed to enumerate candidate wimp.com permalinks.
//  2. For each permalink, skip if we already have a catalog row (cheap dedup).
//  3. Fetch the live page, parse title/thumbnail/embed.
//  4. Optionally enrich via oEmbed (YouTube/Vimeo only).
//  5. Upsert into video_catalog with the union of wimp's RSS categories and
//     the page's keyword metadata as labels.
//
// This is the function a CLI, HTTP admin endpoint, or scheduled worker should
// call to run ingest. It's deliberately synchronous and returns a structured
// report so callers can log / assert on it.
type Orchestrator struct {
	Lister   *RSSLister
	Adapter  *Adapter
	Enricher *Enricher
	Repo     *repository.VideoRepo
}

// IngestReport captures what happened during a Run. Useful for CLI output,
// tests, and observability.
type IngestReport struct {
	Seen          int         // items returned by the RSS feed
	AlreadyCached int         // skipped because (provider, video_id) was already in the catalog
	Ingested      int         // rows written or updated
	NoEmbed       int         // pages with no recognizable provider URL
	Errored       int         // live-fetch or upsert errors
	Videos        []IngestHit // concise summary of every row we touched
}

// IngestHit is one row we upserted, with enough context to surface in a CLI
// report or test assertion.
type IngestHit struct {
	WimpURL      string   `json:"wimp_url"`
	VideoID      string   `json:"video_id"`
	Provider     string   `json:"provider"`
	Title        string   `json:"title"`
	ChannelTitle string   `json:"channel_title,omitempty"`
	Labels       []string `json:"labels"`
	Enriched     bool     `json:"enriched"`
}

// Run executes a single ingest pass.
//
// The limit parameter caps how many RSS items are processed. Pass 5 to mirror
// wimp's "daily 5" cadence; pass 0 for unlimited (the feed is ~10 items).
func (o *Orchestrator) Run(ctx context.Context, limit int) (IngestReport, error) {
	if o.Lister == nil || o.Adapter == nil || o.Repo == nil {
		return IngestReport{}, fmt.Errorf("orchestrator: Lister, Adapter, Repo are required")
	}

	items, err := o.Lister.List(ctx, limit)
	if err != nil {
		return IngestReport{}, fmt.Errorf("orchestrator: list rss: %w", err)
	}

	report := IngestReport{Seen: len(items)}
	for _, item := range items {
		hit, class, err := o.ingestOne(ctx, item)
		switch class {
		case classIngested:
			report.Ingested++
			report.Videos = append(report.Videos, hit)
		case classAlreadyCached:
			report.AlreadyCached++
		case classNoEmbed:
			report.NoEmbed++
		case classErrored:
			report.Errored++
			slog.Warn("wimp ingest: item failed",
				"wimp_url", item.Link, "error", err)
		}
	}
	// Persist a lightweight cursor so operators can see the last run time.
	// last_cursor isn't used for resumption (the RSS feed is small and we
	// dedup by provider_video_id), but it's still a useful debug signal.
	if len(items) > 0 {
		_ = o.Repo.RecordIngest("wimp.com", items[0].PubDate.Format("2006-01-02T15:04:05Z"))
	}
	return report, nil
}

type ingestClass int

const (
	classErrored ingestClass = iota
	classAlreadyCached
	classNoEmbed
	classIngested
)

func (o *Orchestrator) ingestOne(ctx context.Context, item RSSItem) (IngestHit, ingestClass, error) {
	candidate, err := o.Adapter.FromLiveURL(ctx, item.Link)
	if err != nil {
		if errors.Is(err, ErrNoLiveEmbed) {
			return IngestHit{}, classNoEmbed, nil
		}
		return IngestHit{}, classErrored, err
	}

	// Cheap dedup: if we already have a row for this (provider, video_id),
	// skip the oEmbed call. The upsert would be idempotent anyway, but oEmbed
	// burns a round-trip and we run this daily.
	if existing, err := o.Repo.GetByProviderID(candidate.Provider, candidate.ProviderVideoID); err == nil && existing != nil {
		return IngestHit{}, classAlreadyCached, nil
	}

	// Merge the RSS categories in before oEmbed: RSS gives us the editorial
	// taxonomy (Dogs, Funny, Technology) which is almost always cleaner than
	// wimp's page-level keywords.
	candidate.Labels = mergeLabels(candidate.Labels, item.Categories)

	enriched := false
	if o.Enricher != nil {
		if err := o.Enricher.Enrich(ctx, &candidate); err != nil {
			slog.Debug("wimp ingest: oembed skipped",
				"wimp_url", item.Link, "provider", candidate.Provider, "reason", err.Error())
		} else {
			enriched = true
		}
	}

	saved, err := o.Repo.UpsertCatalog(candidate)
	if err != nil {
		return IngestHit{}, classErrored, fmt.Errorf("upsert catalog: %w", err)
	}

	return IngestHit{
		WimpURL:      item.Link,
		VideoID:      saved.ID,
		Provider:     saved.Provider,
		Title:        saved.Title,
		ChannelTitle: saved.ChannelTitle,
		Labels:       saved.Labels,
		Enriched:     enriched,
	}, classIngested, nil
}

// mergeLabels preserves order (RSS categories first — they're editorial), dedups
// case-insensitively, and strips empties. Caller-owned slices are not mutated.
func mergeLabels(scraped []string, rss []string) []string {
	out := make([]string, 0, len(scraped)+len(rss))
	seen := make(map[string]bool)
	add := func(label string) {
		lower := normalizeLabel(label)
		if lower == "" || seen[lower] {
			return
		}
		seen[lower] = true
		out = append(out, lower)
	}
	for _, l := range rss {
		add(l)
	}
	for _, l := range scraped {
		add(l)
	}
	return out
}

func normalizeLabel(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
