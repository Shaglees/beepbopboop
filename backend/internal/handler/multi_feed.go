package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type MultiFeedHandler struct {
	userRepo         *repository.UserRepo
	postRepo         *repository.PostRepo
	userSettingsRepo *repository.UserSettingsRepo
	weightsRepo      *repository.WeightsRepo
	eventRepo        *repository.EventRepo
}

func NewMultiFeedHandler(userRepo *repository.UserRepo, postRepo *repository.PostRepo, userSettingsRepo *repository.UserSettingsRepo, weightsRepo *repository.WeightsRepo, eventRepo *repository.EventRepo) *MultiFeedHandler {
	return &MultiFeedHandler{
		userRepo:         userRepo,
		postRepo:         postRepo,
		userSettingsRepo: userSettingsRepo,
		weightsRepo:      weightsRepo,
		eventRepo:        eventRepo,
	}
}

func (h *MultiFeedHandler) GetPersonal(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	cursor, limit, err := parsePagination(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
		return
	}

	posts, nextCursor, err := h.postRepo.ListPersonal(user.ID, cursor, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load feed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.FeedResponse{Posts: posts, NextCursor: nextCursor})
}

func (h *MultiFeedHandler) GetCommunity(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	settings, err := h.userSettingsRepo.Get(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}
	if settings == nil || settings.Latitude == nil || settings.Longitude == nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "location_required"})
		return
	}

	cursor, limit, err := parsePagination(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
		return
	}

	posts, nextCursor, err := h.postRepo.ListCommunity(*settings.Latitude, *settings.Longitude, settings.RadiusKm, cursor, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load feed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.FeedResponse{Posts: posts, NextCursor: nextCursor})
}

func (h *MultiFeedHandler) GetForYou(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	settings, err := h.userSettingsRepo.Get(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}
	if settings == nil || settings.Latitude == nil || settings.Longitude == nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "location_required"})
		return
	}

	cursor, limit, err := parsePagination(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_cursor"})
		return
	}

	// Sensible defaults for new users or when engagement data is sparse.
	defaultWeights := &repository.FeedWeights{
		FreshnessBias: 0.8,
		GeoBias:       0.3,
		LabelWeights: map[string]float64{
			"fashion":     0.4,
			"sports":      0.4,
			"trending":    0.3,
			"hacker-news": 0.3,
			"outfit":      0.3,
			"event":       0.2,
			"discovery":   0.2,
			"article":     0.1,
		},
		TypeWeights: map[string]float64{
			"event":     0.3,
			"discovery": 0.2,
			"article":   0.1,
			"video":     0.2,
		},
	}

	// Compute dynamic weights from user engagement (cached for 1 hour).
	feedWeights, err := h.weightsRepo.GetOrCompute(user.ID, h.eventRepo, defaultWeights)
	if err != nil {
		slog.Warn("failed to compute user weights, using defaults", "error", err)
		feedWeights = defaultWeights
	}

	posts, nextCursor, err := h.postRepo.ListForYou(user.ID, *settings.Latitude, *settings.Longitude, settings.RadiusKm, cursor, limit, feedWeights)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load feed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.FeedResponse{Posts: posts, NextCursor: nextCursor})
}
