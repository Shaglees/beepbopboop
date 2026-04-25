package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestGetProfile_Empty(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	lifestyleRepo := repository.NewUserLifestyleRepo(db)
	prefsRepo := repository.NewUserContentPrefsRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	settingsRepo := repository.NewUserSettingsRepo(db)

	h := handler.NewProfileHandler(userRepo, agentRepo, interestRepo, lifestyleRepo, prefsRepo, settingsRepo)

	req := httptest.NewRequest("GET", "/user/profile", nil)
	ctx := middleware.WithFirebaseUID(req.Context(), "firebase-profile-empty")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.GetProfileFirebase(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var profile model.UserProfile
	json.NewDecoder(w.Body).Decode(&profile)

	if profile.ProfileInitialized {
		t.Error("profile_initialized should be false for new user")
	}
	if profile.Identity.Timezone != "UTC+0" {
		t.Errorf("timezone = %q, want UTC+0", profile.Identity.Timezone)
	}
}

func TestGetProfile_WithData(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	lifestyleRepo := repository.NewUserLifestyleRepo(db)
	prefsRepo := repository.NewUserContentPrefsRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	settingsRepo := repository.NewUserSettingsRepo(db)

	h := handler.NewProfileHandler(userRepo, agentRepo, interestRepo, lifestyleRepo, prefsRepo, settingsRepo)

	// Setup user with profile data
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-profile-data")
	lat := 37.77
	lon := -122.42
	userRepo.UpdateProfile(user.ID, "Shane", "", "UTC-7", "San Francisco", &lat, &lon)

	interestRepo.BulkSetUser(user.ID, []model.UserInterest{
		{Category: "sports", Topic: "NBA", Confidence: 1.0},
	})
	lifestyleRepo.BulkSet(user.ID, []model.LifestyleTag{
		{Category: "diet", Value: "vegetarian"},
	})

	req := httptest.NewRequest("GET", "/user/profile", nil)
	ctx := middleware.WithFirebaseUID(req.Context(), "firebase-profile-data")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.GetProfileFirebase(w, req)

	var profile model.UserProfile
	json.NewDecoder(w.Body).Decode(&profile)

	if !profile.ProfileInitialized {
		t.Error("profile_initialized should be true")
	}
	if profile.Identity.DisplayName != "Shane" {
		t.Errorf("display_name = %q, want Shane", profile.Identity.DisplayName)
	}
	if len(profile.Interests) != 1 {
		t.Errorf("got %d interests, want 1", len(profile.Interests))
	}
	if len(profile.Lifestyle) != 1 {
		t.Errorf("got %d lifestyle tags, want 1", len(profile.Lifestyle))
	}
}
