package calendar

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// Worker scans upcoming calendar events for users who have enabled calendar
// integration and generates anticipatory posts targeted to each user.
type Worker struct {
	calendarRepo     *repository.CalendarRepo
	postRepo         *repository.PostRepo
	userSettingsRepo *repository.UserSettingsRepo
	interval         time.Duration
}

func NewWorker(
	calendarRepo *repository.CalendarRepo,
	postRepo *repository.PostRepo,
	userSettingsRepo *repository.UserSettingsRepo,
	interval time.Duration,
) *Worker {
	return &Worker{
		calendarRepo:     calendarRepo,
		postRepo:         postRepo,
		userSettingsRepo: userSettingsRepo,
		interval:         interval,
	}
}

// Run starts the anticipatory worker loop with an immediate first cycle.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("calendar worker started", "interval", w.interval)
	w.cycle()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("calendar worker stopped")
			return
		case <-ticker.C:
			w.cycle()
		}
	}
}

func (w *Worker) cycle() {
	userIDs, err := w.userSettingsRepo.UsersWithCalendarEnabled()
	if err != nil {
		slog.Error("calendar worker: failed to list calendar users", "error", err)
		return
	}
	if len(userIDs) == 0 {
		slog.Debug("calendar worker: no calendar users, skipping")
		return
	}

	now := time.Now()
	lookahead := now.Add(7 * 24 * time.Hour)

	var ok, failed int
	for _, userID := range userIDs {
		if err := w.processUser(userID, now, lookahead); err != nil {
			slog.Warn("calendar worker: user failed", "user_id", userID, "error", err)
			failed++
		} else {
			ok++
		}
	}
	slog.Info("calendar worker: cycle complete", "users", len(userIDs), "ok", ok, "failed", failed)
}

func (w *Worker) processUser(userID string, from, to time.Time) error {
	events, err := w.calendarRepo.UpcomingEvents(userID, from, to)
	if err != nil {
		return fmt.Errorf("upcoming events: %w", err)
	}
	for _, e := range events {
		intent := extractIntent(e)
		if intent == nil {
			continue
		}
		postKey := fmt.Sprintf("anticipatory-%s-%s", userID, e.ID)
		title, body := buildPost(*intent, e)
		if err := w.postRepo.UpsertAnticipatoryPost(postKey, userID, title, body, intent.postType, intent.labels); err != nil {
			slog.Warn("calendar worker: upsert post failed", "user_id", userID, "event_id", e.ID, "error", err)
		}
	}
	return nil
}

// intent represents a parsed calendar signal.
type intent struct {
	kind     string
	postType string
	labels   []string
	payload  map[string]string
}

var (
	flightRe    = regexp.MustCompile(`(?i)\b(flight|flying|fly|depart|departure|arrive|arrival)\b`)
	cityRe      = regexp.MustCompile(`(?i)\b(?:to|from|in)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*)\b`)
	sportRe     = regexp.MustCompile(`(?i)\b(game|match|vs\.?|versus|final|playoff|championship)\b`)
	conferenceRe = regexp.MustCompile(`(?i)\b(conference|summit|meetup|workshop|symposium|expo|hackathon)\b`)
	restaurantRe = regexp.MustCompile(`(?i)\b(dinner|lunch|brunch|breakfast|restaurant|cafe|bistro)\b`)
)

func extractIntent(e model.CalendarEvent) *intent {
	text := strings.ToLower(e.Title + " " + e.Location + " " + e.Notes)

	switch {
	case flightRe.MatchString(e.Title) || strings.Contains(text, "airport") || strings.Contains(text, "terminal"):
		city := extractCity(e)
		return &intent{
			kind:     "travel",
			postType: "discovery",
			labels:   []string{"travel"},
			payload:  map[string]string{"city": city},
		}

	case sportRe.MatchString(e.Title):
		return &intent{
			kind:     "sports",
			postType: "discovery",
			labels:   []string{"sports"},
			payload:  map[string]string{"event": e.Title},
		}

	case conferenceRe.MatchString(e.Title):
		city := extractCity(e)
		return &intent{
			kind:     "conference",
			postType: "discovery",
			labels:   []string{"news"},
			payload:  map[string]string{"event": e.Title, "city": city},
		}

	case restaurantRe.MatchString(e.Title):
		return &intent{
			kind:     "food",
			postType: "discovery",
			labels:   []string{"food"},
			payload:  map[string]string{"event": e.Title},
		}
	}
	return nil
}

func extractCity(e model.CalendarEvent) string {
	// Try location field first, then title.
	for _, src := range []string{e.Location, e.Title} {
		if m := cityRe.FindStringSubmatch(src); len(m) > 1 {
			return m[1]
		}
	}
	if e.Location != "" {
		return e.Location
	}
	return ""
}

func buildPost(i intent, e model.CalendarEvent) (title, body string) {
	when := timeLabel(e.StartTime)
	switch i.kind {
	case "travel":
		city := i.payload["city"]
		if city == "" {
			city = "your destination"
		}
		title = fmt.Sprintf("Getting ready for %s", city)
		body = fmt.Sprintf("You have travel coming up %s. Here's what to know about %s — weather, local tips, and food worth trying.", when, city)
	case "sports":
		title = fmt.Sprintf("Game day: %s", e.Title)
		body = fmt.Sprintf("You have \"%s\" %s. Check live scores, venue info, and nearby spots.", e.Title, when)
	case "conference":
		city := i.payload["city"]
		if city == "" {
			title = fmt.Sprintf("Heads up: %s", e.Title)
			body = fmt.Sprintf("Your event \"%s\" is %s. Speaker news and practical tips incoming.", e.Title, when)
		} else {
			title = fmt.Sprintf("%s in %s", e.Title, city)
			body = fmt.Sprintf("Your event \"%s\" in %s is %s. Local tips and venue info to help you prepare.", e.Title, city, when)
		}
	case "food":
		title = fmt.Sprintf("Heading out %s", when)
		body = fmt.Sprintf("You have \"%s\" coming up. Discover nearby options and what's worth ordering.", e.Title)
	default:
		title = e.Title
		body = fmt.Sprintf("Upcoming: %s (%s)", e.Title, when)
	}
	return
}

func timeLabel(t time.Time) string {
	now := time.Now()
	diff := t.Sub(now)
	switch {
	case diff < 24*time.Hour:
		return "today"
	case diff < 48*time.Hour:
		return "tomorrow"
	case diff < 7*24*time.Hour:
		return "this week"
	default:
		return "soon"
	}
}
