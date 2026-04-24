package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// TestBatchEvents_RecordsAbVariant verifies that POST /events/batch with
// ab_variant set stores the value on the post_events row.
func TestBatchEvents_RecordsAbVariant(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	h := handler.NewEventsHandler(userRepo, agentRepo, eventRepo)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abvariant-user")
	agent, _ := agentRepo.Create(user.ID, "ab-variant-agent")
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "AB variant post", Body: "body",
	})

	body := map[string]any{"events": []map[string]any{
		{"post_id": post.ID, "event_type": "view", "dwell_ms": 3000, "ab_variant": "treatment"},
	}}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/events/batch", bytes.NewReader(b))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abvariant-user"))
	rec := httptest.NewRecorder()

	h.BatchTrack(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var stored sql.NullString
	db.QueryRow(
		"SELECT ab_variant FROM post_events WHERE post_id=$1 AND user_id=$2 AND event_type='view'",
		post.ID, user.ID,
	).Scan(&stored)

	if !stored.Valid || stored.String != "treatment" {
		t.Errorf("expected ab_variant='treatment', got %q (valid=%v)", stored.String, stored.Valid)
	}
}

// TestBatchEvents_NilAbVariantWhenOmitted verifies that omitting ab_variant
// stores NULL (not an empty string) in the database.
func TestBatchEvents_NilAbVariantWhenOmitted(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	h := handler.NewEventsHandler(userRepo, agentRepo, eventRepo)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abnil-user")
	agent, _ := agentRepo.Create(user.ID, "ab-nil-agent")
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "No variant post", Body: "body",
	})

	body := map[string]any{"events": []map[string]any{
		{"post_id": post.ID, "event_type": "view"},
	}}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/events/batch", bytes.NewReader(b))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "firebase-abnil-user"))
	rec := httptest.NewRecorder()

	h.BatchTrack(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var stored sql.NullString
	db.QueryRow(
		"SELECT ab_variant FROM post_events WHERE post_id=$1 AND user_id=$2",
		post.ID, user.ID,
	).Scan(&stored)

	if stored.Valid {
		t.Errorf("expected NULL ab_variant when omitted, got %q", stored.String)
	}
}
