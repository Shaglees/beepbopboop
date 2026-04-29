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

	h := handler.NewProfileHandler(userRepo, agentRepo, interestRepo, lifestyleRepo, prefsRepo, settingsRepo, repository.NewUserSkillRepo(db))

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

	h := handler.NewProfileHandler(userRepo, agentRepo, interestRepo, lifestyleRepo, prefsRepo, settingsRepo, repository.NewUserSkillRepo(db))

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

func TestGetProfileAgent_IncludesUserSkills(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	lifestyleRepo := repository.NewUserLifestyleRepo(db)
	prefsRepo := repository.NewUserContentPrefsRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	settingsRepo := repository.NewUserSettingsRepo(db)
	skillRepo := repository.NewUserSkillRepo(db)

	h := handler.NewProfileHandler(userRepo, agentRepo, interestRepo, lifestyleRepo, prefsRepo, settingsRepo, skillRepo)

	user, _ := userRepo.FindOrCreateByFirebaseUID("fb-profile-agent")
	agent, _ := agentRepo.Create(user.ID, "openclaw")
	_, err := skillRepo.Upsert(user.ID, "local-skill", model.UserSkillKindStandalone, "", "intent", 14, nil,
		[]repository.FileInput{{Path: "SKILL.md", Content: []byte("---\nname: local-skill\n---\n")}})
	if err != nil {
		t.Fatalf("seed skill: %v", err)
	}

	req := httptest.NewRequest("GET", "/user/profile", nil)
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	w := httptest.NewRecorder()
	h.GetProfileAgent(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var profile model.UserProfile
	if err := json.NewDecoder(w.Body).Decode(&profile); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(profile.UserSkills) != 1 {
		t.Fatalf("expected one user_skill in agent profile, got %d", len(profile.UserSkills))
	}
	if profile.UserSkills[0].Name != "local-skill" {
		t.Errorf("got skill name %q, want local-skill", profile.UserSkills[0].Name)
	}
	if len(profile.UserSkills[0].Files) != 1 || profile.UserSkills[0].Files[0].SHA256 == "" {
		t.Errorf("file metadata missing: %+v", profile.UserSkills[0].Files)
	}
}

func TestGetProfileFirebase_OmitsUserSkills(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	interestRepo := repository.NewUserInterestRepo(db)
	lifestyleRepo := repository.NewUserLifestyleRepo(db)
	prefsRepo := repository.NewUserContentPrefsRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	settingsRepo := repository.NewUserSettingsRepo(db)
	skillRepo := repository.NewUserSkillRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("fb-profile-fb")
	_, err := skillRepo.Upsert(user.ID, "ios-only", model.UserSkillKindStandalone, "", "intent", 14, nil,
		[]repository.FileInput{{Path: "SKILL.md", Content: []byte("---\nname: ios-only\n---\n")}})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	h := handler.NewProfileHandler(userRepo, agentRepo, interestRepo, lifestyleRepo, prefsRepo, settingsRepo, skillRepo)
	req := httptest.NewRequest("GET", "/user/profile", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "fb-profile-fb"))
	w := httptest.NewRecorder()
	h.GetProfileFirebase(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var profile model.UserProfile
	json.NewDecoder(w.Body).Decode(&profile)
	if len(profile.UserSkills) != 0 {
		t.Errorf("firebase profile must not include user_skills (agent-only field), got %+v", profile.UserSkills)
	}
}
