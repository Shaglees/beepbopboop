package repository_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestCalendarEventRepo_Upsert(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewCalendarEventRepo(db)

	now := time.Now().UTC().Truncate(time.Second)
	e := model.InterestCalendarEvent{
		EventKey:     "test-upsert-event-001",
		Domain:       "sports",
		Title:        "Initial Title",
		StartTime:    now.Add(24 * time.Hour),
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "game",
		EntityIDs:    json.RawMessage(`{"team_id":"abc"}`),
		InterestTags: []string{"basketball", "nba"},
		Payload:      json.RawMessage(`{"league":"NBA"}`),
	}

	// First upsert.
	if err := repo.Upsert(e); err != nil {
		t.Fatalf("Upsert (first): %v", err)
	}

	// Upsert again with updated title.
	e.Title = "Updated Title"
	if err := repo.Upsert(e); err != nil {
		t.Fatalf("Upsert (second): %v", err)
	}

	// Verify exactly one event with the updated title.
	from := now
	to := now.Add(48 * time.Hour)
	results, err := repo.Upcoming("sports", from, to)
	if err != nil {
		t.Fatalf("Upcoming: %v", err)
	}

	var found []model.InterestCalendarEvent
	for _, r := range results {
		if r.EventKey == e.EventKey {
			found = append(found, r)
		}
	}

	if len(found) != 1 {
		t.Fatalf("expected exactly 1 event for event_key %q, got %d", e.EventKey, len(found))
	}
	if found[0].Title != "Updated Title" {
		t.Errorf("Title after upsert = %q, want %q", found[0].Title, "Updated Title")
	}
}

func TestCalendarEventRepo_ForUser(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewCalendarEventRepo(db)

	now := time.Now().UTC().Truncate(time.Second)
	from := now
	to := now.Add(72 * time.Hour)

	sports := model.InterestCalendarEvent{
		EventKey:     "test-foruser-sports-001",
		Domain:       "sports",
		Title:        "NBA Game",
		StartTime:    now.Add(24 * time.Hour),
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "game",
		EntityIDs:    json.RawMessage(`{}`),
		InterestTags: []string{"basketball", "nba"},
		Payload:      json.RawMessage(`{}`),
	}
	entertainment := model.InterestCalendarEvent{
		EventKey:     "test-foruser-entertainment-001",
		Domain:       "entertainment",
		Title:        "Sci-Fi Movie Premiere",
		StartTime:    now.Add(48 * time.Hour),
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "movie",
		EntityIDs:    json.RawMessage(`{}`),
		InterestTags: []string{"sci-fi"},
		Payload:      json.RawMessage(`{}`),
	}

	if err := repo.Upsert(sports); err != nil {
		t.Fatalf("Upsert sports: %v", err)
	}
	if err := repo.Upsert(entertainment); err != nil {
		t.Fatalf("Upsert entertainment: %v", err)
	}

	// Query with "basketball" interest → only sports.
	basketballResults, err := repo.ForUser("user-any", []string{"basketball"}, from, to)
	if err != nil {
		t.Fatalf("ForUser basketball: %v", err)
	}
	foundSports := false
	foundEntertainment := false
	for _, r := range basketballResults {
		if r.EventKey == sports.EventKey {
			foundSports = true
		}
		if r.EventKey == entertainment.EventKey {
			foundEntertainment = true
		}
	}
	if !foundSports {
		t.Error("expected sports event when filtering by 'basketball'")
	}
	if foundEntertainment {
		t.Error("entertainment event should not appear when filtering by 'basketball'")
	}

	// Query with "sci-fi" interest → only entertainment.
	scifiResults, err := repo.ForUser("user-any", []string{"sci-fi"}, from, to)
	if err != nil {
		t.Fatalf("ForUser sci-fi: %v", err)
	}
	foundSports = false
	foundEntertainment = false
	for _, r := range scifiResults {
		if r.EventKey == sports.EventKey {
			foundSports = true
		}
		if r.EventKey == entertainment.EventKey {
			foundEntertainment = true
		}
	}
	if foundSports {
		t.Error("sports event should not appear when filtering by 'sci-fi'")
	}
	if !foundEntertainment {
		t.Error("expected entertainment event when filtering by 'sci-fi'")
	}

	// Empty interests returns nil.
	nilResults, err := repo.ForUser("user-any", nil, from, to)
	if err != nil {
		t.Fatalf("ForUser nil interests: %v", err)
	}
	if nilResults != nil {
		t.Errorf("expected nil for empty interests, got %v", nilResults)
	}
}

func TestCalendarPostLog_Dedup(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewCalendarEventRepo(db)

	// Create a test user and agent with UUID-formatted IDs (calendar_post_log.user_id is UUID type).
	var userID, agentID, postID string
	err := db.QueryRow(`
		INSERT INTO users (id, firebase_uid)
		VALUES (gen_random_uuid()::text, 'firebase-cal-dedup-test')
		RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	err = db.QueryRow(`
		INSERT INTO agents (id, user_id, name, status)
		VALUES (gen_random_uuid()::text, $1, 'test-agent', 'active')
		RETURNING id`, userID).Scan(&agentID)
	if err != nil {
		t.Fatalf("create test agent: %v", err)
	}

	err = db.QueryRow(`
		INSERT INTO posts (id, agent_id, user_id, title, body)
		VALUES (gen_random_uuid()::text, $1, $2, 'Test Post', 'Body')
		RETURNING id`, agentID, userID).Scan(&postID)
	if err != nil {
		t.Fatalf("create test post: %v", err)
	}

	eventKey := "test-dedup-event-001"
	window := "day"

	// Before LogPost: IsPublished should be false.
	published, err := repo.IsPublished(eventKey, userID, window)
	if err != nil {
		t.Fatalf("IsPublished (before): %v", err)
	}
	if published {
		t.Error("expected IsPublished = false before LogPost")
	}

	// LogPost.
	if err := repo.LogPost(eventKey, userID, window, postID); err != nil {
		t.Fatalf("LogPost: %v", err)
	}

	// After LogPost: IsPublished should be true.
	published, err = repo.IsPublished(eventKey, userID, window)
	if err != nil {
		t.Fatalf("IsPublished (after): %v", err)
	}
	if !published {
		t.Error("expected IsPublished = true after LogPost")
	}

	// LogPost again with same combo — ON CONFLICT DO NOTHING, no error.
	if err := repo.LogPost(eventKey, userID, window, postID); err != nil {
		t.Fatalf("LogPost (duplicate): %v", err)
	}

	// Still only one row.
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM calendar_post_log
		WHERE event_key = $1 AND user_id = $2 AND "window" = $3`,
		eventKey, userID, window,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count calendar_post_log: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row in calendar_post_log after dedup, got %d", count)
	}
}
