package handler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// labelDisplayNames maps raw label keys to user-friendly display names.
var labelDisplayNames = map[string]string{
	"sports":       "Sports",
	"local_events": "Local Events",
	"local-events": "Local Events",
	"weather":      "Weather",
	"fashion":      "Fashion",
	"trending":     "Trending",
	"hacker-news":  "Tech News",
	"hacker_news":  "Tech News",
	"nhl":          "NHL Hockey",
	"nba":          "NBA Basketball",
	"nfl":          "NFL Football",
	"music":        "Music",
	"food":         "Food & Drink",
	"travel":       "Travel",
	"technology":   "Technology",
	"arts":         "Arts & Culture",
	"community":    "Community",
	"health":       "Health",
	"business":     "Business",
}

var summaryDefaultWeights = &repository.FeedWeights{
	FreshnessBias: 0.8,
	GeoBias:       0.3,
	LabelWeights:  map[string]float64{"fashion": 0.4, "sports": 0.4, "trending": 0.3},
	TypeWeights:   map[string]float64{"event": 0.3, "discovery": 0.2, "article": 0.1, "video": 0.2},
}

type weightsSummaryResponse struct {
	TopLabels  []string `json:"top_labels"`
	DataPoints int      `json:"data_points"`
}

// WeightsSummaryHandler returns a human-readable feed personalisation summary (Firebase auth).
type WeightsSummaryHandler struct {
	userRepo    *repository.UserRepo
	weightsRepo *repository.WeightsRepo
	eventRepo   *repository.EventRepo
}

func NewWeightsSummaryHandler(
	userRepo *repository.UserRepo,
	weightsRepo *repository.WeightsRepo,
	eventRepo *repository.EventRepo,
) *WeightsSummaryHandler {
	return &WeightsSummaryHandler{
		userRepo:    userRepo,
		weightsRepo: weightsRepo,
		eventRepo:   eventRepo,
	}
}

func (h *WeightsSummaryHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	uid := middleware.FirebaseUIDFromContext(r.Context())
	user, err := h.userRepo.FindOrCreateByFirebaseUID(uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve user"})
		return
	}

	summary, err := h.eventRepo.Summary(user.ID, 14)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load engagement"})
		return
	}

	weights, err := h.weightsRepo.GetOrCompute(user.ID, h.eventRepo, summaryDefaultWeights)
	if err != nil || weights == nil {
		weights = summaryDefaultWeights
	}

	type kv struct {
		key string
		val float64
	}
	sorted := make([]kv, 0, len(weights.LabelWeights))
	for k, v := range weights.LabelWeights {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].val > sorted[j].val })

	topLabels := make([]string, 0, 5)
	for _, item := range sorted {
		if len(topLabels) >= 5 {
			break
		}
		if name, ok := labelDisplayNames[item.key]; ok {
			topLabels = append(topLabels, name)
		} else {
			topLabels = append(topLabels, toDisplayName(item.key))
		}
	}

	writeJSON(w, http.StatusOK, weightsSummaryResponse{
		TopLabels:  topLabels,
		DataPoints: summary.TotalEvents,
	})
}

// toDisplayName converts a raw label key like "local-events" to "Local Events".
func toDisplayName(key string) string {
	s := strings.ReplaceAll(key, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
