package handler

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	videoselector "github.com/shanegleeson/beepbopboop/backend/internal/video"
)

// VideosHandler exposes two read endpoints on the historical video catalog
// for agents/skills composing `display_hint: video_embed` posts.
//
//  - GET /videos          — simple catalog list. No personalization, no dedup.
//                           Caller picks one of the returned videos.
//  - GET /videos/for-me   — personalized selection using the existing Selector
//                           so per-user 180-day dedup + embedding similarity
//                           apply.
//
// Both return the same shape: `{videos: [Video...], diagnostics: {...}}`.
type VideosHandler struct {
	agentRepo *repository.AgentRepo
	videoRepo *repository.VideoRepo
	selector  *videoselector.Selector
}

// NewVideosHandler builds the handler. The selector is optional — if nil the
// /videos/for-me route returns 503 so callers know personalization is off.
func NewVideosHandler(agentRepo *repository.AgentRepo, videoRepo *repository.VideoRepo, selector *videoselector.Selector) *VideosHandler {
	return &VideosHandler{agentRepo: agentRepo, videoRepo: videoRepo, selector: selector}
}

// videosListResponse is the envelope returned by both endpoints.
//
// We add a diagnostics block so operators and agents can see why a query
// returned fewer rows than requested ("2 excluded by embed_health, 1 by dedup").
type videosListResponse struct {
	Videos      []model.Video         `json:"videos"`
	Diagnostics videosListDiagnostics `json:"diagnostics"`
}

type videosListDiagnostics struct {
	RequestedLimit int      `json:"requested_limit"`
	ReturnedCount  int      `json:"returned_count"`
	IncludeLabels  []string `json:"include_labels,omitempty"`
	ExcludeLabels  []string `json:"exclude_labels,omitempty"`
	Providers      []string `json:"providers,omitempty"`
	HealthyOnly    bool     `json:"healthy_only"`
	Personalized   bool     `json:"personalized"`
}

// List handles GET /videos.
//
// Query parameters (all optional):
//   - limit=N              1..100, default 20
//   - labels=a,b,c         ANY-match include filter
//   - exclude_labels=d,e   NONE-match exclude filter
//   - providers=youtube    whitelist providers
//   - healthy_only=true    (default true) only return embed_health='ok'
func (h *VideosHandler) List(w http.ResponseWriter, r *http.Request) {
	params := parseVideoListParams(r)

	videos, err := h.videoRepo.ListCatalog(r.Context(), params)
	if err != nil {
		slog.Error("videos: list catalog failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list videos"})
		return
	}

	writeJSON(w, http.StatusOK, videosListResponse{
		Videos: videos,
		Diagnostics: videosListDiagnostics{
			RequestedLimit: params.Limit,
			ReturnedCount:  len(videos),
			IncludeLabels:  params.IncludeLabels,
			ExcludeLabels:  params.ExcludeLabels,
			Providers:      params.Providers,
			HealthyOnly:    params.HealthyOnly,
			Personalized:   false,
		},
	})
}

// ForMe handles GET /videos/for-me. Requires agent-token auth (AgentAuth
// middleware chain), resolves the agent's user, and calls the Selector.
func (h *VideosHandler) ForMe(w http.ResponseWriter, r *http.Request) {
	if h.selector == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "personalization unavailable on this deployment",
		})
		return
	}

	agentID := middleware.AgentIDFromContext(r.Context())
	agent, err := h.agentRepo.GetByID(agentID)
	// Distinguish "auth lookup failed" (500) from "agent not found" (401)
	// from "agent has no user" (fall back to non-personalized list).
	// AgentRepo.GetByID wraps sql.ErrNoRows with a query prefix, hence the
	// errors.Is check — callers upstream of the wrap still match cleanly.
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unknown agent"})
			return
		}
		slog.Error("videos/for-me: agent lookup failed", "error", err, "agent_id", agentID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve agent"})
		return
	}
	if agent == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unknown agent"})
		return
	}

	q := r.URL.Query()
	limit := parseIntBounded(q.Get("limit"), 5, 1, 100)
	include := parseCSV(q.Get("labels"), maxCSVEntries)
	exclude := parseCSV(q.Get("exclude_labels"), maxCSVEntries)

	// An unassigned (or shared) agent has no per-user history or embedding,
	// which means Selector can't personalize. Fall back to the simple list
	// rather than returning garbage personalized-but-not-really results.
	if agent.UserID == "" {
		params := repository.VideoCatalogListParams{
			Limit:         limit,
			IncludeLabels: include,
			ExcludeLabels: exclude,
			HealthyOnly:   true,
		}
		videos, listErr := h.videoRepo.ListCatalog(r.Context(), params)
		if listErr != nil {
			slog.Error("videos/for-me: catalog fallback failed", "error", listErr, "agent_id", agentID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list videos"})
			return
		}
		writeJSON(w, http.StatusOK, videosListResponse{
			Videos: videos,
			Diagnostics: videosListDiagnostics{
				RequestedLimit: limit,
				ReturnedCount:  len(videos),
				IncludeLabels:  include,
				ExcludeLabels:  exclude,
				HealthyOnly:    true,
				Personalized:   false, // fallback — surfaced so callers can notice.
			},
		})
		return
	}

	result, err := h.selector.Select(r.Context(), videoselector.SelectOptions{
		UserID:        agent.UserID,
		Limit:         limit,
		IncludeLabels: include,
		ExcludeLabels: exclude,
	})
	if err != nil {
		slog.Error("videos/for-me: select failed", "error", err, "user_id", agent.UserID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to select videos"})
		return
	}

	writeJSON(w, http.StatusOK, videosListResponse{
		Videos: result.Videos,
		Diagnostics: videosListDiagnostics{
			RequestedLimit: result.Diagnostics.RequestedLimit,
			ReturnedCount:  result.Diagnostics.ReturnedCount,
			IncludeLabels:  result.Diagnostics.IncludeLabels,
			ExcludeLabels:  result.Diagnostics.ExcludeLabels,
			HealthyOnly:    true, // selector always enforces it
			Personalized:   true,
		},
	})
}

func parseVideoListParams(r *http.Request) repository.VideoCatalogListParams {
	q := r.URL.Query()
	healthy := true
	if raw := q.Get("healthy_only"); raw != "" {
		b, err := strconv.ParseBool(raw)
		if err == nil {
			healthy = b
		}
	}
	return repository.VideoCatalogListParams{
		Limit:         parseIntBounded(q.Get("limit"), 20, 1, 100),
		IncludeLabels: parseCSV(q.Get("labels"), maxCSVEntries),
		ExcludeLabels: parseCSV(q.Get("exclude_labels"), maxCSVEntries),
		Providers:     parseCSV(q.Get("providers"), maxCSVEntries),
		HealthyOnly:   healthy,
	}
}

// maxCSVEntries caps how many comma-separated values we accept in a single
// query parameter. Keeps a single request from allocating a 10k-entry JSONB
// filter or exploding the SQL IN list.
const maxCSVEntries = 32

// parseCSV splits a comma-separated query parameter, trims and dedups entries
// (case-insensitive normalization is a caller concern), and truncates to
// `cap` entries. A zero/negative cap disables the cap.
func parseCSV(s string, cap int) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
		if cap > 0 && len(out) >= cap {
			break
		}
	}
	return out
}

// parseIntBounded clamps a query-string int to [lo,hi] with default def.
// `lo`/`hi` are used instead of `min`/`max` to avoid shadowing the Go 1.21+
// built-ins.
func parseIntBounded(raw string, def, lo, hi int) int {
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}
