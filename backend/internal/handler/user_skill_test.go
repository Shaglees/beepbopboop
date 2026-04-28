package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type userSkillTestEnv struct {
	h         *handler.UserSkillHandler
	userRepo  *repository.UserRepo
	agentRepo *repository.AgentRepo
	skillRepo *repository.UserSkillRepo
}

func setupUserSkillHandler(t *testing.T) userSkillTestEnv {
	t.Helper()
	db := database.OpenTestDB(t)
	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	skillRepo := repository.NewUserSkillRepo(db)
	h := handler.NewUserSkillHandler(userRepo, agentRepo, skillRepo)
	return userSkillTestEnv{h: h, userRepo: userRepo, agentRepo: agentRepo, skillRepo: skillRepo}
}

func TestUserSkillHandler_Submit_Standalone(t *testing.T) {
	env := setupUserSkillHandler(t)

	body := `{"intent": "local high school football for Springfield, IL"}`
	req := httptest.NewRequest("POST", "/skills/user", bytes.NewBufferString(body))
	req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "fb-submit-1"))
	rec := httptest.NewRecorder()

	env.h.Submit(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp model.CreateUserSkillResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.SkillName == "" {
		t.Error("skill_name should be populated")
	}
	if resp.Status != model.UserSkillStatusReady {
		t.Errorf("expected ready, got %s", resp.Status)
	}

	// Skill should be persisted and visible to repo lookups.
	user, _ := env.userRepo.FindOrCreateByFirebaseUID("fb-submit-1")
	skill, err := env.skillRepo.GetByName(user.ID, resp.SkillName)
	if err != nil {
		t.Fatalf("skill should be persisted: %v", err)
	}
	if skill.Kind != model.UserSkillKindStandalone {
		t.Errorf("expected standalone kind, got %s", skill.Kind)
	}
}

func TestUserSkillHandler_Submit_BadRequest(t *testing.T) {
	env := setupUserSkillHandler(t)

	cases := []struct {
		name string
		body string
		code int
	}{
		{"empty intent", `{"intent": ""}`, http.StatusBadRequest},
		{"extension without extends", `{"intent":"x","kind":"extension"}`, http.StatusBadRequest},
		{"invalid json", `not json`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/skills/user", bytes.NewBufferString(tc.body))
			req = req.WithContext(middleware.WithFirebaseUID(req.Context(), "fb-bad"))
			rec := httptest.NewRecorder()
			env.h.Submit(rec, req)
			if rec.Code != tc.code {
				t.Errorf("expected %d, got %d: %s", tc.code, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestUserSkillHandler_Manifest(t *testing.T) {
	env := setupUserSkillHandler(t)
	user, _ := env.userRepo.FindOrCreateByFirebaseUID("fb-manifest")
	agent, _ := env.agentRepo.Create(user.ID, "openclaw")
	_, err := env.skillRepo.Upsert(user.ID, "my-skill", model.UserSkillKindStandalone, "", "intent", nil,
		[]repository.FileInput{{Path: "SKILL.md", Content: []byte("---\nname: my-skill\n---\n")}})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest("GET", "/skills/user/manifest", nil)
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()

	env.h.Manifest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp model.UserSkillManifest
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.UserID != user.ID {
		t.Errorf("user_id mismatch: got %s want %s", resp.UserID, user.ID)
	}
	if len(resp.Skills) != 1 || resp.Skills[0].Name != "my-skill" {
		t.Fatalf("expected one skill, got %+v", resp.Skills)
	}
	if len(resp.Skills[0].Files) != 1 || resp.Skills[0].Files[0].SHA256 == "" {
		t.Errorf("file metadata missing: %+v", resp.Skills[0].Files)
	}
}

func TestUserSkillHandler_GetFile(t *testing.T) {
	env := setupUserSkillHandler(t)
	user, _ := env.userRepo.FindOrCreateByFirebaseUID("fb-getfile")
	agent, _ := env.agentRepo.Create(user.ID, "openclaw")

	body := []byte("# preferences\n- avoid paywalls\n")
	_, err := env.skillRepo.Upsert(user.ID, "beepbopboop-local-news", model.UserSkillKindExtension,
		"beepbopboop-local-news", "avoid paywalls", nil,
		[]repository.FileInput{{Path: "preferences.md", Content: body}})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := newFileRequest(agent.ID, "beepbopboop-local-news", "preferences.md")
	rec := httptest.NewRecorder()
	env.h.GetFile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Equal(rec.Body.Bytes(), body) {
		t.Errorf("body mismatch: got %q", rec.Body.String())
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header missing")
	}

	// If-None-Match short-circuits with 304.
	req2 := newFileRequest(agent.ID, "beepbopboop-local-news", "preferences.md")
	req2.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	env.h.GetFile(rec2, req2)
	if rec2.Code != http.StatusNotModified {
		t.Errorf("expected 304 on cached fetch, got %d", rec2.Code)
	}

	// 404 for missing file.
	req3 := newFileRequest(agent.ID, "beepbopboop-local-news", "no-such-file.md")
	rec3 := httptest.NewRecorder()
	env.h.GetFile(rec3, req3)
	if rec3.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec3.Code)
	}
}

func TestUserSkillHandler_GetFile_ForeignUserDenied(t *testing.T) {
	env := setupUserSkillHandler(t)
	owner, _ := env.userRepo.FindOrCreateByFirebaseUID("fb-owner")
	intruder, _ := env.userRepo.FindOrCreateByFirebaseUID("fb-intruder")
	intruderAgent, _ := env.agentRepo.Create(intruder.ID, "intruder")

	_, err := env.skillRepo.Upsert(owner.ID, "private", model.UserSkillKindStandalone, "", "x", nil,
		[]repository.FileInput{{Path: "SKILL.md", Content: []byte("secret")}})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := newFileRequest(intruderAgent.ID, "private", "SKILL.md")
	rec := httptest.NewRecorder()
	env.h.GetFile(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("intruder must not see other user's files (404), got %d", rec.Code)
	}
}

func newFileRequest(agentID, name, path string) *http.Request {
	req := httptest.NewRequest("GET", "/skills/user/files/"+name+"/"+path, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", name)
	rctx.URLParams.Add("*", path)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(middleware.WithAgentID(ctx, agentID))
}
