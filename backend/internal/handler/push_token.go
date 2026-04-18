package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type PushTokenHandler struct {
	userRepo      *repository.UserRepo
	pushTokenRepo *repository.PushTokenRepo
}

func NewPushTokenHandler(userRepo *repository.UserRepo, pushTokenRepo *repository.PushTokenRepo) *PushTokenHandler {
	return &PushTokenHandler{
		userRepo:      userRepo,
		pushTokenRepo: pushTokenRepo,
	}
}

type registerPushTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

func (h *PushTokenHandler) RegisterPushToken(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	var req registerPushTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token is required"})
		return
	}

	platform := req.Platform
	if platform == "" {
		platform = "apns"
	}

	if err := h.pushTokenRepo.Upsert(user.ID, req.Token, platform); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store token"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
