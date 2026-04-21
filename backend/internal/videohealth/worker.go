package videohealth

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type Checker interface {
	CheckEmbed(ctx context.Context, v model.Video) (string, error)
}

type Worker struct {
	repo     *repository.VideoRepo
	checker  Checker
	interval time.Duration
}

type Stats struct {
	Checked  int
	OK       int
	Blocked  int
	Gone     int
	Unknown  int
	Failures int
}

func NewWorker(repo *repository.VideoRepo, checker Checker) *Worker {
	return &Worker{repo: repo, checker: checker, interval: 6 * time.Hour}
}

func NewScheduledWorker(repo *repository.VideoRepo, checker Checker, interval time.Duration) *Worker {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	return &Worker{repo: repo, checker: checker, interval: interval}
}

func (w *Worker) Run(ctx context.Context) {
	slog.Info("videohealth worker started", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Do one pass on startup so fresh deployments don't wait for the first tick.
	w.runAndLog(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("videohealth worker stopped")
			return
		case <-ticker.C:
			w.runAndLog(ctx)
		}
	}
}

func (w *Worker) runAndLog(ctx context.Context) {
	stats, err := w.RunOnce(ctx, 100, 7*24*time.Hour)
	if err != nil {
		slog.Error("videohealth worker failed", "error", err)
		return
	}
	if stats.Checked > 0 {
		slog.Info("videohealth worker completed",
			"checked", stats.Checked,
			"ok", stats.OK,
			"blocked", stats.Blocked,
			"gone", stats.Gone,
			"unknown", stats.Unknown,
			"failures", stats.Failures,
		)
	}
}

func (w *Worker) RunOnce(ctx context.Context, limit int, staleAfter time.Duration) (Stats, error) {
	candidates, err := w.repo.ListForEmbedHealthCheck(ctx, staleAfter, limit)
	if err != nil {
		return Stats{}, err
	}
	stats := Stats{}
	for _, candidate := range candidates {
		stats.Checked++
		status, err := w.checker.CheckEmbed(ctx, candidate)
		if err != nil {
			stats.Failures++
			continue
		}
		if status == "" {
			status = "unknown"
		}
		switch status {
		case "ok":
			stats.OK++
		case "blocked":
			stats.Blocked++
		case "gone":
			stats.Gone++
		default:
			status = "unknown"
			stats.Unknown++
		}
		if err := w.repo.UpdateEmbedHealth(candidate.ID, status); err != nil {
			return stats, err
		}
	}
	return stats, nil
}

type HTTPChecker struct {
	Client        *http.Client
	YouTubeOEmbed string
	VimeoOEmbed   string
}

func NewHTTPChecker(client *http.Client) *HTTPChecker {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &HTTPChecker{
		Client:        client,
		YouTubeOEmbed: "https://www.youtube.com/oembed",
		VimeoOEmbed:   "https://vimeo.com/api/oembed.json",
	}
}

func (c *HTTPChecker) CheckEmbed(ctx context.Context, v model.Video) (string, error) {
	endpoint, err := c.oembedEndpoint(v)
	if err != nil {
		return "unknown", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "unknown", err
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return "unknown", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return "ok", nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return "blocked", nil
	case http.StatusNotFound, http.StatusGone:
		return "gone", nil
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return "unknown", nil
	default:
		if resp.StatusCode >= 500 {
			return "unknown", nil
		}
		return "unknown", nil
	}
}

func (c *HTTPChecker) oembedEndpoint(v model.Video) (string, error) {
	watchURL := strings.TrimSpace(v.WatchURL)
	if watchURL == "" {
		watchURL = strings.TrimSpace(v.EmbedURL)
	}
	if watchURL == "" {
		return "", fmt.Errorf("video %s has no watch_url or embed_url", v.ID)
	}
	q := url.Values{}
	q.Set("url", watchURL)
	q.Set("format", "json")

	switch strings.ToLower(v.Provider) {
	case "youtube":
		return c.YouTubeOEmbed + "?" + q.Encode(), nil
	case "vimeo":
		return c.VimeoOEmbed + "?" + q.Encode(), nil
	default:
		return "", fmt.Errorf("unsupported provider %q", v.Provider)
	}
}
