package handler

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type PhotoHandler struct {
	userRepo  *repository.UserRepo
	photoRepo *repository.UserPhotoRepo
	agentRepo *repository.AgentRepo
}

func NewPhotoHandler(userRepo *repository.UserRepo, photoRepo *repository.UserPhotoRepo, agentRepo *repository.AgentRepo) *PhotoHandler {
	return &PhotoHandler{
		userRepo:  userRepo,
		photoRepo: photoRepo,
		agentRepo: agentRepo,
	}
}

// UploadHeadshot handles PUT /user/photos/headshot (Firebase auth).
func (h *PhotoHandler) UploadHeadshot(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, "headshot")
}

// UploadBodyshot handles PUT /user/photos/bodyshot (Firebase auth).
func (h *PhotoHandler) UploadBodyshot(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, "bodyshot")
}

func (h *PhotoHandler) upload(w http.ResponseWriter, r *http.Request, photoType string) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	file, _, err := r.FormFile("photo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing or invalid photo file"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read photo data"})
		return
	}

	switch photoType {
	case "headshot":
		err = h.photoRepo.SaveHeadshot(user.ID, data, "image/jpeg")
	case "bodyshot":
		err = h.photoRepo.SaveBodyshot(user.ID, data, "image/jpeg")
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save photo"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetHeadshot handles GET /user/photos/headshot (Firebase or Agent auth).
func (h *PhotoHandler) GetHeadshot(w http.ResponseWriter, r *http.Request) {
	h.getPhoto(w, r, "headshot")
}

// GetBodyshot handles GET /user/photos/bodyshot (Firebase or Agent auth).
func (h *PhotoHandler) GetBodyshot(w http.ResponseWriter, r *http.Request) {
	h.getPhoto(w, r, "bodyshot")
}

func (h *PhotoHandler) getPhoto(w http.ResponseWriter, r *http.Request, photoType string) {
	var userID string

	// Try Firebase auth first
	uid := middleware.FirebaseUIDFromContext(r.Context())
	if uid != "" {
		user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
			return
		}
		userID = user.ID
	} else {
		// Fall back to agent auth
		agentID := middleware.AgentIDFromContext(r.Context())
		if agentID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		agent, err := h.agentRepo.GetByID(agentID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
			return
		}
		userID = agent.UserID
	}

	var data []byte
	var contentType string
	var err error

	switch photoType {
	case "headshot":
		data, contentType, err = h.photoRepo.GetHeadshot(userID)
	case "bodyshot":
		data, contentType, err = h.photoRepo.GetBodyshot(userID)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get photo"})
		return
	}

	if data == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no photo"})
		return
	}

	if contentType == "" {
		contentType = "image/jpeg"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// DeletePhoto handles DELETE /user/photos/{type} (Firebase auth).
func (h *PhotoHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	photoType := chi.URLParam(r, "type")
	if photoType != "headshot" && photoType != "bodyshot" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid photo type, must be headshot or bodyshot"})
		return
	}

	if err := h.photoRepo.DeletePhoto(user.ID, photoType); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete photo"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
