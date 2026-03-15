package repository_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestAgentRepo_Create(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-abc")
	if err != nil {
		t.Fatal(err)
	}

	agentRepo := repository.NewAgentRepo(db)
	agent, err := agentRepo.Create(user.ID, "My Agent")
	if err != nil {
		t.Fatalf("create agent failed: %v", err)
	}
	if agent.Name != "My Agent" {
		t.Errorf("expected name My Agent, got %s", agent.Name)
	}
	if agent.UserID != user.ID {
		t.Errorf("expected user_id %s, got %s", user.ID, agent.UserID)
	}
	if agent.Status != "active" {
		t.Errorf("expected status active, got %s", agent.Status)
	}
}

func TestAgentRepo_GetByID(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	created, _ := agentRepo.Create(user.ID, "My Agent")

	found, err := agentRepo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get agent failed: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, found.ID)
	}
}

func TestAgentRepo_ListByUserID(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-abc")

	agentRepo := repository.NewAgentRepo(db)
	agentRepo.Create(user.ID, "Agent 1")
	agentRepo.Create(user.ID, "Agent 2")

	agents, err := agentRepo.ListByUserID(user.ID)
	if err != nil {
		t.Fatalf("list agents failed: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}
