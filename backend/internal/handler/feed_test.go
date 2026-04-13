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

func TestFeedHandler_EmptyFeed(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	postRepo := repository.NewPostRepo(db)
	h := handler.NewFeedHandler(userRepo, postRepo)

	req := httptest.NewRequest("GET", "/feed", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.GetFeed(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var posts []model.Post
	json.NewDecoder(rec.Body).Decode(&posts)
	if len(posts) != 0 {
		t.Errorf("expected empty feed, got %d posts", len(posts))
	}
}

func TestFeedHandler_WithPosts(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")
	postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID,
		Title: "Test Post", Body: "Test body",
	})

	h := handler.NewFeedHandler(userRepo, postRepo)

	req := httptest.NewRequest("GET", "/feed", nil)
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abc"))
	rec := httptest.NewRecorder()

	h.GetFeed(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var posts []model.Post
	json.NewDecoder(rec.Body).Decode(&posts)
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Title != "Test Post" {
		t.Errorf("expected title Test Post, got %s", posts[0].Title)
	}
	if posts[0].AgentName != "My Agent" {
		t.Errorf("expected agent name My Agent, got %s", posts[0].AgentName)
	}
}
