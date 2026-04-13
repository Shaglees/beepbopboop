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

func TestPostHandler_CreatePost_DefaultPostType(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "A nice park", "body": "Great park nearby."}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["post_type"] != "discovery" {
		t.Errorf("expected post_type discovery, got %v", resp["post_type"])
	}
}

func TestPostHandler_CreatePost_InvalidPostType(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Test", "body": "Test body", "post_type": "bogus"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), "some-agent"))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostHandler_CreatePost_ArticleType(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "New AI breakthrough", "body": "A major advance in reasoning.", "post_type": "article", "locality": "Anthropic Blog"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["post_type"] != "article" {
		t.Errorf("expected post_type article, got %v", resp["post_type"])
	}
}

func TestPostHandler_CreatePost_VideoType(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "WebGPU explainer", "body": "A 12-minute deep dive.", "post_type": "video", "locality": "Fireship on YouTube"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["post_type"] != "video" {
		t.Errorf("expected post_type video, got %v", resp["post_type"])
	}
}

func TestPostHandler_DefaultVisibility(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Test post", "body": "Test body"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["visibility"] != "public" {
		t.Errorf("expected visibility public, got %v", resp["visibility"])
	}
}

func TestPostHandler_PersonalVisibility(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Personal post", "body": "Family stuff", "visibility": "personal"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["visibility"] != "personal" {
		t.Errorf("expected visibility personal, got %v", resp["visibility"])
	}
}

func TestPostHandler_PrivateVisibility(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Private post", "body": "Calendar event", "visibility": "private"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["visibility"] != "private" {
		t.Errorf("expected visibility private, got %v", resp["visibility"])
	}
}

func TestPostHandler_InvalidVisibility(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Test", "body": "Test body", "visibility": "secret"}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), "some-agent"))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostHandler_LabelsRoundTrip(t *testing.T) {
	db, _ := database.Open(":memory:")
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	h := handler.NewPostHandler(agentRepo, postRepo)

	body := `{"title": "Labeled post", "body": "Post with labels", "labels": ["coffee", "cafe", "specialty-coffee"]}`
	req := httptest.NewRequest("POST", "/posts", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	h.CreatePost(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	labels, ok := resp["labels"].([]any)
	if !ok {
		t.Fatalf("expected labels array, got %T: %v", resp["labels"], resp["labels"])
	}
	if len(labels) != 3 {
		t.Errorf("expected 3 labels, got %d", len(labels))
	}
	if labels[0] != "coffee" || labels[1] != "cafe" || labels[2] != "specialty-coffee" {
		t.Errorf("unexpected labels: %v", labels)
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
