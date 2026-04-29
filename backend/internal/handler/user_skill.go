package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	"github.com/shanegleeson/beepbopboop/backend/internal/skillbuilder"
)

// UserSkillsPerUserCap is the maximum number of skills a single user may
// own. Submissions beyond this return 409. Tunable via the spec.
const UserSkillsPerUserCap = 50

// UserSkillHandler exposes the three endpoints for the user-skills protocol:
//   - POST /skills/user           (firebase-auth or agent-auth)
//   - GET  /skills/user/manifest  (agent-auth,    openclaw)
//   - GET  /skills/user/files/... (agent-auth,    openclaw)
//
// See docs/user-skills-protocol.md.
type UserSkillHandler struct {
	userRepo  *repository.UserRepo
	agentRepo *repository.AgentRepo
	skillRepo *repository.UserSkillRepo
}

func NewUserSkillHandler(
	userRepo *repository.UserRepo,
	agentRepo *repository.AgentRepo,
	skillRepo *repository.UserSkillRepo,
) *UserSkillHandler {
	return &UserSkillHandler{
		userRepo:  userRepo,
		agentRepo: agentRepo,
		skillRepo: skillRepo,
	}
}

// Submit handles POST /skills/user. Authenticated as either a Firebase user
// (iOS app) or an agent token. The skill is built synchronously by the stub
// builder and stored before the response returns. The returned status reflects
// the stub's synchronous nature; the real builder will move this back to
// "queued".
func (h *UserSkillHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, err := h.userIDForSubmit(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req model.CreateUserSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result, err := skillbuilder.Build(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	kind := req.Kind
	if kind == "" {
		kind = model.UserSkillKindStandalone
	}
	extends := req.Extends
	if kind == model.UserSkillKindExtension {
		extends = result.SkillName
	}

	skill, err := h.skillRepo.UpsertWithCap(
		userID,
		result.SkillName,
		kind,
		extends,
		req.Intent,
		req.Hints,
		result.Files,
		UserSkillsPerUserCap,
	)
	if err != nil {
		if errors.Is(err, repository.ErrUserSkillCapReached) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "user skill cap reached"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist skill"})
		return
	}

	writeJSON(w, http.StatusAccepted, model.CreateUserSkillResponse{
		SkillName:   skill.Name,
		Status:      skill.Status,
		SubmittedAt: skill.UpdatedAt,
	})
}

func (h *UserSkillHandler) userIDForSubmit(r *http.Request) (string, error) {
	if uid := middleware.FirebaseUIDFromContext(r.Context()); uid != "" {
		user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
		if err != nil {
			return "", err
		}
		return user.ID, nil
	}
	if agentID := middleware.AgentIDFromContext(r.Context()); agentID != "" {
		agent, err := h.agentRepo.GetByID(agentID)
		if err != nil {
			return "", err
		}
		return agent.UserID, nil
	}
	return "", errors.New("missing authenticated user")
}

// Manifest handles GET /skills/user/manifest. Authenticated as an agent
// (openclaw). The agent token is resolved to a user_id and the manifest is
// scoped to that user.
func (h *UserSkillHandler) Manifest(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	skills, err := h.skillRepo.Manifest(agent.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load manifest"})
		return
	}
	if skills == nil {
		skills = []model.UserSkillManifestEntry{}
	}

	writeJSON(w, http.StatusOK, model.UserSkillManifest{
		UserID: agent.UserID,
		Skills: skills,
	})
}

// GetFile handles GET /skills/user/files/{name}/*. Authenticated as an
// agent. Returns the raw file body with an ETag matching the stored
// sha256 so openclaw can short-circuit unchanged downloads.
func (h *UserSkillHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	skillName := chi.URLParam(r, "name")
	path := chi.URLParam(r, "*")
	if skillName == "" || path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "skill name and path are required"})
		return
	}
	if err := repository.ValidateUserSkillName(skillName); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid skill name"})
		return
	}
	if err := repository.ValidateUserSkillFilePath(path); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid file path"})
		return
	}

	file, err := h.skillRepo.GetFile(agent.UserID, skillName, path)
	if err != nil {
		if errors.Is(err, repository.ErrUserSkillNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load file"})
		return
	}

	w.Header().Set("ETag", `"`+file.SHA256+`"`)
	if match := r.Header.Get("If-None-Match"); match != "" && (match == file.SHA256 || match == `"`+file.SHA256+`"`) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Content)
}
