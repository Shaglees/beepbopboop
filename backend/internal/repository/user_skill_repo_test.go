package repository_test

import (
	"errors"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func setupUserSkillRepo(t *testing.T) (*repository.UserSkillRepo, string) {
	t.Helper()
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-skills-test")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return repository.NewUserSkillRepo(db), user.ID
}

func TestUserSkillRepo_UpsertAndGetByName(t *testing.T) {
	repo, userID := setupUserSkillRepo(t)

	files := []repository.FileInput{
		{Path: "SKILL.md", Content: []byte("---\nname: foo\n---\nbody\n")},
		{Path: "MODE_brief.md", Content: []byte("brief\n")},
	}
	skill, err := repo.Upsert(userID, "foo", model.UserSkillKindStandalone, "", "make a foo skill", 14, nil, files)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if skill.Version != 1 {
		t.Errorf("expected version 1, got %d", skill.Version)
	}
	if skill.Status != model.UserSkillStatusReady {
		t.Errorf("expected status ready, got %s", skill.Status)
	}

	again, err := repo.Upsert(userID, "foo", model.UserSkillKindStandalone, "", "make a foo skill v2", 30, nil, files[:1])
	if err != nil {
		t.Fatalf("upsert again: %v", err)
	}
	if again.Version != 2 {
		t.Errorf("expected version 2 after re-upsert, got %d", again.Version)
	}
	if again.Intent != "make a foo skill v2" {
		t.Errorf("intent should update, got %q", again.Intent)
	}
}

func TestUserSkillRepo_Manifest_OmitsForeignUsers(t *testing.T) {
	repo, userID := setupUserSkillRepo(t)
	db := database.OpenTestDB(t)
	otherUser, _ := repository.NewUserRepo(db).FindOrCreateByFirebaseUID("other-user")
	otherRepo := repository.NewUserSkillRepo(db)

	_, err := repo.Upsert(userID, "mine", model.UserSkillKindStandalone, "", "x", 7, nil,
		[]repository.FileInput{{Path: "SKILL.md", Content: []byte("mine")}})
	if err != nil {
		t.Fatalf("upsert mine: %v", err)
	}
	_, err = otherRepo.Upsert(otherUser.ID, "theirs", model.UserSkillKindStandalone, "", "y", 7, nil,
		[]repository.FileInput{{Path: "SKILL.md", Content: []byte("theirs")}})
	if err != nil {
		t.Fatalf("upsert theirs: %v", err)
	}

	manifest, err := repo.Manifest(userID)
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if len(manifest) != 1 || manifest[0].Name != "mine" {
		t.Fatalf("manifest should only include caller's skills, got %+v", manifest)
	}
	if len(manifest[0].Files) != 1 || manifest[0].Files[0].Path != "SKILL.md" {
		t.Errorf("file metadata missing: %+v", manifest[0].Files)
	}
	if manifest[0].Files[0].SHA256 == "" || manifest[0].Files[0].Size == 0 {
		t.Errorf("expected sha256 and size populated: %+v", manifest[0].Files[0])
	}
}

func TestUserSkillRepo_GetFile(t *testing.T) {
	repo, userID := setupUserSkillRepo(t)

	content := []byte("# preferences\n- avoid paywalls\n")
	_, err := repo.Upsert(userID, "beepbopboop-local-news", model.UserSkillKindExtension,
		"beepbopboop-local-news", "avoid paywalls", 7, nil,
		[]repository.FileInput{{Path: "preferences.md", Content: content}})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	file, err := repo.GetFile(userID, "beepbopboop-local-news", "preferences.md")
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	if string(file.Content) != string(content) {
		t.Errorf("content mismatch: got %q", string(file.Content))
	}
	if file.SHA256 == "" || file.Size != len(content) {
		t.Errorf("sha256/size missing: %+v", file)
	}

	_, err = repo.GetFile(userID, "beepbopboop-local-news", "missing.md")
	if !errors.Is(err, repository.ErrUserSkillNotFound) {
		t.Errorf("expected ErrUserSkillNotFound, got %v", err)
	}
	_, err = repo.GetFile("nope", "beepbopboop-local-news", "preferences.md")
	if !errors.Is(err, repository.ErrUserSkillNotFound) {
		t.Errorf("expected ErrUserSkillNotFound for foreign user, got %v", err)
	}
}

func TestUserSkillRepo_CountByUser(t *testing.T) {
	repo, userID := setupUserSkillRepo(t)

	if n, _ := repo.CountByUser(userID); n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
	_, err := repo.Upsert(userID, "a", model.UserSkillKindStandalone, "", "i", 7, nil,
		[]repository.FileInput{{Path: "SKILL.md", Content: []byte("a")}})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if n, _ := repo.CountByUser(userID); n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
}
