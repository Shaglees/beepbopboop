package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// userEmbeddingGetter loads user vectors for ForYou ML blending (issue #44).
// Implemented by *repository.UserEmbeddingRepo and *repository.EmbeddingCache.
type userEmbeddingGetter interface {
	Get(ctx context.Context, userID string) (*model.UserEmbedding, error)
}

type MultiFeedHandler struct {
	userRepo         *repository.UserRepo
	postRepo         *repository.PostRepo
	userSettingsRepo *repository.UserSettingsRepo
	weightsRepo      *repository.WeightsRepo
	eventRepo        *repository.EventRepo
	reactionRepo     *repository.ReactionRepo
	followRepo       *repository.FollowRepo
	userEmb          userEmbeddingGetter
}

func NewMultiFeedHandler(userRepo *repository.UserRepo, postRepo *repository.PostRepo, userSettingsRepo *repository.UserSettingsRepo, weightsRepo *repository.WeightsRepo, eventRepo *repository.EventRepo, reactionRepo *repository.ReactionRepo, followRepo *repository.FollowRepo, userEmb userEmbeddingGetter) *MultiFeedHandler {
	return &MultiFeedHandler{
		userRepo:         userRepo,
		postRepo:         postRepo,
		userSettingsRepo: userSettingsRepo,
		weightsRepo:      weightsRepo,
		eventRepo:        eventRepo,
		reactionRepo:     reactionRepo,
		followRepo:       followRepo,
		userEmb:          userEmb,
	}
}

// enrichAndFilter batch-looks up reactions and save state, sets MyReaction
// and Saved on each post, and removes posts that the user has negatively reacted to.
func (h *MultiFeedHandler) enrichAndFilter(posts []model.Post, userID string) []model.Post {
	if len(posts) == 0 {
		return posts
	}

	postIDs := make([]string, len(posts))
	for i := range posts {
		postIDs[i] = posts[i].ID
	}

	reactions, err := h.reactionRepo.GetForPosts(postIDs, userID)
	if err != nil {
		slog.Warn("failed to lookup reactions for feed", "error", err)
	}

	savedSet, err := h.eventRepo.GetSavedForPosts(postIDs, userID)
	if err != nil {
		slog.Warn("failed to lookup saved state for feed", "error", err)
	}

	filtered := make([]model.Post, 0, len(posts))
	for i := range posts {
		if r, ok := reactions[posts[i].ID]; ok {
			if repository.NegativeReactions[r] {
				continue
			}
			posts[i].MyReaction = &r
		}
		if savedSet[posts[i].ID] {
			posts[i].Saved = true
		}
		filtered = append(filtered, posts[i])
	}
	return filtered
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

	posts = h.enrichAndFilter(posts, user.ID)

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

	posts = h.enrichAndFilter(posts, user.ID)

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

	if settings != nil && len(settings.FollowedTeams) > 0 {
		feedWeights.FollowedTeams = make(map[string]bool, len(settings.FollowedTeams))
		for _, t := range settings.FollowedTeams {
			feedWeights.FollowedTeams[t] = true
		}
	}

	// Inject followed-agent IDs so scorePost can boost their posts.
	if followedSet, err := h.followRepo.FollowedAgentIDSet(user.ID); err == nil {
		feedWeights.FollowedAgentIDs = followedSet
	}

	var userEmbed []float32
	if h.userEmb != nil {
		ue, err := h.userEmb.Get(context.Background(), user.ID)
		if err != nil {
			slog.Warn("forYou: user embedding lookup failed", "error", err)
		} else if ue != nil {
			userEmbed = ue.Embedding
		}
	}

	posts, nextCursor, err := h.postRepo.ListForYou(user.ID, *settings.Latitude, *settings.Longitude, settings.RadiusKm, cursor, limit, feedWeights, userEmbed)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load feed"})
		return
	}

	posts = h.enrichAndFilter(posts, user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.FeedResponse{Posts: posts, NextCursor: nextCursor})
}

// GetFollowing returns posts from agents the user follows, in reverse chronological order.
func (h *MultiFeedHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
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

	followedIDs, err := h.followRepo.ListFollowedAgentIDs(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load followed agents"})
		return
	}

	posts, nextCursor, err := h.postRepo.ListFollowing(followedIDs, cursor, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load following feed"})
		return
	}

	posts = h.enrichAndFilter(posts, user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.FeedResponse{Posts: posts, NextCursor: nextCursor})
}

func (h *MultiFeedHandler) GetSaved(w http.ResponseWriter, r *http.Request) {
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

	posts, nextCursor, err := h.postRepo.ListSaved(user.ID, cursor, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load saved posts"})
		return
	}

	posts = h.enrichAndFilter(posts, user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.FeedResponse{Posts: posts, NextCursor: nextCursor})
}
