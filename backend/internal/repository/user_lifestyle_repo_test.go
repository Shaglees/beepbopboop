package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUserLifestyleRepo_BulkSetAndList(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	lifestyleRepo := repository.NewUserLifestyleRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-lifestyle-test")

	tags := []model.LifestyleTag{
		{Category: "diet", Value: "vegetarian"},
		{Category: "pets", Value: "dog_owner"},
	}

	err := lifestyleRepo.BulkSet(user.ID, tags)
	if err != nil {
		t.Fatalf("BulkSet: %v", err)
	}

	got, err := lifestyleRepo.List(user.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d tags, want 2", len(got))
	}
}
