package handler

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
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
//   - POST /skills/user           (firebase-auth, iOS)
//   - GET  /skills/user/manifest  (agent-auth,    openclaw)
//   - GET  /skills/user/files/... (agent-auth,    openclaw)
//
// See docs/user-skills-protocol.md.
type UserSkillHandler struct {
	userRepo   *repository.UserRepo
	agentRepo  *repository.AgentRepo
	skillRepo  *repository.UserSkillRepo
	spreadRepo *repository.SpreadRepo
}

func NewUserSkillHandler(
	userRepo *repository.UserRepo,
	agentRepo *repository.AgentRepo,
	skillRepo *repository.UserSkillRepo,
	spreadRepo *repository.SpreadRepo,
) *UserSkillHandler {
	return &UserSkillHandler{
		userRepo:   userRepo,
		agentRepo:  agentRepo,
		skillRepo:  skillRepo,
		spreadRepo: spreadRepo,
	}
}

// Submit handles POST /skills/user. Authenticated as a Firebase user (iOS
// app). The skill is built synchronously by the stub builder and stored
// before the response returns. The returned status reflects the stub's
// synchronous nature; the real builder will move this back to "queued".
func (h *UserSkillHandler) Submit(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req model.CreateUserSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	count, err := h.skillRepo.CountByUser(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to check skill count"})
		return
	}
	if count >= UserSkillsPerUserCap {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "user skill cap reached"})
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

	skill, err := h.skillRepo.Upsert(
		user.ID,
		result.SkillName,
		kind,
		req.Extends,
		req.Intent,
		req.Hints,
		result.Files,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to persist skill"})
		return
	}

	// Standalone skills get their own slot in the user's spread when the
	// caller provided a weight (iOS slider). Extensions modify a shipped
	// vertical that already exists, so they leave the spread alone. A zero
	// weight means "leave the spread alone" — the user can manage it later
	// via the existing PUT /settings/spread. Update failures are logged but
	// non-fatal: the skill is still installed and usable.
	if kind == model.UserSkillKindStandalone && req.Weight > 0 {
		if err := h.spreadRepo.UpsertVertical(user.ID, skill.Name, req.Weight); err != nil {
			log.Printf("warning: spread update failed for user=%s skill=%s: %v", user.ID, skill.Name, err)
		}
	}

	writeJSON(w, http.StatusAccepted, model.CreateUserSkillResponse{
		SkillName:   skill.Name,
		Status:      skill.Status,
		SubmittedAt: skill.UpdatedAt,
	})
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

	file, err := h.skillRepo.GetFile(agent.UserID, skillName, path)
	if err != nil {
		if errors.Is(err, repository.ErrUserSkillNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load file"})
		return
	}

	etag := `"` + hex.EncodeToString([]byte(file.SHA256)) + `"`
	w.Header().Set("ETag", `"`+file.SHA256+`"`)
	if match := r.Header.Get("If-None-Match"); match != "" && (match == file.SHA256 || match == `"`+file.SHA256+`"` || match == etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Content)
}
