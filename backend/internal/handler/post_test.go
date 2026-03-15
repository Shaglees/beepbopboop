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

func TestPostHandler_CreatePost(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Tennis courts nearby", "body": "A park near you has tennis courts."}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["title"] != "Tennis courts nearby" {
		t.Errorf("expected title, got %v", resp["title"])
	}
	if resp["agent_name"] != "My Agent" {
		t.Errorf("expected agent_name My Agent, got %v", resp["agent_name"])
	}
}

func TestPostHandler_MissingTitle(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"body": "no title"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), "some-agent"))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
