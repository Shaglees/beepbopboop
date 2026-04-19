package repository_test

// Test F: concurrent reactions must produce a correct denormalized reaction_count.
// Verifies that syncReactionCountTx is transactional and race-free when multiple
// users react to the same post simultaneously.

import (
	"fmt"
	"sync"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestReactionCountTransactional(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	reactionRepo := repository.NewReactionRepo(db)

	owner, _ := userRepo.FindOrCreateByFirebaseUID("firebase-rc-owner")
	agent, _ := agentRepo.Create(owner.ID, "Reaction Count Agent")

	post, err := postRepo.Create(repository.CreatePostParams{
		AgentID:    agent.ID,
		UserID:     owner.ID,
		Title:      "Concurrent Reaction Test",
		Body:       "Testing concurrent reactions",
		Visibility: "public",
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	const reactorCount = 10
	var wg sync.WaitGroup
	for i := 0; i < reactorCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u, err := userRepo.FindOrCreateByFirebaseUID(fmt.Sprintf("firebase-reactor-%d", i))
			if err != nil {
				t.Errorf("reactor %d: find user: %v", i, err)
				return
			}
			if _, err := reactionRepo.Upsert(post.ID, u.ID, "more"); err != nil {
				t.Errorf("reactor %d: upsert reaction: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	var count int
	if err := db.QueryRow("SELECT reaction_count FROM posts WHERE id = $1", post.ID).Scan(&count); err != nil {
		t.Fatalf("query reaction_count: %v", err)
	}
	if count != reactorCount {
		t.Errorf("expected reaction_count=%d after %d concurrent 'more' reactions, got %d", reactorCount, reactorCount, count)
	}

	// Removing one reaction must decrement the count atomically.
	firstUser, _ := userRepo.FindOrCreateByFirebaseUID("firebase-reactor-0")
	if err := reactionRepo.Delete(post.ID, firstUser.ID); err != nil {
		t.Fatalf("delete reaction: %v", err)
	}
	if err := db.QueryRow("SELECT reaction_count FROM posts WHERE id = $1", post.ID).Scan(&count); err != nil {
		t.Fatalf("query reaction_count after delete: %v", err)
	}
	if count != reactorCount-1 {
		t.Errorf("expected reaction_count=%d after delete, got %d", reactorCount-1, count)
	}
}
