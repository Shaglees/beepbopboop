package wimp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	"github.com/shanegleeson/beepbopboop/backend/internal/video"
)

const ingestSourceName = "wimp-cdx"

type ArchiveLister interface {
	ListPageURLs(ctx context.Context, offset, limit int) ([]string, error)
}

type Inspector interface {
	InspectArchivedURL(ctx context.Context, rawURL string) (Inspection, error)
}

type BackfillOptions struct {
	CrawlBudget int
	PageSize    int
	MaxRetries  int
}

type BackfillStats struct {
	Processed          int
	PagesStored        int
	CandidatesUpserted int
	DeadLetters        int
	Retries            int
}

type Backfiller struct {
	lister    ArchiveLister
	inspector Inspector
	repo      *repository.VideoRepo
}

// RetryableError marks transient failures that should be retried a bounded
// number of times before being recorded as dead letters.
type RetryableError struct {
	Err error
}

func (e RetryableError) Error() string { return e.Err.Error() }
func (e RetryableError) Unwrap() error { return e.Err }

func NewBackfiller(lister ArchiveLister, inspector Inspector, repo *repository.VideoRepo) *Backfiller {
	return &Backfiller{lister: lister, inspector: inspector, repo: repo}
}

func (b *Backfiller) Run(ctx context.Context, opts BackfillOptions) (BackfillStats, error) {
	if opts.CrawlBudget <= 0 {
		opts.CrawlBudget = 100
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 25
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 2
	}

	offset := 0
	if ingest, err := b.repo.GetIngest(ingestSourceName); err == nil && ingest != nil && ingest.LastCursor != "" {
		if parsed, err := strconv.Atoi(ingest.LastCursor); err == nil {
			offset = parsed
		}
	} else if err != nil {
		return BackfillStats{}, err
	}

	stats := BackfillStats{}
	for stats.Processed < opts.CrawlBudget {
		pageURLs, err := b.lister.ListPageURLs(ctx, offset, opts.PageSize)
		if err != nil {
			return stats, err
		}
		if len(pageURLs) == 0 {
			break
		}
		for _, rawURL := range pageURLs {
			if stats.Processed >= opts.CrawlBudget {
				break
			}
			stats.Processed++
			normURL, err := NormalizeWimpURL(rawURL)
			if err != nil {
				stats.DeadLetters++
				_ = b.repo.UpsertSourcePage(model.VideoSourcePage{
					SourceName: "wimp",
					SourceURL:  rawURL,
					LastError:  fmt.Sprintf("normalize url: %v", err),
				})
				continue
			}

			inspection, err := b.inspectWithRetry(ctx, normURL, opts.MaxRetries, &stats)
			if err != nil {
				stats.DeadLetters++
				_ = b.repo.UpsertSourcePage(model.VideoSourcePage{
					SourceName: "wimp",
					SourceURL:  normURL,
					LastError:  err.Error(),
				})
				continue
			}

			payload, payloadErr := json.Marshal(map[string]any{
				"capture_timestamp": inspection.Capture.Timestamp,
				"archive_url":       inspection.Capture.IDURL(),
				"title":             inspection.Metadata.Title,
				"description":       inspection.Metadata.Description,
				"thumbnail_url":     inspection.Metadata.ThumbnailURL,
				"canonical_url":     inspection.Metadata.CanonicalURL,
				"embed":             inspection.Embed,
			})
			if payloadErr != nil {
				return stats, fmt.Errorf("marshal source payload: %w", payloadErr)
			}

			page := model.VideoSourcePage{
				SourceName: "wimp",
				SourceURL:  normURL,
				ArchiveURL: inspection.Capture.IDURL(),
				RawPayload: payload,
			}

			videoCandidate, videoErr := b.videoFromInspection(inspection)
			if videoErr != nil {
				page.LastError = videoErr.Error()
			} else {
				if _, err := b.repo.UpsertCatalog(videoCandidate); err != nil {
					return stats, fmt.Errorf("upsert catalog for %s: %w", normURL, err)
				}
				stats.CandidatesUpserted++
			}

			if err := b.repo.UpsertSourcePage(page); err != nil {
				return stats, err
			}
			stats.PagesStored++
		}

		offset += len(pageURLs)
		if err := b.repo.RecordIngest(ingestSourceName, strconv.Itoa(offset)); err != nil {
			return stats, err
		}
	}
	return stats, nil
}

func (b *Backfiller) inspectWithRetry(ctx context.Context, sourceURL string, maxRetries int, stats *BackfillStats) (Inspection, error) {
	var lastErr error
	attempts := maxRetries + 1
	for i := 0; i < attempts; i++ {
		inspection, err := b.inspector.InspectArchivedURL(ctx, sourceURL)
		if err == nil {
			return inspection, nil
		}
		lastErr = err
		var retryable RetryableError
		if !errors.As(err, &retryable) || i == attempts-1 {
			break
		}
		stats.Retries++
	}
	return Inspection{}, lastErr
}

func (b *Backfiller) videoFromInspection(inspection Inspection) (model.Video, error) {
	if inspection.Embed == nil {
		return model.Video{}, ErrNoLiveEmbed
	}

	v := model.Video{
		Provider:        inspection.Embed.Provider,
		ProviderVideoID: inspection.Embed.VideoID,
		WatchURL:        inspection.Embed.WatchURL,
		EmbedURL:        inspection.Embed.EmbedURL,
		Title:           inspection.Metadata.Title,
		Description:     inspection.Metadata.Description,
		ThumbnailURL:    inspection.Metadata.ThumbnailURL,
		SourceURL:       inspection.Capture.IDURL(),
		SourceDesc:      inspection.Metadata.Description,
		Labels:          buildLabels(inspection.Capture, inspection.Metadata),
		EmbedHealth:     model.EmbedHealthUnknown,
		SupportsPrevCap: video.PolicyForProvider(inspection.Embed.Provider).SupportsPreviewCap,
	}
	if t := inspection.Capture.CaptureTime(); !t.IsZero() {
		tt := t
		v.PublishedAt = &tt
	}

	enrichment := video.EnrichMetadata(v)
	if enrichment.SourceDescription != "" {
		v.SourceDesc = enrichment.SourceDescription
		v.Description = enrichment.SourceDescription
	}
	if len(enrichment.Labels) > 0 {
		v.Labels = append([]string(nil), enrichment.Labels...)
	}
	if title := video.GenerateTitle(v, enrichment); title != "" {
		v.Title = title
	}
	return v, nil
}

func NormalizeWimpURL(rawURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", fmt.Errorf("missing host")
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	if host != "wimp.com" {
		return "", fmt.Errorf("unexpected host %q", u.Host)
	}
	u.Scheme = "https"
	u.Host = "www.wimp.com"
	u.RawQuery = ""
	u.Fragment = ""
	if u.Path == "" || u.Path == "/" {
		u.Path = "/"
		return u.String(), nil
	}
	clean := path.Clean(u.Path)
	if clean == "." {
		clean = "/"
	}
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	if !strings.HasSuffix(clean, "/") {
		clean += "/"
	}
	u.Path = clean
	return u.String(), nil
}
