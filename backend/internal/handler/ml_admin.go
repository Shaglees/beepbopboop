package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/ranking"
)

// MLAdminHandler exposes model versioning endpoints for agents and operators.
type MLAdminHandler struct {
	versionRepo     *ranking.ModelVersionRepo
	gate            *ranking.DeploymentGate
	operatorAgentID string // only this agent may deploy; empty disables the check
}

func NewMLAdminHandler(versionRepo *ranking.ModelVersionRepo, gate *ranking.DeploymentGate, operatorAgentID string) *MLAdminHandler {
	return &MLAdminHandler{versionRepo: versionRepo, gate: gate, operatorAgentID: operatorAgentID}
}

// ListVersions returns all model versions sorted newest-first.
// Route: GET /admin/ml/versions (agent-auth)
func (h *MLAdminHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	versions, err := h.versionRepo.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list model versions"})
		return
	}
	if versions == nil {
		versions = []model.ModelVersion{}
	}
	writeJSON(w, http.StatusOK, versions)
}

// DeployVersion manually deploys a model version after checking the AUC gate.
// Route: POST /admin/ml/models/{id}/deploy (agent-auth, operator only)
func (h *MLAdminHandler) DeployVersion(w http.ResponseWriter, r *http.Request) {
	if h.operatorAgentID != "" {
		caller := middleware.AgentIDFromContext(r.Context())
		if caller != h.operatorAgentID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only the operator agent may deploy models"})
			return
		}
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid model version id"})
		return
	}

	candidate, err := h.versionRepo.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load model version"})
		return
	}
	if candidate == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "model version not found"})
		return
	}

	active, err := h.versionRepo.GetActive(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load active version"})
		return
	}

	var currentAUC float64
	if active != nil {
		currentAUC = active.AUCROC
	}

	if !h.gate.ShouldDeploy(currentAUC, candidate.AUCROC) {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "deployment blocked: AUC improvement below threshold",
		})
		return
	}

	if err := h.versionRepo.MarkDeployed(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to deploy model version"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deployed", "version": candidate.Version})
}
