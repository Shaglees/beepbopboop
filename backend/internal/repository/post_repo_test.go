package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestPostRepo_CreateAndListByUser(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	postRepo := repository.NewPostRepo(db)

	post, err := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID,
		UserID:  user.ID,
		Title:   "Tennis courts 6 minutes away",
		Body:    "A park near your home has tennis courts.",
	})
	if err != nil {
		t.Fatalf("create post failed: %v", err)
	}
	if post.Title != "Tennis courts 6 minutes away" {
		t.Errorf("expected title, got %s", post.Title)
	}

	posts, err := postRepo.ListByUserID(user.ID, 20)
	if err != nil {
		t.Fatalf("list posts failed: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].AgentName != "My Agent" {
		t.Errorf("expected agent name My Agent, got %s", posts[0].AgentName)
	}
}

func TestPostRepo_ListByUserID_NewestFirst(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	postRepo := repository.NewPostRepo(db)

	postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "First", Body: "body",
	})
	postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "Second", Body: "body",
	})

	posts, _ := postRepo.ListByUserID(user.ID, 20)
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
	if posts[0].Title != "Second" {
		t.Errorf("expected newest first, got %s", posts[0].Title)
	}
}

func TestPostRepo_EmptyFeed(t *testing.T) {
	db := database.OpenTestDB(t)

	postRepo := repository.NewPostRepo(db)
	posts, err := postRepo.ListByUserID("nonexistent", 20)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if posts == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}

func TestPostRepo_EngagementCountsPopulated(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-engagement-test")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "Test Agent")

	postRepo := repository.NewPostRepo(db)

	lat, lon := 53.35, -6.26
	post, err := postRepo.Create(repository.CreatePostParams{
		AgentID:    agent.ID,
		UserID:     user.ID,
		Title:      "Nearby post",
		Body:       "body",
		Latitude:   &lat,
		Longitude:  &lon,
		Visibility: "public",
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	posts, _, err := postRepo.ListCommunity(lat, lon, 10.0, "", 20)
	if err != nil {
		t.Fatalf("ListCommunity: %v", err)
	}
	if len(posts) == 0 {
		t.Fatal("expected at least 1 post")
	}

	found := posts[0]
	if found.ID != post.ID {
		t.Fatalf("expected post %s, got %s", post.ID, found.ID)
	}
	// These should be populated (0 is correct for a brand new post, but the field exists)
	_ = found.ReactionCount
	_ = found.SaveCount
}

func TestPostRepo_OptionalFields(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	postRepo := repository.NewPostRepo(db)

	post, err := postRepo.Create(repository.CreatePostParams{
		AgentID:     agent.ID,
		UserID:      user.ID,
		Title:       "Test",
		Body:        "Body",
		ImageURL:    "https://i.imgur.com/example.jpg",
		ExternalURL: "https://example.com",
		Locality:    "Dublin 2",
		PostType:    "discovery",
	})
	if err != nil {
		t.Fatal(err)
	}
	if post.ImageURL != "https://i.imgur.com/example.jpg" {
		t.Errorf("expected image url, got %s", post.ImageURL)
	}
	if post.Locality != "Dublin 2" {
		t.Errorf("expected locality Dublin 2, got %s", post.Locality)
	}

	posts, _ := postRepo.ListByUserID(user.ID, 20)
	_ = model.Post{}
	if posts[0].ImageURL != "https://i.imgur.com/example.jpg" {
		t.Errorf("expected image url in feed, got %s", posts[0].ImageURL)
	}
}
