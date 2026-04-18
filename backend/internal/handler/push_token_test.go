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

func TestPushTokenHandler_Register(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)
	h := handler.NewPushTokenHandler(userRepo, pushTokenRepo)

	body := `{"token": "abc123devicetoken", "platform": "apns"}`
	req := httptest.NewRequest("PUT", "/user/push-token", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-push-test"))
	rec := httptest.NewRecorder()

	h.RegisterPushToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
}

func TestPushTokenHandler_Register_DefaultPlatform(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)
	h := handler.NewPushTokenHandler(userRepo, pushTokenRepo)

	body := `{"token": "abc123devicetoken"}`
	req := httptest.NewRequest("PUT", "/user/push-token", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-push-test2"))
	rec := httptest.NewRecorder()

	h.RegisterPushToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPushTokenHandler_Register_Idempotent(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)
	h := handler.NewPushTokenHandler(userRepo, pushTokenRepo)

	for i := 0; i < 2; i++ {
		body := `{"token": "same-token", "platform": "apns"}`
		req := httptest.NewRequest("PUT", "/user/push-token", bytes.NewBufferString(body))
		req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-push-idem"))
		rec := httptest.NewRecorder()
		h.RegisterPushToken(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("call %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestPushTokenHandler_Register_MissingToken(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)
	h := handler.NewPushTokenHandler(userRepo, pushTokenRepo)

	body := `{"platform": "apns"}`
	req := httptest.NewRequest("PUT", "/user/push-token", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-push-test3"))
	rec := httptest.NewRecorder()

	h.RegisterPushToken(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
