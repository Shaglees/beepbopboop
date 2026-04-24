package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUserContentPrefsRepo_SetAndList(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	prefsRepo := repository.NewUserContentPrefsRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-prefs-test")

	sports := "sports"
	maxFive := 5
	prefs := []model.ContentPref{
		{Category: nil, Depth: "standard", Tone: "casual", MaxPerDay: nil},
		{Category: &sports, Depth: "detailed", Tone: "informative", MaxPerDay: &maxFive},
	}

	err := prefsRepo.BulkSet(user.ID, prefs)
	if err != nil {
		t.Fatalf("BulkSet: %v", err)
	}

	got, err := prefsRepo.List(user.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d prefs, want 2", len(got))
	}
}
