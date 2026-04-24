package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/ab"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// ExperimentsHandler serves A/B experiment endpoints.
type ExperimentsHandler struct {
	assigner *ab.Assigner
	userRepo *repository.UserRepo
	expRepo  *repository.ExperimentRepo
}

func NewExperimentsHandler(assigner *ab.Assigner, userRepo *repository.UserRepo, expRepo *repository.ExperimentRepo) *ExperimentsHandler {
	return &ExperimentsHandler{assigner: assigner, userRepo: userRepo, expRepo: expRepo}
}

// GetVariant returns the caller's stable variant for a named experiment.
// Route: GET /experiments/{name}/variant (Firebase-auth)
func (h *ExperimentsHandler) GetVariant(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing experiment name"})
		return
	}

	exp, err := h.expRepo.Get(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load experiment"})
		return
	}
	if exp == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "experiment not found"})
		return
	}

	variant := h.assigner.Variant(r.Context(), user.ID, name, exp.TreatmentPct)
	writeJSON(w, http.StatusOK, map[string]string{"variant": variant, "experiment": name})
}

type createExperimentRequest struct {
	Name         string `json:"name"`
	TreatmentPct int    `json:"treatment_pct"`
}

// CreateExperiment upserts an experiment definition.
// Route: POST /admin/experiments (agent-auth)
func (h *ExperimentsHandler) CreateExperiment(w http.ResponseWriter, r *http.Request) {
	var req createExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.TreatmentPct < 0 || req.TreatmentPct > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "treatment_pct must be 0-100"})
		return
	}

	if err := h.expRepo.Upsert(r.Context(), req.Name, req.TreatmentPct); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create experiment"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"name": req.Name, "status": "running"})
}

// GetResults returns per-variant engagement stats for a named experiment.
// Route: GET /admin/experiments/{name}/results (agent-auth)
func (h *ExperimentsHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing experiment name"})
		return
	}

	results, err := h.expRepo.Results(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load results"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
