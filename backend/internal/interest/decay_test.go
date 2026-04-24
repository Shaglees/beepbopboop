package interest_test

import (
	"context"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/interest"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestDecayChecker_GeneratesFeedbackPost(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-decay-test")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "Briefing")

	interestRepo.BulkSetUser(user.ID, []model.UserInterest{
		{Category: "sports", Topic: "NFL", Confidence: 1.0},
	})

	db.Exec("UPDATE user_interests SET created_at = NOW() - INTERVAL '45 days' WHERE user_id = $1", user.ID)

	checker := interest.NewDecayChecker(db, interestRepo, postRepo, agent.ID)
	err := checker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	posts, _ := postRepo.ListByUserID(user.ID, 10)
	found := false
	for _, p := range posts {
		if p.DisplayHint == "feedback" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a feedback post for the disengaged interest")
	}

	interests, _ := interestRepo.ListAll(user.ID)
	if len(interests) == 0 {
		t.Fatal("no interests found")
	}
	if interests[0].TimesAsked != 1 {
		t.Errorf("times_asked = %d, want 1", interests[0].TimesAsked)
	}
}

func TestDecayChecker_RespectsBackoff(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	postRepo := repository.NewPostRepo(db)
	agentRepo := repository.NewAgentRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-decay-backoff")
	agent, _ := agentRepo.Create(user.ID, "Briefing")

	interestRepo.BulkSetUser(user.ID, []model.UserInterest{
		{Category: "sports", Topic: "NFL", Confidence: 1.0},
	})
	db.Exec("UPDATE user_interests SET created_at = NOW() - INTERVAL '45 days' WHERE user_id = $1", user.ID)
	db.Exec("UPDATE user_interests SET last_asked_at = NOW() - INTERVAL '10 days', times_asked = 1 WHERE user_id = $1", user.ID)

	checker := interest.NewDecayChecker(db, interestRepo, postRepo, agent.ID)
	checker.RunOnce(context.Background())

	posts, _ := postRepo.ListByUserID(user.ID, 10)
	for _, p := range posts {
		if p.DisplayHint == "feedback" {
			t.Error("should not generate feedback post during backoff period")
		}
	}
}

func TestDecayChecker_StopsAfterMaxAsks(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	postRepo := repository.NewPostRepo(db)
	agentRepo := repository.NewAgentRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-decay-maxask")
	agent, _ := agentRepo.Create(user.ID, "Briefing")

	interestRepo.BulkSetUser(user.ID, []model.UserInterest{
		{Category: "sports", Topic: "NFL", Confidence: 1.0},
	})
	db.Exec("UPDATE user_interests SET created_at = NOW() - INTERVAL '45 days', times_asked = 3, last_asked_at = NOW() - INTERVAL '100 days' WHERE user_id = $1", user.ID)

	checker := interest.NewDecayChecker(db, interestRepo, postRepo, agent.ID)
	checker.RunOnce(context.Background())

	posts, _ := postRepo.ListByUserID(user.ID, 10)
	for _, p := range posts {
		if p.DisplayHint == "feedback" {
			t.Error("should not generate feedback post after max asks (3)")
		}
	}
}
