package interest_test

import (
	"context"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/interest"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestWorker_InfersFromEngagement(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-infer-test")

	// Seed engagement: user saved 10 sports posts in last 30 days
	for i := 0; i < 10; i++ {
		postID, _ := repository.GenerateTestID()
		db.Exec(`INSERT INTO posts (id, agent_id, user_id, title, body, labels, status, display_hint)
			VALUES ($1, 'agent1', $2, 'test', 'body', '["sports"]', 'published', 'card')`,
			postID, user.ID)
		db.Exec(`INSERT INTO post_events (id, post_id, user_id, event_type, created_at)
			VALUES ($1, $2, $3, 'save', NOW() - INTERVAL '1 day')`,
			postID+"evt", postID, user.ID)
	}

	w := interest.NewWorker(db, interestRepo)
	err := w.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	interests, _ := interestRepo.ListActive(user.ID)
	found := false
	for _, i := range interests {
		if i.Category == "sports" && i.Source == "inferred" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected inferred 'sports' interest from engagement data")
	}
}
