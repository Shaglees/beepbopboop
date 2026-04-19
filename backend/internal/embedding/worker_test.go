package embedding_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestBackfillWorker_ProcessesUnembeddedInBatches(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	embRepo := embedding.NewEmbeddingRepo(db)

	// Create a single user+agent to own all 250 posts
	user, err := userRepo.FindOrCreateByFirebaseUID("backfill-worker-uid")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	agent, err := agentRepo.Create(user.ID, "Backfill Agent")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	const total = 250
	for i := 0; i < total; i++ {
		_, err := postRepo.Create(repository.CreatePostParams{
			AgentID: agent.ID,
			UserID:  user.ID,
			Title:   fmt.Sprintf("Backfill Post %d", i),
			Body:    fmt.Sprintf("Body %d", i),
		})
		if err != nil {
			t.Fatalf("create post %d: %v", i, err)
		}
	}

	mock := &mockEmbedder{}
	worker := embedding.NewBackfillWorker(embRepo, mock, 100)

	if err := worker.Run(context.Background()); err != nil {
		t.Fatalf("backfill worker: %v", err)
	}

	// 250 posts at batchSize=100 → 3 calls (100+100+50)
	if mock.calls != 3 {
		t.Errorf("expected 3 EmbedBatch calls, got %d", mock.calls)
	}

	// All posts should now have embeddings
	remaining, err := embRepo.GetUnembedded(total + 1)
	if err != nil {
		t.Fatalf("GetUnembedded after backfill: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 unembedded posts after backfill, got %d", len(remaining))
	}
}
