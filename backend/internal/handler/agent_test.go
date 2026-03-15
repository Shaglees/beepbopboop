package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func setupAgentTest(t *testing.T) (*handler.AgentHandler, *repository.UserRepo, *repository.AgentRepo, *repository.TokenRepo) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	tokenRepo := repository.NewTokenRepo(db)
	h := handler.NewAgentHandler(userRepo, agentRepo, tokenRepo)
	return h, userRepo, agentRepo, tokenRepo
}

func TestAgentHandler_CreateAgent(t *testing.T) {
	h, _, _, _ := setupAgentTest(t)

	body := `{"name": "My Agent"}`
	req := httptest.NewRequest("POST", "/agents", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.CreateAgent(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["name"] != "My Agent" {
		t.Errorf("expected name My Agent, got %v", resp["name"])
	}
}

func TestAgentHandler_CreateToken(t *testing.T) {
	h, userRepo, agentRepo, _ := setupAgentTest(t)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	req := httptest.NewRequest("POST", "/agents/"+agent.ID+"/tokens", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agentID", agent.ID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(middleware.WithFirebaseUID(ctx, "firebase-abc"))
	rec := httptest.NewRecorder()

	h.CreateToken(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	token, ok := resp["token"].(string)
	if !ok || token == "" {
		t.Error("expected non-empty token in response")
	}
}
