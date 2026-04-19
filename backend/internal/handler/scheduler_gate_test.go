package handler_test

// Test D: PublishScheduled must set created_at = scheduled_at.
// This guards against regressions where the scheduler publishes posts but
// loses the intended publish timestamp (making them appear at the wrong
// position in time-ordered feeds).

import (
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestPublishScheduled_UsesScheduledAt(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)

	owner, _ := userRepo.FindOrCreateByFirebaseUID("firebase-sched-test")
	agent, _ := agentRepo.Create(owner.ID, "Scheduler Gate Agent")

	// Create a post scheduled 5 minutes in the future (repo sets status=scheduled).
	futureTime := time.Now().UTC().Add(5 * time.Minute)
	post, err := postRepo.Create(repository.CreatePostParams{
		AgentID:     agent.ID,
		UserID:      owner.ID,
		Title:       "Scheduled Post",
		Body:        "Publish me later",
		Visibility:  "public",
		ScheduledAt: &futureTime,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if post.Status != "scheduled" {
		t.Fatalf("expected status=scheduled, got %q", post.Status)
	}

	// Backdate scheduled_at so PublishScheduled picks it up immediately.
	pastTime := time.Now().UTC().Add(-1 * time.Hour)
	if _, err := db.Exec("UPDATE posts SET scheduled_at = $1 WHERE id = $2", pastTime, post.ID); err != nil {
		t.Fatalf("backdate scheduled_at: %v", err)
	}

	n, err := postRepo.PublishScheduled()
	if err != nil {
		t.Fatalf("PublishScheduled: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 post published, got %d", n)
	}

	// Verify status transitioned and created_at was set to scheduled_at.
	var status string
	var createdAt, scheduledAt time.Time
	err = db.QueryRow(
		"SELECT status, created_at, scheduled_at FROM posts WHERE id = $1",
		post.ID,
	).Scan(&status, &createdAt, &scheduledAt)
	if err != nil {
		t.Fatalf("query post: %v", err)
	}
	if status != "published" {
		t.Errorf("expected status=published after PublishScheduled, got %q", status)
	}
	// PublishScheduled sets created_at = scheduled_at so the post sorts correctly in feeds.
	if !createdAt.Equal(scheduledAt) {
		t.Errorf("expected created_at == scheduled_at after publishing\n  created_at:   %s\n  scheduled_at: %s", createdAt, scheduledAt)
	}
}
