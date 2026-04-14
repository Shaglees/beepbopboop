package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type TemplatesHandler struct {
	userRepo     *repository.UserRepo
	agentRepo    *repository.AgentRepo
	templateRepo *repository.TemplateRepo
}

func NewTemplatesHandler(userRepo *repository.UserRepo, agentRepo *repository.AgentRepo, templateRepo *repository.TemplateRepo) *TemplatesHandler {
	return &TemplatesHandler{
		userRepo:     userRepo,
		agentRepo:    agentRepo,
		templateRepo: templateRepo,
	}
}

// ListTemplatesFirebase returns templates for the Firebase-authenticated user (iOS app).
func (h *TemplatesHandler) ListTemplatesFirebase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	templates, err := h.templateRepo.ListByUserID(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load templates"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// ListTemplatesAgent returns templates for the agent-authenticated user.
func (h *TemplatesHandler) ListTemplatesAgent(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	templates, err := h.templateRepo.ListByUserID(agent.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load templates"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

type upsertTemplateRequest struct {
	Definition json.RawMessage `json:"definition"`
}

// UpsertTemplate creates or updates a custom display template (agent-auth).
func (h *TemplatesHandler) UpsertTemplate(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	hintName := chi.URLParam(r, "hint")
	if hintName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing hint name"})
		return
	}

	var req upsertTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Definition) == 0 || !json.Valid(req.Definition) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "definition must be valid JSON"})
		return
	}

	tmpl, err := h.templateRepo.Upsert(agent.UserID, hintName, req.Definition)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save template"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tmpl)
}

// DeleteTemplate removes a custom display template (agent-auth).
func (h *TemplatesHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	hintName := chi.URLParam(r, "hint")
	if hintName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing hint name"})
		return
	}

	if err := h.templateRepo.Delete(agent.UserID, hintName); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete template"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
