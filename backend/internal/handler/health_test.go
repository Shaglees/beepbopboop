package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestHealthHandler(t *testing.T) {
	h := handler.NewHealthHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestMeHandler(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	h := handler.NewMeHandler(userRepo)

	req := httptest.NewRequest("GET", "/me", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["firebase_uid"] != "firebase-abc" {
		t.Errorf("expected firebase_uid firebase-abc, got %v", resp["firebase_uid"])
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty user id")
	}
}
