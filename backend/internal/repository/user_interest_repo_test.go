package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUserInterestRepo_BulkSetAndList(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-interest-test")

	interests := []model.UserInterest{
		{Category: "sports", Topic: "NBA", Source: "user", Confidence: 1.0},
		{Category: "food", Topic: "ramen", Source: "user", Confidence: 1.0},
	}

	err := interestRepo.BulkSetUser(user.ID, interests)
	if err != nil {
		t.Fatalf("BulkSetUser: %v", err)
	}

	got, err := interestRepo.ListActive(user.ID)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d interests, want 2", len(got))
	}
	if got[0].Category != "food" && got[0].Category != "sports" {
		t.Errorf("unexpected category: %q", got[0].Category)
	}
}

func TestUserInterestRepo_BulkSetReplacesExisting(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-interest-replace")

	first := []model.UserInterest{
		{Category: "sports", Topic: "NBA", Source: "user", Confidence: 1.0},
		{Category: "food", Topic: "ramen", Source: "user", Confidence: 1.0},
	}
	interestRepo.BulkSetUser(user.ID, first)

	second := []model.UserInterest{
		{Category: "music", Topic: "indie rock", Source: "user", Confidence: 1.0},
	}
	interestRepo.BulkSetUser(user.ID, second)

	got, _ := interestRepo.ListActive(user.ID)
	if len(got) != 1 {
		t.Fatalf("got %d interests, want 1 (replaced)", len(got))
	}
	if got[0].Category != "music" {
		t.Errorf("category = %q, want music", got[0].Category)
	}
}

func TestUserInterestRepo_PauseAndDismiss(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-interest-pause")

	interests := []model.UserInterest{
		{Category: "sports", Topic: "NFL", Source: "user", Confidence: 1.0},
	}
	interestRepo.BulkSetUser(user.ID, interests)

	all, _ := interestRepo.ListActive(user.ID)
	if len(all) != 1 {
		t.Fatalf("setup: got %d, want 1", len(all))
	}

	// Pause for 120 days
	err := interestRepo.Pause(all[0].ID, 120)
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}

	// Should be excluded from active list
	active, _ := interestRepo.ListActive(user.ID)
	if len(active) != 0 {
		t.Errorf("paused interest should be excluded from active, got %d", len(active))
	}

	// ListAll should include it
	allInc, _ := interestRepo.ListAll(user.ID)
	if len(allInc) != 1 {
		t.Fatalf("ListAll should include paused, got %d", len(allInc))
	}
	if allInc[0].PausedUntil == nil {
		t.Error("PausedUntil should be set")
	}
}

func TestUserInterestRepo_InferredPreservation(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-interest-inferred")

	// Add an inferred interest directly
	err := interestRepo.UpsertInferred(user.ID, "travel", "Japan", 0.8)
	if err != nil {
		t.Fatalf("UpsertInferred: %v", err)
	}

	// BulkSetUser (user-declared) should NOT remove inferred ones
	userInterests := []model.UserInterest{
		{Category: "sports", Topic: "NBA", Source: "user", Confidence: 1.0},
	}
	interestRepo.BulkSetUser(user.ID, userInterests)

	all, _ := interestRepo.ListActive(user.ID)
	if len(all) != 2 {
		t.Fatalf("got %d interests, want 2 (1 user + 1 inferred)", len(all))
	}
}
