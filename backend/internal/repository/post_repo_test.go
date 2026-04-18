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
	if found.ViewCount != 0 || found.SaveCount != 0 || found.ReactionCount != 0 {
		t.Errorf("new post should have zero engagement counts, got view=%d save=%d reaction=%d",
			found.ViewCount, found.SaveCount, found.ReactionCount)
	}
}

func TestPostRepo_ReactionCountMaintained(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user1, _ := userRepo.FindOrCreateByFirebaseUID("firebase-rxn-1")
	user2, _ := userRepo.FindOrCreateByFirebaseUID("firebase-rxn-2")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user1.ID, "Agent")

	postRepo := repository.NewPostRepo(db)
	lat, lon := 53.35, -6.26
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user1.ID,
		Title: "Post", Body: "body", Latitude: &lat, Longitude: &lon, Visibility: "public",
	})

	reactionRepo := repository.NewReactionRepo(db)

	// Two "more" reactions
	if _, err := reactionRepo.Upsert(post.ID, user1.ID, "more"); err != nil {
		t.Fatalf("upsert user1 more: %v", err)
	}
	if _, err := reactionRepo.Upsert(post.ID, user2.ID, "more"); err != nil {
		t.Fatalf("upsert user2 more: %v", err)
	}

	posts, _, _ := postRepo.ListCommunity(lat, lon, 10.0, "", 20)
	if posts[0].ReactionCount != 2 {
		t.Errorf("expected reaction_count 2, got %d", posts[0].ReactionCount)
	}

	// One "less" reaction replaces user1's "more" — count drops to 1
	if _, err := reactionRepo.Upsert(post.ID, user1.ID, "less"); err != nil {
		t.Fatalf("upsert user1 less: %v", err)
	}
	posts, _, _ = postRepo.ListCommunity(lat, lon, 10.0, "", 20)
	if posts[0].ReactionCount != 1 {
		t.Errorf("expected reaction_count 1 after downgrade, got %d", posts[0].ReactionCount)
	}

	// Delete user2's reaction — count drops to 0
	if err := reactionRepo.Delete(post.ID, user2.ID); err != nil {
		t.Fatalf("delete user2 reaction: %v", err)
	}
	posts, _, _ = postRepo.ListCommunity(lat, lon, 10.0, "", 20)
	if posts[0].ReactionCount != 0 {
		t.Errorf("expected reaction_count 0 after delete, got %d", posts[0].ReactionCount)
	}
}

func TestPostRepo_SaveCountMaintained(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-save-test")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "Agent")

	postRepo := repository.NewPostRepo(db)
	lat, lon := 53.35, -6.26
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID,
		Title: "Post", Body: "body", Latitude: &lat, Longitude: &lon, Visibility: "public",
	})

	eventRepo := repository.NewEventRepo(db)
	if err := eventRepo.BatchCreate(user.ID, []model.EventInput{
		{PostID: post.ID, EventType: "save"},
	}); err != nil {
		t.Fatalf("BatchCreate: %v", err)
	}

	posts, _, _ := postRepo.ListCommunity(lat, lon, 10.0, "", 20)
	if posts[0].SaveCount != 1 {
		t.Errorf("expected save_count 1, got %d", posts[0].SaveCount)
	}
}

func TestPostRepo_SaveCountUnsave(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-unsave-test")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "Agent")

	postRepo := repository.NewPostRepo(db)
	lat, lon := 53.35, -6.26
	post, _ := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID,
		Title: "Post", Body: "body", Latitude: &lat, Longitude: &lon, Visibility: "public",
	})

	eventRepo := repository.NewEventRepo(db)

	// Save then unsave
	if err := eventRepo.BatchCreate(user.ID, []model.EventInput{{PostID: post.ID, EventType: "save"}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	posts, _, _ := postRepo.ListCommunity(lat, lon, 10.0, "", 20)
	if posts[0].SaveCount != 1 {
		t.Errorf("expected save_count 1 after save, got %d", posts[0].SaveCount)
	}

	if err := eventRepo.BatchCreate(user.ID, []model.EventInput{{PostID: post.ID, EventType: "unsave"}}); err != nil {
		t.Fatalf("unsave: %v", err)
	}
	posts, _, _ = postRepo.ListCommunity(lat, lon, 10.0, "", 20)
	if posts[0].SaveCount != 0 {
		t.Errorf("expected save_count 0 after unsave, got %d", posts[0].SaveCount)
	}
}

func TestListCommunity_RankedByScore(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-ranking-test")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "Agent")

	postRepo := repository.NewPostRepo(db)
	lat, lon := 53.35, -6.26

	// NEWER: 1h old, no engagement — chronologically first but scored second
	newer, err := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "Newer no engagement", Body: "body",
		Latitude: &lat, Longitude: &lon, Visibility: "public",
	})
	if err != nil {
		t.Fatalf("create newer: %v", err)
	}
	if _, err := db.Exec(`UPDATE posts SET created_at = NOW() - INTERVAL '1 hour' WHERE id = $1`, newer.ID); err != nil {
		t.Fatalf("backdate newer: %v", err)
	}

	// OLDER_ENGAGED: 3h old, save_count=12 — chronologically second but scored first
	olderEngaged, err := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "Older with engagement", Body: "body",
		Latitude: &lat, Longitude: &lon, Visibility: "public",
	})
	if err != nil {
		t.Fatalf("create olderEngaged: %v", err)
	}
	if _, err := db.Exec(`UPDATE posts SET created_at = NOW() - INTERVAL '3 hours', save_count = 12 WHERE id = $1`, olderEngaged.ID); err != nil {
		t.Fatalf("backdate+save olderEngaged: %v", err)
	}

	// STALE: 24h old, no engagement — last in both orders
	stale, err := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID, UserID: user.ID, Title: "Stale post", Body: "body",
		Latitude: &lat, Longitude: &lon, Visibility: "public",
	})
	if err != nil {
		t.Fatalf("create stale: %v", err)
	}
	if _, err := db.Exec(`UPDATE posts SET created_at = NOW() - INTERVAL '24 hours' WHERE id = $1`, stale.ID); err != nil {
		t.Fatalf("backdate stale: %v", err)
	}

	posts, _, err := postRepo.ListCommunity(lat, lon, 10.0, "", 20)
	if err != nil {
		t.Fatalf("ListCommunity: %v", err)
	}
	if len(posts) < 3 {
		t.Fatalf("expected 3 posts, got %d", len(posts))
	}

	// OLDER_ENGAGED should rank first (engagement lifts it above the 1h-newer post)
	if posts[0].ID != olderEngaged.ID {
		t.Errorf("expected older+engaged post first (engagement > recency diff), got %q", posts[0].Title)
	}
	// STALE should rank last (24h old)
	if posts[len(posts)-1].ID != stale.ID {
		t.Errorf("expected stale post last, got %q", posts[len(posts)-1].Title)
	}
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
