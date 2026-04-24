package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type ProfileHandler struct {
	userRepo      *repository.UserRepo
	agentRepo     *repository.AgentRepo
	interestRepo  *repository.UserInterestRepo
	lifestyleRepo *repository.UserLifestyleRepo
	prefsRepo     *repository.UserContentPrefsRepo
}

func NewProfileHandler(
	userRepo *repository.UserRepo,
	agentRepo *repository.AgentRepo,
	interestRepo *repository.UserInterestRepo,
	lifestyleRepo *repository.UserLifestyleRepo,
	prefsRepo *repository.UserContentPrefsRepo,
) *ProfileHandler {
	return &ProfileHandler{
		userRepo:      userRepo,
		agentRepo:     agentRepo,
		interestRepo:  interestRepo,
		lifestyleRepo: lifestyleRepo,
		prefsRepo:     prefsRepo,
	}
}

func (h *ProfileHandler) buildProfile(userID string, includeInactive bool) (*model.UserProfile, error) {
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	var interests []model.UserInterest
	if includeInactive {
		interests, err = h.interestRepo.ListAll(userID)
	} else {
		interests, err = h.interestRepo.ListActive(userID)
	}
	if err != nil {
		return nil, err
	}
	if interests == nil {
		interests = []model.UserInterest{}
	}

	lifestyle, err := h.lifestyleRepo.List(userID)
	if err != nil {
		return nil, err
	}
	if lifestyle == nil {
		lifestyle = []model.LifestyleTag{}
	}

	prefs, err := h.prefsRepo.List(userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		prefs = []model.ContentPref{}
	}

	hasUserInterest := false
	for _, i := range interests {
		if i.Source == "user" {
			hasUserInterest = true
			break
		}
	}

	return &model.UserProfile{
		Identity: model.UserProfileIdentity{
			DisplayName:  user.DisplayName,
			AvatarURL:    user.AvatarURL,
			Timezone:     user.Timezone,
			HomeLocation: user.HomeLocation,
			HomeLat:      user.HomeLat,
			HomeLon:      user.HomeLon,
		},
		Interests:          interests,
		Lifestyle:          lifestyle,
		ContentPrefs:       prefs,
		ProfileInitialized: user.DisplayName != "" && hasUserInterest,
	}, nil
}

// GetProfileFirebase handles GET /user/profile (Firebase auth).
func (h *ProfileHandler) GetProfileFirebase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	includeInactive := r.URL.Query().Get("include_inactive") == "true"
	profile, err := h.buildProfile(user.ID, includeInactive)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build profile"})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// GetProfileAgent handles GET /user/profile (Agent auth).
func (h *ProfileHandler) GetProfileAgent(w http.ResponseWriter, r *http.Request) {
	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}

	profile, err := h.buildProfile(agent.UserID, false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build profile"})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// UpdateProfileFirebase handles PUT /user/profile (Firebase auth).
func (h *ProfileHandler) UpdateProfileFirebase(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req struct {
		DisplayName  string   `json:"display_name"`
		AvatarURL    string   `json:"avatar_url"`
		Timezone     string   `json:"timezone"`
		HomeLocation string   `json:"home_location"`
		HomeLat      *float64 `json:"home_lat"`
		HomeLon      *float64 `json:"home_lon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.DisplayName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "display_name is required"})
		return
	}
	if req.Timezone != "" && !strings.HasPrefix(req.Timezone, "UTC") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "timezone must be UTC offset (e.g. UTC-7)"})
		return
	}

	err = h.userRepo.UpdateProfile(user.ID, req.DisplayName, req.AvatarURL, req.Timezone, req.HomeLocation, req.HomeLat, req.HomeLon)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update profile"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SetInterests handles PUT /user/interests (Firebase auth).
func (h *ProfileHandler) SetInterests(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req struct {
		Interests []model.UserInterest `json:"interests"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	err = h.interestRepo.BulkSetUser(user.ID, req.Interests)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to set interests"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// PromoteInterest handles POST /user/interests/{id}/promote.
func (h *ProfileHandler) PromoteInterest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.interestRepo.Promote(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to promote interest"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DismissInterest handles POST /user/interests/{id}/dismiss.
func (h *ProfileHandler) DismissInterest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.interestRepo.Dismiss(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to dismiss interest"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// PauseInterest handles POST /user/interests/{id}/pause.
func (h *ProfileHandler) PauseInterest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Days int `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Days <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "days must be a positive integer"})
		return
	}
	if err := h.interestRepo.Pause(id, req.Days); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to pause interest"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SetLifestyle handles PUT /user/lifestyle (Firebase auth).
func (h *ProfileHandler) SetLifestyle(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req struct {
		Tags []model.LifestyleTag `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.lifestyleRepo.BulkSet(user.ID, req.Tags); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to set lifestyle tags"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SetContentPrefs handles PUT /user/content-prefs (Firebase auth).
func (h *ProfileHandler) SetContentPrefs(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req struct {
		Prefs []model.ContentPref `json:"prefs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.prefsRepo.BulkSet(user.ID, req.Prefs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to set content prefs"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
