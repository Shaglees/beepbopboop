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
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestCreatorsHandler_Create(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	creatorRepo := repository.NewLocalCreatorRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-creator-test")
	agent, _ := agentRepo.Create(user.ID, "Local Scout")

	h := handler.NewCreatorsHandler(creatorRepo, userRepo, userSettingsRepo)

	body := `{"name":"Maria Chen","designation":"Painter","bio":"Oil painter.","lat":40.7128,"lon":-74.0060,"area_name":"Brooklyn, NY","source":"Brooklyn Rail"}`
	req := httptest.NewRequest("POST", "/creators", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.LocalCreator
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "Maria Chen" {
		t.Errorf("expected Maria Chen, got %q", resp.Name)
	}
	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreatorsHandler_Create_MissingRequired(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	creatorRepo := repository.NewLocalCreatorRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-creator-test2")
	agent, _ := agentRepo.Create(user.ID, "Local Scout 2")

	h := handler.NewCreatorsHandler(creatorRepo, userRepo, userSettingsRepo)

	body := `{"bio":"missing name and designation"}`
	req := httptest.NewRequest("POST", "/creators", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreatorsHandler_GetNearby(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	creatorRepo := repository.NewLocalCreatorRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-nearby-test")

	lat, lon := 40.7128, -74.0060
	userSettingsRepo.Upsert(user.ID, "Brooklyn, NY", &lat, &lon, 25.0, nil, true, 8, nil)

	creatorRepo.Upsert(model.CreateCreatorRequest{
		Name:        "Maria Chen",
		Designation: "Painter",
		Lat:         &lat,
		Lon:         &lon,
		Source:      "Brooklyn Rail",
	})

	h := handler.NewCreatorsHandler(creatorRepo, userRepo, userSettingsRepo)

	req := httptest.NewRequest("GET", "/creators/nearby", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-nearby-test"))
	rec := httptest.NewRecorder()

	h.GetNearby(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.FeedResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(resp.Posts))
	}
	if resp.Posts[0].Title != "Maria Chen" {
		t.Errorf("expected title Maria Chen, got %q", resp.Posts[0].Title)
	}
	if resp.Posts[0].DisplayHint != "creator_spotlight" {
		t.Errorf("expected creator_spotlight, got %q", resp.Posts[0].DisplayHint)
	}
}

func TestCreatorsHandler_GetNearby_NoLocation(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	creatorRepo := repository.NewLocalCreatorRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-noloc-test")
	_ = user // no location set

	h := handler.NewCreatorsHandler(creatorRepo, userRepo, userSettingsRepo)

	req := httptest.NewRequest("GET", "/creators/nearby", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-noloc-test"))
	rec := httptest.NewRecorder()

	h.GetNearby(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 when no location set, got %d", rec.Code)
	}
}
