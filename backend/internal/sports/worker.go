package sports

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// Worker periodically fetches live scores and upserts one post per game.
type Worker struct {
	svc      *Service
	postRepo *repository.PostRepo
	interval time.Duration
}

func NewWorker(svc *Service, postRepo *repository.PostRepo, interval time.Duration) *Worker {
	return &Worker{svc: svc, postRepo: postRepo, interval: interval}
}

// Run starts the sports worker loop. It runs an immediate first cycle then
// repeats on the configured interval until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("sports worker started", "interval", w.interval)

	w.cycle()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("sports worker stopped")
			return
		case <-ticker.C:
			w.cycle()
		}
	}
}

func (w *Worker) cycle() {
	games, err := w.svc.FetchAll()
	if err != nil {
		slog.Error("sports worker: fetch failed", "error", err)
		return
	}

	if len(games) == 0 {
		slog.Debug("sports worker: no games, skipping")
		return
	}

	var ok, failed int
	for _, g := range games {
		if err := w.processGame(g); err != nil {
			slog.Warn("sports worker: game failed", "league", g.League, "eventID", g.EventID, "error", err)
			failed++
		} else {
			ok++
		}
	}

	slog.Info("sports worker: cycle complete", "games", len(games), "ok", ok, "failed", failed)
}

func (w *Worker) processGame(g FetchedGame) error {
	gameID := g.League + "-" + g.EventID
	title := buildPostTitle(g.Data)
	body := buildPostBody(g.Data)

	gameDataJSON, err := json.Marshal(g.Data)
	if err != nil {
		return fmt.Errorf("marshal game data: %w", err)
	}

	hint := "scoreboard"
	if g.State == "pre" {
		hint = "matchup"
	}

	return w.postRepo.UpsertSportsPost(gameID, title, body, g.League, string(gameDataJSON), hint)
}

func buildPostTitle(gd GameData) string {
	if gd.Home.Score != nil && gd.Away.Score != nil {
		return fmt.Sprintf("%s %d · %s %d", gd.Away.Abbr, *gd.Away.Score, gd.Home.Abbr, *gd.Home.Score)
	}
	return fmt.Sprintf("%s @ %s", gd.Away.Abbr, gd.Home.Abbr)
}

func buildPostBody(gd GameData) string {
	parts := []string{gd.Status}
	if gd.Series != nil {
		parts = append(parts, *gd.Series)
	}
	if gd.Headline != nil {
		parts = append(parts, *gd.Headline)
	}
	if gd.Venue != nil {
		parts = append(parts, *gd.Venue)
	}
	return strings.Join(parts, " · ")
}
