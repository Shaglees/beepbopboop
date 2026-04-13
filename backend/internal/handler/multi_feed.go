package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type MultiFeedHandler struct {
	userRepo         *repository.UserRepo
	postRepo         *repository.PostRepo
	userSettingsRepo *repository.UserSettingsRepo
}

func NewMultiFeedHandler(userRepo *repository.UserRepo, postRepo *repository.PostRepo, userSettingsRepo *repository.UserSettingsRepo) *MultiFeedHandler {
	return &MultiFeedHandler{
		userRepo:         userRepo,
		postRepo:         postRepo,
		userSettingsRepo: userSettingsRepo,
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

	posts, nextCursor, err := h.postRepo.ListForYou(user.ID, *settings.Latitude, *settings.Longitude, settings.RadiusKm, cursor, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load feed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.FeedResponse{Posts: posts, NextCursor: nextCursor})
}
