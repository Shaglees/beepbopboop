package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/ranking"
)

// MLAdminHandler exposes model versioning endpoints for agents and operators.
type MLAdminHandler struct {
	versionRepo *ranking.ModelVersionRepo
	gate        *ranking.DeploymentGate
}

func NewMLAdminHandler(versionRepo *ranking.ModelVersionRepo, gate *ranking.DeploymentGate) *MLAdminHandler {
	return &MLAdminHandler{versionRepo: versionRepo, gate: gate}
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

// DeployVersion manually deploys a model version using an atomic gate check
// to prevent TOCTOU races between concurrent deploy requests.
// Route: POST /admin/ml/models/{id}/deploy (agent-auth)
func (h *MLAdminHandler) DeployVersion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid model version id"})
		return
	}

	// Verify the candidate exists before attempting the gate.
	candidate, err := h.versionRepo.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load model version"})
		return
	}
	if candidate == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "model version not found"})
		return
	}

	// MarkDeployedWithGate performs the AUC check and deployment atomically.
	if err := h.versionRepo.MarkDeployedWithGate(r.Context(), id, h.gate.MinImprovement()); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "deployment blocked: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deployed", "version": candidate.Version})
}
