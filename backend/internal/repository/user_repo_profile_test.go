package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUpdateProfile(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewUserRepo(db)

	user, err := repo.FindOrCreateByFirebaseUID("firebase-profile-test")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	lat := 37.77
	lon := -122.42
	err = repo.UpdateProfile(user.ID, "Shane", "", "UTC-7", "San Francisco", &lat, &lon)
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}

	updated, err := repo.FindOrCreateByFirebaseUID("firebase-profile-test")
	if err != nil {
		t.Fatalf("refetch user: %v", err)
	}

	if updated.DisplayName != "Shane" {
		t.Errorf("display_name = %q, want %q", updated.DisplayName, "Shane")
	}
	if updated.Timezone != "UTC-7" {
		t.Errorf("timezone = %q, want %q", updated.Timezone, "UTC-7")
	}
	if updated.HomeLocation != "San Francisco" {
		t.Errorf("home_location = %q, want %q", updated.HomeLocation, "San Francisco")
	}
	if updated.HomeLat == nil || *updated.HomeLat != 37.77 {
		t.Errorf("home_lat = %v, want 37.77", updated.HomeLat)
	}
	if updated.ProfileUpdatedAt == nil {
		t.Error("profile_updated_at should be set after update")
	}
}
