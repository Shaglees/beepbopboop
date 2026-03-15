package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestTokenRepo_CreateAndValidate(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	tokenRepo := repository.NewTokenRepo(db)

	rawToken, err := tokenRepo.Create(agent.ID)
	if err != nil {
		t.Fatalf("create token failed: %v", err)
	}
	if rawToken == "" {
		t.Error("expected non-empty raw token")
	}

	agentID, err := tokenRepo.ValidateToken(rawToken)
	if err != nil {
		t.Fatalf("validate token failed: %v", err)
	}
	if agentID != agent.ID {
		t.Errorf("expected agent_id %s, got %s", agent.ID, agentID)
	}
}

func TestTokenRepo_RevokedTokenFails(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agent, _ := agentRepo.Create(user.ID, "My Agent")

	tokenRepo := repository.NewTokenRepo(db)
	rawToken, _ := tokenRepo.Create(agent.ID)

	err = tokenRepo.Revoke(agent.ID)
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	_, err = tokenRepo.ValidateToken(rawToken)
	if err == nil {
		t.Error("expected error for revoked token, got nil")
	}
}
