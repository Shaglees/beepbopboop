package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestPushTokenRepo_Upsert(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-token-upsert")

	if err := pushTokenRepo.Upsert(user.ID, "device-token-xyz", "apns"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second upsert with same token should succeed (idempotent)
	if err := pushTokenRepo.Upsert(user.ID, "device-token-xyz", "apns"); err != nil {
		t.Fatalf("duplicate upsert failed: %v", err)
	}
}

func TestPushTokenRepo_TopUnseenPosts_Empty(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-digest-empty")

	posts, err := pushTokenRepo.TopUnseenPosts(user.ID, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}

func TestPushTokenRepo_TopUnseenPosts_FiltersViewed(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-digest-viewed")
	agent, _ := agentRepo.Create(user.ID, "Digest Agent")

	p1, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID,
		Title: "Unseen Post", Body: "You have not seen this",
	})
	p2, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID,
		Title: "Seen Post", Body: "You already saw this",
	})

	eventRepo.Create(p2.ID, user.ID, "view", nil)

	posts, err := pushTokenRepo.TopUnseenPosts(user.ID, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 unseen post, got %d", len(posts))
	}
	if posts[0].ID != p1.ID {
		t.Errorf("expected post %s, got %s", p1.ID, posts[0].ID)
	}
}

func TestPushTokenRepo_TopUnseenPosts_Limit(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-digest-limit")
	agent, _ := agentRepo.Create(user.ID, "Limit Agent")

	for i := 0; i < 5; i++ {
		postRepo.Create(repository.CreatePostParams{
			AgentID: agent.ID, UserID: user.ID,
			Title: "Post", Body: "Body",
		})
	}

	posts, err := pushTokenRepo.TopUnseenPosts(user.ID, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) > 3 {
		t.Errorf("expected at most 3 posts, got %d", len(posts))
	}
}

func TestPushTokenRepo_TopUnseenPosts_RankedByEngagement(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	eventRepo := repository.NewEventRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)

	viewer, _ := userRepo.FindOrCreateByFirebaseUID("firebase-digest-rank-viewer")
	author, _ := userRepo.FindOrCreateByFirebaseUID("firebase-digest-rank-author")
	agent, _ := agentRepo.Create(author.ID, "Rank Agent")

	popular, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: author.ID,
		Title: "Popular Post", Body: "Many saves",
	})
	quiet, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: author.ID,
		Title: "Quiet Post", Body: "No engagement",
	})

	// popular gets 3 saves from the author (viewer hasn't seen either)
	for i := 0; i < 3; i++ {
		eventRepo.Create(popular.ID, author.ID, "save", nil)
	}
	_ = quiet

	posts, err := pushTokenRepo.TopUnseenPosts(viewer.ID, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) < 1 {
		t.Fatalf("expected at least 1 post, got 0")
	}
	if posts[0].ID != popular.ID {
		t.Errorf("expected popular post first, got %s", posts[0].ID)
	}
}
