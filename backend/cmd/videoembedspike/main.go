package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
	"github.com/shanegleeson/beepbopboop/backend/internal/video"
	"github.com/shanegleeson/beepbopboop/backend/internal/videohealth"
)

func main() {
	var (
		inputFile  = flag.String("input-file", "", "optional file of Wimp URLs (one per line)")
		sampleSize = flag.Int("sample-size", 10, "number of pages to inspect when not using --input-file")
		format     = flag.String("format", "json", "output format: json or markdown")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	urls, err := loadURLs(ctx, *inputFile, *sampleSize)
	if err != nil {
		fatal(err)
	}
	adapter := wimp.NewAdapter(wimp.Config{})
	checker := videohealth.NewHTTPChecker(nil)

	samples := make([]video.ViabilitySample, 0, len(urls))
	for _, rawURL := range urls {
		inspection, err := adapter.InspectArchivedURL(ctx, rawURL)
		if err != nil {
			samples = append(samples, video.ViabilitySample{Health: "error"})
			continue
		}
		if inspection.Embed == nil {
			samples = append(samples, video.ViabilitySample{Health: "no_live_embed"})
			continue
		}
		candidate, err := adapter.FromArchivedURL(ctx, rawURL)
		if err != nil {
			samples = append(samples, video.ViabilitySample{Health: "error"})
			continue
		}
		health, err := checker.CheckEmbed(ctx, candidate)
		if err != nil {
			health = "unknown"
		}
		samples = append(samples, video.ViabilitySample{
			Provider: candidate.Provider,
			Health:   health,
		})
	}

	report := video.BuildViabilityReport(samples)
	switch strings.ToLower(*format) {
	case "markdown":
		fmt.Print(renderMarkdown(report))
	default:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fatal(err)
		}
	}
}

func loadURLs(ctx context.Context, inputFile string, sampleSize int) ([]string, error) {
	if inputFile != "" {
		f, err := os.Open(inputFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		var urls []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			urls = append(urls, line)
		}
		return urls, scanner.Err()
	}

	lister := wimp.NewCDXArchiveLister("", nil)
	return lister.ListPageURLs(ctx, 0, sampleSize)
}

func renderMarkdown(report video.ViabilityReport) string {
	var b strings.Builder
	b.WriteString("# Video Embed Spike\n\n")
	b.WriteString(fmt.Sprintf("- Sample size: %d\n", report.SampleSize))
	b.WriteString(fmt.Sprintf("- No live embed pages: %d\n", report.NoLiveEmbedCount))
	b.WriteString("- Recommendation: " + report.Recommendation + "\n\n")
	b.WriteString("## Provider Matrix\n")
	for provider, stats := range report.Providers {
		b.WriteString(fmt.Sprintf("- `%s`: ok=%d blocked=%d gone=%d unknown=%d preview_cap=%t fallback=%s\n",
			provider, stats.OK, stats.Blocked, stats.Gone, stats.Unknown,
			stats.Policy.SupportsPreviewCap, stats.Policy.FallbackBehavior))
	}
	return b.String()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
