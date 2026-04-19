package embedding_test

import (
	"fmt"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// seedPost creates a user+agent+post for embedding tests. Returns post ID.
func seedPost(t *testing.T, postRepo *repository.PostRepo, userRepo *repository.UserRepo, agentRepo *repository.AgentRepo, i int) string {
	t.Helper()
	uid := fmt.Sprintf("embed-uid-%d", i)
	user, err := userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	agent, err := agentRepo.Create(user.ID, fmt.Sprintf("Agent %d", i))
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	post, err := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID,
		UserID:  user.ID,
		Title:   fmt.Sprintf("Post %d", i),
		Body:    fmt.Sprintf("Body %d", i),
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	return post.ID
}

func makeVec(dim, hotDim int) []float32 {
	v := make([]float32, dim)
	if hotDim < dim {
		v[hotDim] = 1.0
	}
	return v
}

func TestStoreAndRetrieveEmbedding(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	embRepo := embedding.NewEmbeddingRepo(db)

	postID := seedPost(t, postRepo, userRepo, agentRepo, 1)

	vec := makeVec(1536, 0)
	if err := embRepo.StoreEmbedding(postID, vec); err != nil {
		t.Fatalf("StoreEmbedding: %v", err)
	}

	got, err := embRepo.GetEmbedding(postID)
	if err != nil {
		t.Fatalf("GetEmbedding: %v", err)
	}
	if len(got) != 1536 {
		t.Fatalf("expected 1536 dims, got %d", len(got))
	}
	if got[0] != 1.0 {
		t.Errorf("expected got[0]=1.0, got %f", got[0])
	}
}

func TestStoreEmbedding_Idempotent(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	embRepo := embedding.NewEmbeddingRepo(db)

	postID := seedPost(t, postRepo, userRepo, agentRepo, 2)

	vec1 := makeVec(1536, 0)
	vec2 := makeVec(1536, 1) // different vector

	if err := embRepo.StoreEmbedding(postID, vec1); err != nil {
		t.Fatalf("first StoreEmbedding: %v", err)
	}
	if err := embRepo.StoreEmbedding(postID, vec2); err != nil {
		t.Fatalf("second StoreEmbedding: %v", err)
	}

	// Second write overwrites: check value is vec2
	got, err := embRepo.GetEmbedding(postID)
	if err != nil {
		t.Fatalf("GetEmbedding: %v", err)
	}
	if got[1] != 1.0 {
		t.Errorf("expected second vector to overwrite first, got[1]=%f", got[1])
	}

	// Check no duplicate key errors — row count must be 1 for this post's embedding
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = $1 AND embedding IS NOT NULL", postID)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 row with embedding, got %d", count)
	}
}

func TestGetUnembedded_ReturnsOnlyNullEmbeddings(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	embRepo := embedding.NewEmbeddingRepo(db)

	id1 := seedPost(t, postRepo, userRepo, agentRepo, 10)
	id2 := seedPost(t, postRepo, userRepo, agentRepo, 11)
	id3 := seedPost(t, postRepo, userRepo, agentRepo, 12)

	// Embed posts 1 and 2, leave 3 unembedded
	if err := embRepo.StoreEmbedding(id1, makeVec(1536, 0)); err != nil {
		t.Fatalf("StoreEmbedding id1: %v", err)
	}
	if err := embRepo.StoreEmbedding(id2, makeVec(1536, 1)); err != nil {
		t.Fatalf("StoreEmbedding id2: %v", err)
	}

	posts, err := embRepo.GetUnembedded(10)
	if err != nil {
		t.Fatalf("GetUnembedded: %v", err)
	}

	// Should contain only the unembedded post
	found := false
	for _, p := range posts {
		if p.ID == id3 {
			found = true
		}
		if p.ID == id1 || p.ID == id2 {
			t.Errorf("expected embedded posts to be excluded, but found %s", p.ID)
		}
	}
	if !found {
		t.Errorf("expected unembedded post %s to be returned", id3)
	}
}

func TestFindSimilar_ReturnsCosineNearest(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	embRepo := embedding.NewEmbeddingRepo(db)

	// Create 5 posts each with a unit vector in a different dimension
	ids := make([]string, 5)
	for i := range ids {
		ids[i] = seedPost(t, postRepo, userRepo, agentRepo, 20+i)
		if err := embRepo.StoreEmbedding(ids[i], makeVec(1536, i)); err != nil {
			t.Fatalf("StoreEmbedding[%d]: %v", i, err)
		}
	}

	// Query with vector identical to post[0]'s embedding
	query := makeVec(1536, 0)
	results, err := embRepo.FindSimilar(query, 5)
	if err != nil {
		t.Fatalf("FindSimilar: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	// Post 0 should be first (cosine distance 0 = most similar)
	if results[0].ID != ids[0] {
		t.Errorf("expected post[0] (%s) as nearest, got %s", ids[0], results[0].ID)
	}
}

func TestStoreEmbedding_NonExistentPost_ReturnsError(t *testing.T) {
	db := database.OpenTestDB(t)
	embRepo := embedding.NewEmbeddingRepo(db)

	err := embRepo.StoreEmbedding("does-not-exist", makeVec(1536, 0))
	if err == nil {
		t.Error("expected error when storing embedding for non-existent post, got nil")
	}
}

func TestFindSimilar_ExcludesNonPublishedPosts(t *testing.T) {
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	embRepo := embedding.NewEmbeddingRepo(db)

	// Post A: will be set to non-published status via SQL after creation
	idA := seedPost(t, postRepo, userRepo, agentRepo, 30)
	// Post B: stays published
	idB := seedPost(t, postRepo, userRepo, agentRepo, 31)

	vec := makeVec(1536, 5)
	if err := embRepo.StoreEmbedding(idA, vec); err != nil {
		t.Fatalf("StoreEmbedding A: %v", err)
	}
	if err := embRepo.StoreEmbedding(idB, vec); err != nil {
		t.Fatalf("StoreEmbedding B: %v", err)
	}

	// Mark post A as archived (non-published)
	if _, err := db.Exec(`UPDATE posts SET status = 'archived' WHERE id = $1`, idA); err != nil {
		t.Fatalf("update status: %v", err)
	}

	results, err := embRepo.FindSimilar(vec, 10)
	if err != nil {
		t.Fatalf("FindSimilar: %v", err)
	}

	for _, p := range results {
		if p.ID == idA {
			t.Errorf("FindSimilar returned non-published post %s", idA)
		}
	}
	found := false
	for _, p := range results {
		if p.ID == idB {
			found = true
		}
	}
	if !found {
		t.Errorf("FindSimilar did not return published post %s", idB)
	}
}

func TestFindSimilar_EmptyTableReturnsEmpty(t *testing.T) {
	db := database.OpenTestDB(t)
	embRepo := embedding.NewEmbeddingRepo(db)

	query := makeVec(1536, 0)
	results, err := embRepo.FindSimilar(query, 5)
	if err != nil {
		t.Fatalf("FindSimilar on empty table: %v", err)
	}
	if results == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
