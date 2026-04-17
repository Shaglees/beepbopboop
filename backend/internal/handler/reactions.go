package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

var validReactions = map[string]bool{
	"more":       true,
	"less":       true,
	"stale":      true,
	"not_for_me": true,
}

type ReactionsHandler struct {
	userRepo     *repository.UserRepo
	agentRepo    *repository.AgentRepo
	reactionRepo *repository.ReactionRepo
}

func NewReactionsHandler(userRepo *repository.UserRepo, agentRepo *repository.AgentRepo, reactionRepo *repository.ReactionRepo) *ReactionsHandler {
	return &ReactionsHandler{
		userRepo:     userRepo,
		agentRepo:    agentRepo,
		reactionRepo: reactionRepo,
	}
}

// SetReaction upserts a user's reaction on a post (Firebase-auth).
func (h *ReactionsHandler) SetReaction(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	postID := chi.URLParam(r, "postID")
	if postID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing post ID"})
		return
	}

	var req struct {
		Reaction string `json:"reaction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if !validReactions[req.Reaction] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid reaction"})
		return
	}

	reaction, err := h.reactionRepo.Upsert(postID, user.ID, req.Reaction)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to set reaction"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reaction)
}

// RemoveReaction deletes a user's reaction from a post (Firebase-auth).
func (h *ReactionsHandler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	postID := chi.URLParam(r, "postID")
	if postID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing post ID"})
		return
	}

	if err := h.reactionRepo.Delete(postID, user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove reaction"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Summary returns aggregated reaction counts (agent-auth, for agents to read).
func (h *ReactionsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	summary, err := h.reactionRepo.Summary(agent.UserID, 30)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to compute summary"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
