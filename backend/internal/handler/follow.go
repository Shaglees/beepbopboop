package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type FollowHandler struct {
	userRepo   *repository.UserRepo
	followRepo *repository.FollowRepo
}

func NewFollowHandler(userRepo *repository.UserRepo, followRepo *repository.FollowRepo) *FollowHandler {
	return &FollowHandler{
		userRepo:   userRepo,
		followRepo: followRepo,
	}
}

// Follow handles POST /agents/{agentID}/follow — follow an agent.
func (h *FollowHandler) Follow(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	agentID := chi.URLParam(r, "agentID")
	if agentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing agent ID"})
		return
	}

	count, err := h.followRepo.Follow(user.ID, agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to follow agent"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"following":      true,
		"follower_count": count,
	})
}

// Unfollow handles DELETE /agents/{agentID}/follow — unfollow an agent.
func (h *FollowHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	agentID := chi.URLParam(r, "agentID")
	if agentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing agent ID"})
		return
	}

	count, err := h.followRepo.Unfollow(user.ID, agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to unfollow agent"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"following":      false,
		"follower_count": count,
	})
}

// GetAgentProfile handles GET /agents/{agentID} — public agent profile with follow state.
func (h *FollowHandler) GetAgentProfile(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	if agentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing agent ID"})
		return
	}

	profile, err := h.followRepo.GetAgentProfile(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load agent profile"})
		return
	}
	if profile == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
		return
	}

	// If the request is authenticated, populate is_following.
	uid := middleware.FirebaseUIDFromContext(r.Context())
	if uid != "" {
		user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
		if err == nil {
			following, _ := h.followRepo.IsFollowing(user.ID, agentID)
			profile.IsFollowing = following
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// ListFollowing handles GET /agents/following — agents the current user follows.
func (h *FollowHandler) ListFollowing(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	agents, err := h.followRepo.ListFollowedAgents(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load followed agents"})
		return
	}

	// Mark all as followed and return an empty array instead of null.
	if agents == nil {
		agents = make([]model.AgentProfile, 0)
	}
	for i := range agents {
		agents[i].IsFollowing = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}
