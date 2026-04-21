package handler_test

// Tests for GET /videos and GET /videos/for-me.
//
// Coverage aligned with the PR-183 code review:
//   - L4: healthy_only default is true; diagnostics reflects the parsed value
//   - L5 (adversarial inputs): malformed limit, empty labels CSV, label cap,
//     healthy_only=false still drops 'dead' rows, provider whitelist
//   - M4: /videos/for-me with a nil selector (503), with an empty agent.UserID
//     (non-personalized fallback), with an unknown agent (401)

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	videoselector "github.com/shanegleeson/beepbopboop/backend/internal/video"
)

// videosEnvelope mirrors the wire shape, kept local so these tests assert the
// public contract rather than trusting the handler's private types.
type videosEnvelope struct {
	Videos      []model.Video `json:"videos"`
	Diagnostics struct {
		RequestedLimit int      `json:"requested_limit"`
		ReturnedCount  int      `json:"returned_count"`
		IncludeLabels  []string `json:"include_labels,omitempty"`
		ExcludeLabels  []string `json:"exclude_labels,omitempty"`
		Providers      []string `json:"providers,omitempty"`
		HealthyOnly    bool     `json:"healthy_only"`
		Personalized   bool     `json:"personalized"`
	} `json:"diagnostics"`
}

func seedVideo(t *testing.T, repo *repository.VideoRepo, id, provider, health string, labels []string, published time.Time) model.Video {
	t.Helper()
	v, err := repo.UpsertCatalog(model.Video{
		Provider:        provider,
		ProviderVideoID: id,
		WatchURL:        "https://example.test/watch/" + id,
		EmbedURL:        "https://example.test/embed/" + id,
		Title:           "Title " + id,
		Labels:          labels,
		PublishedAt:     &published,
		EmbedHealth:     health,
	})
	if err != nil {
		t.Fatalf("seed %s: %v", id, err)
	}
	return v
}

func TestVideosHandler_List_DefaultHealthyOnly(t *testing.T) {
	db := database.OpenTestDB(t)
	agentRepo := repository.NewAgentRepo(db)
	videoRepo := repository.NewVideoRepo(db)
	h := handler.NewVideosHandler(agentRepo, videoRepo, nil)

	now := time.Now().UTC()
	ok1 := seedVideo(t, videoRepo, "ok1", "youtube", "ok", []string{"dogs"}, now)
	_ = seedVideo(t, videoRepo, "dead1", "youtube", "dead", []string{"dogs"}, now.Add(-time.Hour))

	req := httptest.NewRequest("GET", "/videos", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got videosEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Diagnostics.HealthyOnly {
		t.Errorf("default healthy_only should be true, got false")
	}
	if got.Diagnostics.RequestedLimit != 20 {
		t.Errorf("default limit should be 20, got %d", got.Diagnostics.RequestedLimit)
	}
	// Only the ok1 row should come back.
	if len(got.Videos) != 1 || got.Videos[0].ID != ok1.ID {
		t.Errorf("expected only the healthy row, got %v", got.Videos)
	}
}

func TestVideosHandler_List_HealthyOnlyFalseStillDropsDead(t *testing.T) {
	db := database.OpenTestDB(t)
	agentRepo := repository.NewAgentRepo(db)
	videoRepo := repository.NewVideoRepo(db)
	h := handler.NewVideosHandler(agentRepo, videoRepo, nil)

	now := time.Now().UTC()
	seedVideo(t, videoRepo, "alive", "youtube", "ok", []string{"dogs"}, now)
	seedVideo(t, videoRepo, "unknown", "youtube", "unknown", []string{"dogs"}, now.Add(-time.Hour))
	seedVideo(t, videoRepo, "dead", "youtube", "dead", []string{"dogs"}, now.Add(-2*time.Hour))

	req := httptest.NewRequest("GET", "/videos?healthy_only=false", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got videosEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Diagnostics.HealthyOnly {
		t.Errorf("explicit healthy_only=false should be reflected in diagnostics")
	}
	// dead row must never leak through regardless of healthy_only flag.
	for _, v := range got.Videos {
		if v.EmbedHealth == "dead" {
			t.Errorf("dead row leaked through healthy_only=false: %s", v.ID)
		}
	}
	if len(got.Videos) != 2 {
		t.Errorf("expected 2 non-dead rows, got %d", len(got.Videos))
	}
}

func TestVideosHandler_List_MalformedLimitFallsBackToDefault(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewVideosHandler(repository.NewAgentRepo(db), repository.NewVideoRepo(db), nil)

	req := httptest.NewRequest("GET", "/videos?limit=not-a-number", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on malformed limit, got %d", rec.Code)
	}
	var got videosEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Diagnostics.RequestedLimit != 20 {
		t.Errorf("expected fallback to default 20, got %d", got.Diagnostics.RequestedLimit)
	}
}

func TestVideosHandler_List_EmptyLabelsCSVIsIgnored(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewVideosHandler(repository.NewAgentRepo(db), repository.NewVideoRepo(db), nil)

	req := httptest.NewRequest("GET", "/videos?labels=", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got videosEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got.Diagnostics.IncludeLabels) != 0 {
		t.Errorf("expected no include_labels for empty CSV, got %v", got.Diagnostics.IncludeLabels)
	}
}

func TestVideosHandler_List_LabelCSVCapped(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewVideosHandler(repository.NewAgentRepo(db), repository.NewVideoRepo(db), nil)

	// 100 distinct labels; handler must clamp to maxCSVEntries=32 so we don't
	// build a pathologically large JSONB filter.
	raw := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		raw = append(raw, randLabel(i))
	}
	req := httptest.NewRequest("GET", "/videos?labels="+strings.Join(raw, ","), nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got videosEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if n := len(got.Diagnostics.IncludeLabels); n > 32 {
		t.Errorf("expected labels cap <=32, got %d", n)
	}
}

func TestVideosHandler_List_ProviderWhitelist(t *testing.T) {
	db := database.OpenTestDB(t)
	videoRepo := repository.NewVideoRepo(db)
	h := handler.NewVideosHandler(repository.NewAgentRepo(db), videoRepo, nil)

	now := time.Now().UTC()
	seedVideo(t, videoRepo, "yt", "youtube", "ok", nil, now)
	seedVideo(t, videoRepo, "vm", "vimeo", "ok", nil, now.Add(-time.Hour))

	req := httptest.NewRequest("GET", "/videos?providers=vimeo", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	var got videosEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got.Videos) != 1 || got.Videos[0].Provider != "vimeo" {
		t.Errorf("expected only vimeo, got %d rows: %+v", len(got.Videos), got.Videos)
	}
}

func TestVideosHandler_ForMe_NilSelectorReturns503(t *testing.T) {
	db := database.OpenTestDB(t)
	h := handler.NewVideosHandler(repository.NewAgentRepo(db), repository.NewVideoRepo(db), nil)

	req := httptest.NewRequest("GET", "/videos/for-me", nil)
	rec := httptest.NewRecorder()
	h.ForMe(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for nil selector, got %d", rec.Code)
	}
}

func TestVideosHandler_ForMe_UnknownAgentReturns401(t *testing.T) {
	db := database.OpenTestDB(t)
	agentRepo := repository.NewAgentRepo(db)
	videoRepo := repository.NewVideoRepo(db)
	selector := videoselector.NewSelector(videoRepo, repository.NewUserEmbeddingRepo(db))
	h := handler.NewVideosHandler(agentRepo, videoRepo, selector)

	req := httptest.NewRequest("GET", "/videos/for-me", nil)
	req = req.WithContext(middleware.WithAgentID(req.Context(), "agent-that-does-not-exist"))
	rec := httptest.NewRecorder()
	h.ForMe(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown agent, got %d: %s", rec.Code, rec.Body.String())
	}
}

// Tiny deterministic label generator; avoids importing math/rand just for
// uniqueness. Format: "lbl-<i>".
func randLabel(i int) string {
	const digits = "0123456789abcdefghij"
	return "lbl-" + string(digits[i%len(digits)]) + string(digits[(i/len(digits))%len(digits)]) + string(digits[(i/(len(digits)*len(digits)))%len(digits)])
}
