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

func TestEventsHandler_ImpressionEventTypeAccepted(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	h := handler.NewEventsHandler(userRepo, agentRepo, eventRepo)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-impression-user")
	agent, _ := agentRepo.Create(user.ID, "Test Agent")
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "Impression post", Body: "body",
	})

	body := map[string]any{"events": []map[string]any{
		{"post_id": post.ID, "event_type": "impression"},
	}}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/events/batch", bytes.NewReader(b))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-impression-user"))
	rec := httptest.NewRecorder()

	h.BatchTrack(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for impression event, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestEventsHandler_UnknownEventTypeRejected(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	h := handler.NewEventsHandler(userRepo, agentRepo, eventRepo)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-bad-event-user")
	agent, _ := agentRepo.Create(user.ID, "Test Agent")
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "Post", Body: "body",
	})

	body := map[string]any{"events": []map[string]any{
		{"post_id": post.ID, "event_type": "explode"},
	}}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/events/batch", bytes.NewReader(b))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-bad-event-user"))
	rec := httptest.NewRecorder()

	h.BatchTrack(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown event type, got %d", rec.Code)
	}
}
