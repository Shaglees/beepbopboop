package creators

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

const (
	minCreatorsThreshold = 3  // expand radius if fewer than this many creators found
	stalenessDays        = 60 // re-research a region after this many days
)

// Worker periodically discovers local creators for active user locations
// and upserts creator_spotlight posts so they appear in nearby users' feeds.
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

// Run starts the worker loop. It runs an immediate first cycle, then repeats
// on the configured interval until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("creators worker started", "interval", w.interval)

	w.cycle()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("creators worker stopped")
			return
		case <-ticker.C:
			w.cycle()
		}
	}
}

func (w *Worker) cycle() {
	cells, err := w.settingsRepo.DistinctLocationCells(0.5) // 0.5° ≈ 55 km grid
	if err != nil {
		slog.Error("creators worker: failed to get location cells", "error", err)
		return
	}

	if len(cells) == 0 {
		slog.Debug("creators worker: no user locations, skipping")
		return
	}

	var ok, failed int
	for _, cell := range cells {
		if err := w.processCell(cell); err != nil {
			slog.Warn("creators worker: cell failed", "lat", cell.Lat, "lon", cell.Lon, "error", err)
			failed++
		} else {
			ok++
		}
	}

	slog.Info("creators worker: cycle complete", "cells", len(cells), "ok", ok, "failed", failed)
}

func (w *Worker) processCell(cell repository.GridCell) error {
	// Check if we already have fresh posts for this region.
	gridKey := fmt.Sprintf("creators-%.1f,%.1f", cell.Lat, cell.Lon)
	if w.postRepo.HasFreshCreatorPosts(gridKey, stalenessDays) {
		slog.Debug("creators worker: region is fresh, skipping", "lat", cell.Lat, "lon", cell.Lon)
		return nil
	}

	// Determine initial radius using density heuristic.
	radius := AdaptiveRadiusKm(cell.Lat, cell.Lon)

	var creators []LocalCreator
	var areaName string
	var err error

	// Adaptive radius: expand until we hit the threshold or max radius.
	for {
		creators, areaName, err = w.svc.ResearchCreators(cell.Lat, cell.Lon, radius)
		if err != nil {
			return fmt.Errorf("research creators at radius %.0f km: %w", radius, err)
		}
		if len(creators) >= minCreatorsThreshold {
			break
		}
		next, ok := ExpandRadius(radius)
		if !ok {
			break // already at maximum
		}
		slog.Debug("creators worker: sparse results, expanding radius",
			"lat", cell.Lat, "lon", cell.Lon,
			"current_radius", radius, "next_radius", next,
			"found", len(creators),
		)
		radius = next
	}

	if len(creators) == 0 {
		slog.Debug("creators worker: no creators found", "lat", cell.Lat, "lon", cell.Lon, "area", areaName)
		return nil
	}

	// Upsert one post per creator.
	var upsertOk, upsertFailed int
	for _, c := range creators {
		if err := w.upsertCreatorPost(c, gridKey, areaName, radius); err != nil {
			slog.Warn("creators worker: upsert failed", "creator", c.Name, "error", err)
			upsertFailed++
		} else {
			upsertOk++
		}
	}

	slog.Info("creators worker: cell complete",
		"lat", cell.Lat, "lon", cell.Lon,
		"area", areaName, "radius_km", radius,
		"creators", len(creators), "ok", upsertOk, "failed", upsertFailed,
	)
	return nil
}

func (w *Worker) upsertCreatorPost(c LocalCreator, gridKey, areaName string, radiusKm float64) error {
	title := fmt.Sprintf("%s · %s", c.Name, titleCase(c.Designation))

	body := buildCreatorBody(c)

	profileJSON, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal creator: %w", err)
	}

	return w.postRepo.UpsertCreatorPost(c.ID, gridKey, title, body, c.Lat, c.Lon, areaName, c.Designation, string(profileJSON))
}

// buildCreatorBody assembles the post body text for a creator.
func buildCreatorBody(c LocalCreator) string {
	s := c.Bio
	if c.NotableWorks != "" {
		s += "\n\nNotable works: " + c.NotableWorks
	}
	if len(c.Tags) > 0 {
		s += "\n\nTags: " + joinTags(c.Tags)
	}
	if c.Source != "" {
		s += "\n\nDiscovered via: " + c.Source
	}
	return s
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	words := []rune(s)
	if len(words) > 0 {
		words[0] = []rune(fmt.Sprintf("%c", []rune(s)[0]-32))[0]
		if s[0] >= 'a' && s[0] <= 'z' {
			return string(rune(s[0]-32)) + s[1:]
		}
	}
	return s
}

func joinTags(tags []string) string {
	result := ""
	for i, t := range tags {
		if i > 0 {
			result += ", "
		}
		result += t
	}
	return result
}
