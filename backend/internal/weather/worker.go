package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// Worker periodically fetches weather for active user locations and upserts
// weather posts so they appear in nearby users' feeds.
type Worker struct {
	svc          *Service
	postRepo     *repository.PostRepo
	settingsRepo *repository.UserSettingsRepo
	interval     time.Duration
}

func NewWorker(svc *Service, postRepo *repository.PostRepo, settingsRepo *repository.UserSettingsRepo, interval time.Duration) *Worker {
	return &Worker{
		svc:          svc,
		postRepo:     postRepo,
		settingsRepo: settingsRepo,
		interval:     interval,
	}
}

// Run starts the weather worker loop. It runs an immediate first cycle, then
// repeats on the configured interval until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("weather worker started", "interval", w.interval)

	// Run immediately on startup.
	w.cycle()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("weather worker stopped")
			return
		case <-ticker.C:
			w.cycle()
		}
	}
}

func (w *Worker) cycle() {
	cells, err := w.settingsRepo.DistinctLocationCells(gridSize)
	if err != nil {
		slog.Error("weather worker: failed to get location cells", "error", err)
		return
	}

	if len(cells) == 0 {
		slog.Debug("weather worker: no user locations, skipping")
		return
	}

	var ok, failed int
	for _, cell := range cells {
		if err := w.processCell(cell); err != nil {
			slog.Warn("weather worker: cell failed", "lat", cell.Lat, "lon", cell.Lon, "error", err)
			failed++
		} else {
			ok++
		}
	}

	slog.Info("weather worker: cycle complete", "cells", len(cells), "ok", ok, "failed", failed)
}

func (w *Worker) processCell(cell repository.GridCell) error {
	resp, err := w.svc.Fetch(cell.Lat, cell.Lon)
	if err != nil {
		return fmt.Errorf("fetch weather: %w", err)
	}

	key := gridKey(cell.Lat, cell.Lon)
	title := fmt.Sprintf("%d° %s", int(resp.Current.Temp+0.5), resp.Current.Condition)

	body := buildWeatherBody(resp)

	// Pack the full forecast as a JSON string in external_url for iOS to parse.
	forecastData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal forecast: %w", err)
	}

	return w.postRepo.UpsertWeatherPost(key, title, body, cell.Lat, cell.Lon, string(forecastData))
}

// buildWeatherBody creates a human-readable weather summary.
func buildWeatherBody(r *Response) string {
	s := fmt.Sprintf("Currently %d° (%s). Feels like %d°. Wind %d km/h, humidity %d%%.",
		int(r.Current.Temp+0.5),
		r.Current.Condition,
		int(r.Current.FeelsLike+0.5),
		int(r.Current.WindSpeed+0.5),
		r.Current.Humidity,
	)

	if r.Current.UVIndex >= 3 {
		s += fmt.Sprintf(" UV index %d.", int(r.Current.UVIndex+0.5))
	}

	// Next few hours summary.
	if len(r.Hourly) >= 3 {
		s += fmt.Sprintf("\n\nNext hours: %d° → %d° → %d°.",
			int(r.Hourly[0].Temp+0.5),
			int(r.Hourly[1].Temp+0.5),
			int(r.Hourly[2].Temp+0.5),
		)
		for _, h := range r.Hourly[:3] {
			if h.PrecipProb > 30 {
				s += fmt.Sprintf(" %d%% chance of precipitation.", h.PrecipProb)
				break
			}
		}
	}

	// Tomorrow preview.
	if len(r.Daily) >= 2 {
		d := r.Daily[1]
		s += fmt.Sprintf("\n\nTomorrow: %s, %d°/%d°.",
			d.Condition,
			int(d.High+0.5),
			int(d.Low+0.5),
		)
	}

	return s
}
