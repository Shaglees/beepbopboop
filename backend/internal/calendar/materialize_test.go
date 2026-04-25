package calendar

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// createCalendarBotAgent creates a system user and calendar-bot agent for use in tests.
// Returns the agentID.
func createCalendarBotAgent(t *testing.T, db *sql.DB, suffix string) string {
	t.Helper()

	fbUID := "firebase-calbot-sys-" + suffix
	agentID := "calendar-bot-" + suffix

	var sysUserID string
	err := db.QueryRow(`
		INSERT INTO users (id, firebase_uid)
		VALUES (gen_random_uuid()::text, $1)
		ON CONFLICT (firebase_uid) DO UPDATE SET firebase_uid = EXCLUDED.firebase_uid
		RETURNING id`, fbUID).Scan(&sysUserID)
	if err != nil {
		t.Fatalf("create system user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO agents (id, user_id, name, status)
		VALUES ($1, $2, 'Calendar Bot', 'active')
		ON CONFLICT (id) DO NOTHING`,
		agentID, sysUserID,
	)
	if err != nil {
		t.Fatalf("create calendar-bot agent: %v", err)
	}

	return agentID
}

func TestMaterializeWorker_SportsPreview(t *testing.T) {
	db := database.OpenTestDB(t)

	calendarRepo := repository.NewCalendarEventRepo(db)
	postRepo := repository.NewPostRepo(db)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	agentID := createCalendarBotAgent(t, db, "preview")

	// Create test user
	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-materialize-preview-001")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	// Set basketball interest for the test user
	interests := []model.UserInterest{
		{Category: "sports", Topic: "basketball", Source: "user", Confidence: 1.0},
	}
	if err := interestRepo.BulkSetUser(user.ID, interests); err != nil {
		t.Fatalf("BulkSetUser: %v", err)
	}

	// Create a calendar event ~20 hours from now — falls in sports preview window (T-24h to T-12h)
	now := time.Now().UTC()
	startTime := now.Add(20 * time.Hour)

	payload, _ := json.Marshal(map[string]string{
		"home":        "Lakers",
		"away":        "Celtics",
		"home_record": "30-10",
		"away_record": "28-12",
		"venue":       "Crypto.com Arena",
		"broadcast":   "ESPN",
	})

	event := model.InterestCalendarEvent{
		EventKey:     "test-mat-sports-preview-001",
		Domain:       "sports",
		Title:        "Lakers vs Celtics",
		StartTime:    startTime,
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "game",
		EntityIDs:    json.RawMessage(`{}`),
		InterestTags: []string{"basketball", "nba"},
		Payload:      json.RawMessage(payload),
	}

	if err := calendarRepo.Upsert(event); err != nil {
		t.Fatalf("Upsert event: %v", err)
	}

	// Create the worker and run one cycle
	worker := NewMaterializeWorker(calendarRepo, postRepo, userRepo, interestRepo, agentID)
	worker.cycleOnce(context.Background())

	// Verify the preview post was published (IsPublished = true)
	published, err := calendarRepo.IsPublished(event.EventKey, user.ID, "preview")
	if err != nil {
		t.Fatalf("IsPublished: %v", err)
	}
	if !published {
		t.Error("expected IsPublished = true for sports preview after cycleOnce, got false")
	}

	// Imminent window should NOT be published (event is 20h away, not in T-2h to T-0 range)
	imminentPublished, err := calendarRepo.IsPublished(event.EventKey, user.ID, "imminent")
	if err != nil {
		t.Fatalf("IsPublished (imminent): %v", err)
	}
	if imminentPublished {
		t.Error("imminent post should not be published for an event 20h away")
	}
}

func TestMaterializeWorker_Idempotency(t *testing.T) {
	db := database.OpenTestDB(t)

	calendarRepo := repository.NewCalendarEventRepo(db)
	postRepo := repository.NewPostRepo(db)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	agentID := createCalendarBotAgent(t, db, "idem")

	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-materialize-idem-001")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	interests := []model.UserInterest{
		{Category: "sports", Topic: "basketball", Source: "user", Confidence: 1.0},
	}
	if err := interestRepo.BulkSetUser(user.ID, interests); err != nil {
		t.Fatalf("BulkSetUser: %v", err)
	}

	payload, _ := json.Marshal(map[string]string{
		"home": "Warriors", "away": "Bulls",
		"home_record": "20-15", "away_record": "18-17",
		"venue": "Chase Center", "broadcast": "TNT",
	})

	event := model.InterestCalendarEvent{
		EventKey:     "test-mat-idem-001",
		Domain:       "sports",
		Title:        "Warriors vs Bulls",
		StartTime:    time.Now().UTC().Add(20 * time.Hour),
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "game",
		EntityIDs:    json.RawMessage(`{}`),
		InterestTags: []string{"basketball"},
		Payload:      json.RawMessage(payload),
	}
	if err := calendarRepo.Upsert(event); err != nil {
		t.Fatalf("Upsert event: %v", err)
	}

	worker := NewMaterializeWorker(calendarRepo, postRepo, userRepo, interestRepo, agentID)

	// Run twice — should not create duplicate posts (idempotent via IsPublished check)
	worker.cycleOnce(context.Background())
	worker.cycleOnce(context.Background())

	// Count posts for this user created by the calendar bot
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE user_id = $1 AND agent_id = $2`,
		user.ID, agentID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count posts: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 post after 2 cycles (idempotent), got %d", count)
	}
}
