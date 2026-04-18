package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestSettingsHandler_GetSettings_Default(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	h := handler.NewSettingsHandler(userRepo, userSettingsRepo)

	req := httptest.NewRequest("GET", "/user/settings", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.GetSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var settings map[string]any
	json.NewDecoder(rec.Body).Decode(&settings)
	if settings["radius_km"] != 25.0 {
		t.Errorf("expected default radius_km 25, got %v", settings["radius_km"])
	}
}

func TestSettingsHandler_UpdateSettings_FollowedTeams(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	h := handler.NewSettingsHandler(userRepo, userSettingsRepo)

	body := `{
		"location_name": "Toronto",
		"latitude": 43.651070,
		"longitude": -79.347015,
		"radius_km": 25,
		"followed_teams": ["nhl:tor", "nba:lal"]
	}`
	req := httptest.NewRequest("PUT", "/user/settings", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Round-trip: GET should return the same followed_teams
	req2 := httptest.NewRequest("GET", "/user/settings", nil)
	req2 = req2.WithContext(middleware.WithFirebaseUID(req2.Context(), "firebase-abc"))
	rec2 := httptest.NewRecorder()
	h.GetSettings(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", rec2.Code)
	}

	var settings map[string]any
	json.NewDecoder(rec2.Body).Decode(&settings)
	teams, ok := settings["followed_teams"].([]any)
	if !ok {
		t.Fatalf("expected followed_teams array, got %T: %v", settings["followed_teams"], settings["followed_teams"])
	}
	if len(teams) != 2 {
		t.Errorf("expected 2 followed teams, got %d: %v", len(teams), teams)
	}
}

func TestSettingsHandler_UpdateSettings_ClearsFollowedTeams(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	h := handler.NewSettingsHandler(userRepo, userSettingsRepo)

	// First: set some teams
	body1 := `{"location_name":"Toronto","latitude":43.65,"longitude":-79.35,"radius_km":25,"followed_teams":["nhl:tor"]}`
	req1 := httptest.NewRequest("PUT", "/user/settings", bytes.NewBufferString(body1))
	req1 = req1.WithContext(middleware.WithFirebaseUID(req1.Context(), "firebase-clear"))
	h.UpdateSettings(httptest.NewRecorder(), req1)

	// Second: clear teams by sending empty array
	body2 := `{"location_name":"Toronto","latitude":43.65,"longitude":-79.35,"radius_km":25,"followed_teams":[]}`
	req2 := httptest.NewRequest("PUT", "/user/settings", bytes.NewBufferString(body2))
	req2 = req2.WithContext(middleware.WithFirebaseUID(req2.Context(), "firebase-clear"))
	rec2 := httptest.NewRecorder()
	h.UpdateSettings(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}

	req3 := httptest.NewRequest("GET", "/user/settings", nil)
	req3 = req3.WithContext(middleware.WithFirebaseUID(req3.Context(), "firebase-clear"))
	rec3 := httptest.NewRecorder()
	h.GetSettings(rec3, req3)

	var settings map[string]any
	json.NewDecoder(rec3.Body).Decode(&settings)
	if _, exists := settings["followed_teams"]; exists {
		t.Errorf("expected followed_teams to be absent after clearing, got %v", settings["followed_teams"])
	}
}
