package sports

import (
	"testing"
	"time"
)

func TestTeamSlug(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Los Angeles Lakers", "los-angeles-lakers"},
		{"Boston Celtics", "boston-celtics"},
		{"76ers", "76ers"},
	}
	for _, tt := range tests {
		if got := teamSlug(tt.name); got != tt.want {
			t.Errorf("teamSlug(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSportDisplayName(t *testing.T) {
	tests := []struct {
		league string
		want   string
	}{
		{"nba", "basketball"},
		{"nfl", "football"},
		{"mlb", "baseball"},
		{"nhl", "hockey"},
		{"mls", "soccer"},
		{"wnba", "basketball"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		if got := sportDisplayName(tt.league); got != tt.want {
			t.Errorf("sportDisplayName(%q) = %q, want %q", tt.league, got, tt.want)
		}
	}
}

func TestUpsertCalendarEvent_SkipsNonScheduled(t *testing.T) {
	// Worker with nil calendarRepo — skipped cases return before calling Upsert.
	w := &Worker{calendarRepo: nil}
	now := time.Now()
	horizon := now.Add(7 * 24 * time.Hour)

	// Game without GameTime (live or final) should be skipped.
	g := FetchedGame{
		EventID: "401234",
		League:  "nba",
		Data: GameData{
			Sport:  "basketball",
			Status: "live",
			Home:   TeamInfo{Name: "Lakers", Abbr: "LAL"},
			Away:   TeamInfo{Name: "Celtics", Abbr: "BOS"},
		},
	}

	result := w.upsertCalendarEvent(g, now, horizon)
	if result != calResultSkipped {
		t.Errorf("expected calResultSkipped for live game, got %d", result)
	}
}

func TestUpsertCalendarEvent_SkipsPastGames(t *testing.T) {
	w := &Worker{calendarRepo: nil}
	now := time.Now()
	horizon := now.Add(7 * 24 * time.Hour)

	pastTime := now.Add(-2 * time.Hour).Format(time.RFC3339)
	g := FetchedGame{
		EventID: "401235",
		League:  "nba",
		Data: GameData{
			Sport:    "basketball",
			Status:   "pre",
			GameTime: &pastTime,
			Home:     TeamInfo{Name: "Lakers", Abbr: "LAL"},
			Away:     TeamInfo{Name: "Celtics", Abbr: "BOS"},
		},
	}

	result := w.upsertCalendarEvent(g, now, horizon)
	if result != calResultSkipped {
		t.Errorf("expected calResultSkipped for past game, got %d", result)
	}
}

func TestUpsertCalendarEvent_SkipsBeyondHorizon(t *testing.T) {
	w := &Worker{calendarRepo: nil}
	now := time.Now()
	horizon := now.Add(7 * 24 * time.Hour)

	farFuture := now.Add(10 * 24 * time.Hour).Format(time.RFC3339)
	g := FetchedGame{
		EventID: "401236",
		League:  "nba",
		Data: GameData{
			Sport:    "basketball",
			Status:   "pre",
			GameTime: &farFuture,
			Home:     TeamInfo{Name: "Lakers", Abbr: "LAL"},
			Away:     TeamInfo{Name: "Celtics", Abbr: "BOS"},
		},
	}

	result := w.upsertCalendarEvent(g, now, horizon)
	if result != calResultSkipped {
		t.Errorf("expected calResultSkipped for far-future game, got %d", result)
	}
}
