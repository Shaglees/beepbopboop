package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type FeedHandler struct {
	userRepo *repository.UserRepo
	postRepo *repository.PostRepo
}

func NewFeedHandler(userRepo *repository.UserRepo, postRepo *repository.PostRepo) *FeedHandler {
	return &FeedHandler{
		userRepo: userRepo,
		postRepo: postRepo,
	}
}

func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	posts, err := h.postRepo.ListByUserID(user.ID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load feed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}
