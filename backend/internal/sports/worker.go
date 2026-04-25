package sports

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// Worker periodically fetches live scores and upserts one post per game.
type Worker struct {
	svc          *Service
	postRepo     *repository.PostRepo
	calendarRepo *repository.CalendarEventRepo // nil means skip calendar upserts
	interval     time.Duration
}

func NewWorker(svc *Service, postRepo *repository.PostRepo, calendarRepo *repository.CalendarEventRepo, interval time.Duration) *Worker {
	return &Worker{svc: svc, postRepo: postRepo, calendarRepo: calendarRepo, interval: interval}
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

	// Upsert upcoming scheduled games to interest_calendar_events.
	if w.calendarRepo != nil {
		var calOK, calSkipped, calFailed int
		now := time.Now()
		horizon := now.Add(7 * 24 * time.Hour)
		for _, g := range games {
			result := w.upsertCalendarEvent(g, now, horizon)
			switch result {
			case calResultOK:
				calOK++
			case calResultSkipped:
				calSkipped++
			case calResultFailed:
				calFailed++
			}
		}
		slog.Info("sports worker: calendar upsert complete",
			"ok", calOK, "skipped", calSkipped, "failed", calFailed)
	}
}

func (w *Worker) processGame(g FetchedGame) error {
	gameID := g.League + "-" + g.EventID
	title := buildPostTitle(g.Data)
	body := buildPostBody(g.Data)

	gameDataJSON, err := json.Marshal(g.Data)
	if err != nil {
		return fmt.Errorf("marshal game data: %w", err)
	}

	return w.postRepo.UpsertSportsPost(gameID, title, body, g.League, string(gameDataJSON))
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

type calUpsertResult int

const (
	calResultOK      calUpsertResult = iota
	calResultSkipped                 // not scheduled or too far away
	calResultFailed
)

// upsertCalendarEvent upserts a scheduled game to interest_calendar_events.
// Returns calResultSkipped if the game is not scheduled or is beyond the horizon.
func (w *Worker) upsertCalendarEvent(g FetchedGame, now, horizon time.Time) calUpsertResult {
	// Only process scheduled (pre-game) events — GameTime is only set for "pre" state.
	if g.Data.GameTime == nil {
		return calResultSkipped
	}

	startTime, err := time.Parse(time.RFC3339, *g.Data.GameTime)
	if err != nil {
		slog.Warn("sports worker: could not parse game time", "league", g.League, "eventID", g.EventID, "gameTime", *g.Data.GameTime, "error", err)
		return calResultFailed
	}

	// Skip games that have already started or are too far in the future.
	if startTime.Before(now) || startTime.After(horizon) {
		return calResultSkipped
	}

	league := strings.ToLower(g.League)
	homeSlug := teamSlug(g.Data.Home.Name)
	awaySlug := teamSlug(g.Data.Away.Name)
	sportName := sportDisplayName(league)

	interestTags := []string{league, homeSlug, awaySlug, sportName}

	entityIDs, err := json.Marshal(map[string]string{
		"home_team": homeSlug,
		"away_team": awaySlug,
		"league":    league,
	})
	if err != nil {
		slog.Warn("sports worker: marshal entity_ids failed", "league", g.League, "eventID", g.EventID, "error", err)
		return calResultFailed
	}

	type calPayload struct {
		HomeTeam  string  `json:"home_team"`
		AwayTeam  string  `json:"away_team"`
		HomeAbbr  string  `json:"home_abbr"`
		AwayAbbr  string  `json:"away_abbr"`
		HomeRecord *string `json:"home_record,omitempty"`
		AwayRecord *string `json:"away_record,omitempty"`
		Venue     *string `json:"venue,omitempty"`
		Broadcast *string `json:"broadcast,omitempty"`
		League    string  `json:"league"`
	}
	payload, err := json.Marshal(calPayload{
		HomeTeam:  g.Data.Home.Name,
		AwayTeam:  g.Data.Away.Name,
		HomeAbbr:  g.Data.Home.Abbr,
		AwayAbbr:  g.Data.Away.Abbr,
		HomeRecord: g.Data.Home.Record,
		AwayRecord: g.Data.Away.Record,
		Venue:     g.Data.Venue,
		Broadcast: g.Data.Broadcast,
		League:    league,
	})
	if err != nil {
		slog.Warn("sports worker: marshal payload failed", "league", g.League, "eventID", g.EventID, "error", err)
		return calResultFailed
	}

	title := fmt.Sprintf("%s @ %s", g.Data.Away.Name, g.Data.Home.Name)
	eventKey := fmt.Sprintf("espn:event:%s", g.EventID)

	evt := model.InterestCalendarEvent{
		EventKey:     eventKey,
		Domain:       "sports",
		Title:        title,
		StartTime:    startTime,
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "game",
		EntityIDs:    entityIDs,
		InterestTags: interestTags,
		Payload:      payload,
	}

	if err := w.calendarRepo.Upsert(evt); err != nil {
		slog.Warn("sports worker: calendar upsert failed", "league", g.League, "eventID", g.EventID, "error", err)
		return calResultFailed
	}
	return calResultOK
}

// teamSlug returns a lowercase hyphenated slug from a team display name.
// e.g. "Los Angeles Lakers" → "los-angeles-lakers"
func teamSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}

// sportDisplayName maps a league name to its sport display name.
func sportDisplayName(league string) string {
	switch league {
	case "nba", "wnba":
		return "basketball"
	case "nfl":
		return "football"
	case "mlb":
		return "baseball"
	case "nhl":
		return "hockey"
	case "mls":
		return "soccer"
	default:
		return league
	}
}
