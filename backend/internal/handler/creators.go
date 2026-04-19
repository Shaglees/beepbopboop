package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

type CreatorsHandler struct {
	creatorRepo      *repository.LocalCreatorRepo
	userRepo         *repository.UserRepo
	userSettingsRepo *repository.UserSettingsRepo
}

func NewCreatorsHandler(creatorRepo *repository.LocalCreatorRepo, userRepo *repository.UserRepo, userSettingsRepo *repository.UserSettingsRepo) *CreatorsHandler {
	return &CreatorsHandler{
		creatorRepo:      creatorRepo,
		userRepo:         userRepo,
		userSettingsRepo: userSettingsRepo,
	}
}

// Create handles POST /creators (agent-auth). Upserts a creator profile.
func (h *CreatorsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateCreatorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Name == "" || req.Designation == "" || req.Source == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, designation, and source are required"})
		return
	}

	creator, err := h.creatorRepo.Upsert(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save creator"})
		return
	}

	writeJSON(w, http.StatusCreated, creator)
}

// GetNearby handles GET /creators/nearby (Firebase-auth). Returns cached creators
// near the user's stored location as creator_spotlight posts.
func (h *CreatorsHandler) GetNearby(w http.ResponseWriter, r *http.Request) {
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

	radius := settings.RadiusKm
	if radius <= 0 {
		radius = 25.0
	}

	creators, _, err := h.creatorRepo.ListNearby(*settings.Latitude, *settings.Longitude, radius, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query creators"})
		return
	}

	posts := make([]model.Post, 0, len(creators))
	for _, c := range creators {
		posts = append(posts, creatorToPost(c))
	}

	writeJSON(w, http.StatusOK, model.FeedResponse{Posts: posts})
}

func creatorToPost(c model.LocalCreator) model.Post {
	payload := map[string]any{
		"designation":   c.Designation,
		"notable_works": c.NotableWorks,
		"tags":          c.Tags,
		"source":        c.Source,
		"area_name":     c.AreaName,
	}
	if len(c.Links) > 0 {
		var links any
		json.Unmarshal(c.Links, &links)
		payload["links"] = links
	}
	externalJSON, _ := json.Marshal(payload)

	return model.Post{
		ID:          c.ID,
		AgentID:     "system",
		AgentName:   "Local Creators",
		UserID:      "system",
		Title:       c.Name,
		Body:        c.Bio,
		ImageURL:    c.ImageURL,
		Locality:    c.AreaName,
		Latitude:    c.Lat,
		Longitude:   c.Lon,
		PostType:    "discovery",
		Visibility:  "public",
		DisplayHint: "creator_spotlight",
		ExternalURL: string(externalJSON),
		Status:      "published",
		CreatedAt:   c.DiscoveredAt,
	}
}
