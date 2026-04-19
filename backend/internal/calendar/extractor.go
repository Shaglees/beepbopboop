// Package calendar extracts structured intent signals from raw calendar events.
// It uses lightweight pattern matching (no LLM calls) to map event titles/locations
// into typed UserIntent records that feed into the discovery ranking layer.
package calendar

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
)

// intentRule is a function that attempts to classify a calendar event.
// It returns (intentType, payload, ok).
type intentRule func(event model.CalendarEvent) (string, json.RawMessage, bool)

var rules = []intentRule{
	sportsRule,
	travelRule,
	fitnessRule,
	foodRule,
	conferenceRule,
}

// ExtractIntents analyses events and returns zero or more UserIntent records.
// activeUntil for each intent is the event start time so stale intents expire automatically.
func ExtractIntents(userID string, events []model.CalendarEvent) []model.UserIntent {
	var intents []model.UserIntent
	now := time.Now()

	for _, ev := range events {
		// Skip events already in the past.
		if ev.StartTime.Before(now) {
			continue
		}

		for _, rule := range rules {
			intentType, payload, ok := rule(ev)
			if !ok {
				continue
			}

			// Surface the intent starting 48h before the event (sports/travel)
			// or 24h before (general).
			leadTime := 48 * time.Hour
			if intentType == "conference" {
				leadTime = 5 * 24 * time.Hour
			}

			activeFrom := ev.StartTime.Add(-leadTime)
			if activeFrom.Before(now) {
				activeFrom = now
			}

			id := generateIntentID(userID, intentType, ev.StartTime)
			intent := model.UserIntent{
				ID:          id,
				UserID:      userID,
				SignalType:  "calendar",
				IntentType:  intentType,
				Payload:     payload,
				ActiveFrom:  activeFrom,
				ActiveUntil: ev.StartTime.Add(4 * time.Hour), // intent expires a few hours after the event
				CreatedAt:   now,
			}
			intents = append(intents, intent)
			break // one intent per event is sufficient
		}
	}

	return intents
}

// IntentLabelBoosts converts a slice of active UserIntents into a map of
// label -> boost score suitable for merging into FeedWeights.LabelWeights.
// Boosts are additive on top of existing weights.
func IntentLabelBoosts(intents []model.UserIntent) map[string]float64 {
	boosts := make(map[string]float64)
	now := time.Now()

	for _, intent := range intents {
		if now.Before(intent.ActiveFrom) || now.After(intent.ActiveUntil) {
			continue
		}

		// Scale boost by how imminent the event is: max 1.2 within 24h, 0.6 at 48h.
		urgency := urgencyFactor(intent.ActiveUntil, now)

		switch intent.IntentType {
		case "sports":
			boosts["sports"] += urgency * 1.2
		case "travel":
			boosts["travel"] += urgency * 1.0
			boosts["food"] += urgency * 0.4
		case "fitness":
			boosts["fitness"] += urgency * 1.0
			boosts["health"] += urgency * 0.4
		case "food":
			boosts["food"] += urgency * 0.8
			boosts["discovery"] += urgency * 0.3
		case "conference":
			boosts["article"] += urgency * 0.6
			boosts["trending"] += urgency * 0.4
		}
	}

	return boosts
}

// urgencyFactor returns a 0.5–1.0 multiplier based on how close the event is.
// Within 12h → 1.0, at 48h → 0.5, linear interpolation in between.
func urgencyFactor(eventTime, now time.Time) float64 {
	hoursUntil := eventTime.Sub(now).Hours()
	if hoursUntil <= 0 {
		return 1.0
	}
	if hoursUntil >= 48 {
		return 0.5
	}
	// Linear: 1.0 at 0h, 0.5 at 48h
	return 1.0 - (hoursUntil/48)*0.5
}

// --- Rules ---

func sportsRule(ev model.CalendarEvent) (string, json.RawMessage, bool) {
	title := strings.ToLower(ev.Title)

	// Common patterns: "Warriors vs Lakers", "GSW @ LAL", "NBA game", team names
	sportsKeywords := []string{
		" vs ", " @ ", " v ", " game", "match", "nba ", "nfl ", "mlb ", "nhl ",
		"soccer", "football", "basketball", "baseball", "hockey", "tennis",
		"grand prix", "f1 ", "race", "playoff", "championship", "final",
	}

	for _, kw := range sportsKeywords {
		if strings.Contains(title, kw) {
			payload, _ := json.Marshal(map[string]string{"event_title": ev.Title})
			return "sports", payload, true
		}
	}
	return "", nil, false
}

func travelRule(ev model.CalendarEvent) (string, json.RawMessage, bool) {
	title := strings.ToLower(ev.Title)
	loc := strings.ToLower(ev.Location)

	travelKeywords := []string{
		"flight", "fly ", "travel", "trip to", "vacation", "holiday",
		"hotel", "airbnb", "check-in", "check in", "depart", "arrive",
	}
	locationKeywords := []string{"airport", "terminal", "jfk", "lax", "lhr", "sfo", "ord", "atl"}

	for _, kw := range travelKeywords {
		if strings.Contains(title, kw) {
			payload, _ := json.Marshal(map[string]string{
				"event_title": ev.Title,
				"location":    ev.Location,
			})
			return "travel", payload, true
		}
	}
	for _, kw := range locationKeywords {
		if strings.Contains(loc, kw) {
			payload, _ := json.Marshal(map[string]string{
				"event_title": ev.Title,
				"location":    ev.Location,
			})
			return "travel", payload, true
		}
	}
	return "", nil, false
}

func fitnessRule(ev model.CalendarEvent) (string, json.RawMessage, bool) {
	title := strings.ToLower(ev.Title)
	fitnessKeywords := []string{
		"run", "gym", "workout", "marathon", "5k", "10k", "half marathon",
		"cycling", "bike ride", "yoga", "pilates", "crossfit", "swim",
		"triathlon", "hike", "hiking", "fitness", "training", "spartan",
	}
	for _, kw := range fitnessKeywords {
		if strings.Contains(title, kw) {
			payload, _ := json.Marshal(map[string]string{"event_title": ev.Title})
			return "fitness", payload, true
		}
	}
	return "", nil, false
}

func foodRule(ev model.CalendarEvent) (string, json.RawMessage, bool) {
	title := strings.ToLower(ev.Title)
	foodKeywords := []string{
		"dinner", "lunch", "breakfast", "brunch", "restaurant",
		"drinks", "happy hour", "date night", "food tour", "tasting",
	}
	for _, kw := range foodKeywords {
		if strings.Contains(title, kw) {
			payload, _ := json.Marshal(map[string]string{
				"event_title": ev.Title,
				"location":    ev.Location,
			})
			return "food", payload, true
		}
	}
	return "", nil, false
}

func conferenceRule(ev model.CalendarEvent) (string, json.RawMessage, bool) {
	title := strings.ToLower(ev.Title)
	confKeywords := []string{
		"conference", "summit", "meetup", "meet up", "hackathon",
		"workshop", "seminar", "expo", "symposium", "forum",
	}
	for _, kw := range confKeywords {
		if strings.Contains(title, kw) {
			payload, _ := json.Marshal(map[string]string{
				"event_title": ev.Title,
				"location":    ev.Location,
			})
			return "conference", payload, true
		}
	}
	return "", nil, false
}

// generateIntentID creates a stable deterministic ID for an intent so upserts work correctly.
func generateIntentID(userID, intentType string, eventTime time.Time) string {
	// Format: "cal:{userID}:{intentType}:{date}" — one intent per type per day per user.
	return "cal:" + userID + ":" + intentType + ":" + eventTime.UTC().Format("2006-01-02T15")
}
