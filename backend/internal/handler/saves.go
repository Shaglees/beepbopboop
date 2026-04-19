package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type SavesHandler struct {
	userRepo *repository.UserRepo
	saveRepo *repository.SaveRepo
}

func NewSavesHandler(userRepo *repository.UserRepo, saveRepo *repository.SaveRepo) *SavesHandler {
	return &SavesHandler{
		userRepo: userRepo,
		saveRepo: saveRepo,
	}
}

// SavePost records a save for the authenticated user on the given post.
func (h *SavesHandler) SavePost(w http.ResponseWriter, r *http.Request) {
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

	if err := h.saveRepo.Save(postID, user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save post"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"saved": true})
}

// UnsavePost records an unsave for the authenticated user on the given post.
func (h *SavesHandler) UnsavePost(w http.ResponseWriter, r *http.Request) {
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

	if err := h.saveRepo.Unsave(postID, user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to unsave post"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
