package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestAgentAuth_ValidToken(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("fb-123")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "Test Agent")

	tokenRepo := repository.NewTokenRepo(db)
	rawToken, _ := tokenRepo.Create(agent.ID)

	handler := middleware.AgentAuth(tokenRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentID := middleware.AgentIDFromContext(r.Context())
		if agentID != agent.ID {
			t.Errorf("expected agent ID %s, got %s", agent.ID, agentID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/posts", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAgentAuth_MissingToken(t *testing.T) {
	db := database.OpenTestDB(t)

	tokenRepo := repository.NewTokenRepo(db)

	handler := middleware.AgentAuth(tokenRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/posts", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAgentAuth_InvalidToken(t *testing.T) {
	db := database.OpenTestDB(t)

	tokenRepo := repository.NewTokenRepo(db)

	handler := middleware.AgentAuth(tokenRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/posts", nil)
	req.Header.Set("Authorization", "Bearer bbp_invalidtoken")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
