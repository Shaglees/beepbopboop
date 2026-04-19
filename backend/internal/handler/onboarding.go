package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type OnboardingHandler struct {
	userRepo    *repository.UserRepo
	prototypes  *embedding.PrototypeStore
	userEmbRepo *repository.UserEmbeddingRepo
}

func NewOnboardingHandler(
	userRepo *repository.UserRepo,
	prototypes *embedding.PrototypeStore,
	userEmbRepo *repository.UserEmbeddingRepo,
) *OnboardingHandler {
	return &OnboardingHandler{
		userRepo:    userRepo,
		prototypes:  prototypes,
		userEmbRepo: userEmbRepo,
	}
}

type submitInterestsRequest struct {
	Interests []string `json:"interests"`
}

// SubmitInterests accepts a list of onboarding interest names, combines their
// prototype vectors, and seeds the user's embedding. A no-op (still 200) when
// no matching prototypes exist.
func (h *OnboardingHandler) SubmitInterests(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())

	var req submitInterestsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	vec := h.prototypes.CombineFor(req.Interests)
	if embedding.IsZero(vec) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "no_prototypes"})
		return
	}

	if err := h.userEmbRepo.Upsert(r.Context(), user.ID, vec, 0); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store embedding"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
