package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type WeightsHandler struct {
	agentRepo   *repository.AgentRepo
	userRepo    *repository.UserRepo
	weightsRepo *repository.WeightsRepo
}

func NewWeightsHandler(agentRepo *repository.AgentRepo, userRepo *repository.UserRepo, weightsRepo *repository.WeightsRepo) *WeightsHandler {
	return &WeightsHandler{
		agentRepo:   agentRepo,
		userRepo:    userRepo,
		weightsRepo: weightsRepo,
	}
}

// GetWeights returns the current user weights (agent-auth).
func (h *WeightsHandler) GetWeights(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	weights, err := h.weightsRepo.Get(agent.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load weights"})
		return
	}

	if weights == nil {
		writeJSON(w, http.StatusOK, map[string]any{"user_id": agent.UserID, "weights": nil})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weights)
}

type updateWeightsRequest struct {
	Weights json.RawMessage `json:"weights"`
}

// UpdateWeights sets user preference weights (agent-auth, pushed by Lobs).
func (h *WeightsHandler) UpdateWeights(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	var req updateWeightsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Weights) == 0 || !json.Valid(req.Weights) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "weights must be valid JSON"})
		return
	}

	weights, err := h.weightsRepo.Upsert(agent.UserID, req.Weights)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save weights"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weights)
}

// GetWeightsFirebase returns the current user weights (Firebase-auth, mobile client).
func (h *WeightsHandler) GetWeightsFirebase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	weights, err := h.weightsRepo.Get(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load weights"})
		return
	}

	if weights == nil {
		writeJSON(w, http.StatusOK, map[string]any{"user_id": user.ID, "weights": nil})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weights)
}

// UpdateWeightsFirebase sets user preference weights (Firebase-auth, mobile client).
// Accepts flat FeedWeights JSON: {"freshness_bias": 0.8, "geo_bias": 0.5, ...}
func (h *WeightsHandler) UpdateWeightsFirebase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(raw) == 0 || !json.Valid(raw) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "weights must be valid JSON"})
		return
	}

	weights, err := h.weightsRepo.Upsert(user.ID, raw)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save weights"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weights)
}
