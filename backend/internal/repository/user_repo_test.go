package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestUserRepo_FindOrCreateByFirebaseUID(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewUserRepo(db)

	// First call creates user
	user1, err := repo.FindOrCreateByFirebaseUID("firebase-abc")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if user1.FirebaseUID != "firebase-abc" {
		t.Errorf("expected firebase_uid firebase-abc, got %s", user1.FirebaseUID)
	}
	if user1.ID == "" {
		t.Error("expected non-empty user ID")
	}

	// Second call returns same user
	user2, err := repo.FindOrCreateByFirebaseUID("firebase-abc")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if user1.ID != user2.ID {
		t.Errorf("expected same user ID, got %s and %s", user1.ID, user2.ID)
	}
}
