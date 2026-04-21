package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
	"github.com/shanegleeson/beepbopboop/backend/internal/video"
)

func main() {
	var wimpURL string
	flag.StringVar(&wimpURL, "wimp-url", "https://www.wimp.com/a-blooper-reel-of-beatles-recordings/", "Wimp URL to inspect and convert into a video_embed post payload")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	adapter := wimp.NewAdapter(wimp.Config{})
	candidate, err := adapter.FromArchivedURL(ctx, wimpURL)
	if err != nil {
		fatal(err)
	}

	enrichment := video.EnrichMetadata(candidate)
	candidate.Title = video.GenerateTitle(candidate, enrichment)
	if enrichment.SourceDescription != "" {
		candidate.Description = enrichment.SourceDescription
		candidate.SourceDesc = enrichment.SourceDescription
	}
	if len(enrichment.Labels) > 0 {
		candidate.Labels = enrichment.Labels
	}
	candidate.SupportsPrevCap = video.PolicyForProvider(candidate.Provider).SupportsPreviewCap

	payload := map[string]any{
		"title":        candidate.Title,
		"body":         candidate.Description,
		"post_type":    "video",
		"display_hint": "video_embed",
		"labels":       candidate.Labels,
		"image_url":    candidate.ThumbnailURL,
		"external_url": mustJSON(map[string]any{
			"provider":             candidate.Provider,
			"video_id":             candidate.ProviderVideoID,
			"watch_url":            candidate.WatchURL,
			"embed_url":            candidate.EmbedURL,
			"thumbnail_url":        candidate.ThumbnailURL,
			"channel_title":        candidate.ChannelTitle,
			"supports_preview_cap": candidate.SupportsPrevCap,
		}),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		fatal(err)
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		fatal(err)
	}
	return string(b)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
