package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/embedding"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type OnboardingHandler struct {
	userRepo     *repository.UserRepo
	prototypes   *embedding.PrototypeStore
	userEmbRepo  *repository.UserEmbeddingRepo
	interestRepo *repository.UserInterestRepo
}

func NewOnboardingHandler(
	userRepo *repository.UserRepo,
	prototypes *embedding.PrototypeStore,
	userEmbRepo *repository.UserEmbeddingRepo,
	interestRepo *repository.UserInterestRepo,
) *OnboardingHandler {
	return &OnboardingHandler{
		userRepo:     userRepo,
		prototypes:   prototypes,
		userEmbRepo:  userEmbRepo,
		interestRepo: interestRepo,
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

	// Write plaintext interests for profile display and skill access
	var userInterests []model.UserInterest
	for _, name := range req.Interests {
		userInterests = append(userInterests, model.UserInterest{
			Category:   name,
			Topic:      name,
			Source:     "user",
			Confidence: 1.0,
		})
	}
	if err := h.interestRepo.BulkSetUser(user.ID, userInterests); err != nil {
		slog.Warn("failed to write plaintext interests", "error", err)
		// Non-fatal — embedding was already written
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
