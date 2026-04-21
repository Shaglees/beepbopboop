package video_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/handler"
	"github.com/shanegleeson/beepbopboop/backend/internal/ingest/wimp"
	"github.com/shanegleeson/beepbopboop/backend/internal/middleware"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	videokit "github.com/shanegleeson/beepbopboop/backend/internal/video"
)

type e2eLister struct{ urls []string }

func (l *e2eLister) ListPageURLs(ctx context.Context, offset, limit int) ([]string, error) {
	if offset > 0 {
		return nil, nil
	}
	return l.urls, nil
}

type e2eInspector struct{ inspections map[string]wimp.Inspection }

func (i *e2eInspector) InspectArchivedURL(ctx context.Context, rawURL string) (wimp.Inspection, error) {
	inspection, ok := i.inspections[rawURL]
	if !ok {
		return wimp.Inspection{}, fmt.Errorf("missing inspection for %s", rawURL)
	}
	return inspection, nil
}

func TestVideoDiscoveryFlow_BackfillSelectPublishAndDedup(t *testing.T) {
	db := database.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	videoRepo := repository.NewVideoRepo(db)
	userEmbeddingRepo := repository.NewUserEmbeddingRepo(db)

	user, _ := userRepo.FindOrCreateByFirebaseUID("firebase-video-e2e")
	agent, _ := agentRepo.Create(user.ID, "Discovery Agent")

	backfiller := wimp.NewBackfiller(
		&e2eLister{urls: []string{"https://www.wimp.com/a-blooper-reel-of-beatles-recordings/"}},
		&e2eInspector{inspections: map[string]wimp.Inspection{
			"https://www.wimp.com/a-blooper-reel-of-beatles-recordings/": {
				Capture: wimp.Capture{Timestamp: "20190109001127", Original: "https://www.wimp.com/a-blooper-reel-of-beatles-recordings/"},
				Metadata: wimp.Metadata{
					Title:        "A blooper reel of Beatles recordings",
					Description:  "A collection of studio chatter and rough takes from Beatles recording sessions.",
					ThumbnailURL: "https://example.com/beatles.jpg",
					CanonicalURL: "https://www.wimp.com/a-blooper-reel-of-beatles-recordings/",
				},
				Embed: &wimp.Embed{
					Provider: "youtube",
					VideoID:  "NZd3R2iw4cA",
					WatchURL: "https://www.youtube.com/watch?v=NZd3R2iw4cA",
					EmbedURL: "https://www.youtube.com/embed/NZd3R2iw4cA",
				},
			},
		}},
		videoRepo,
	)

	if stats, err := backfiller.Run(ctx, wimp.BackfillOptions{CrawlBudget: 1, PageSize: 1, MaxRetries: 2}); err != nil {
		t.Fatalf("backfill run: %v", err)
	} else if stats.CandidatesUpserted != 1 {
		t.Fatalf("expected 1 candidate upserted, got %+v", stats)
	}

	selector := videokit.NewSelector(videoRepo, userEmbeddingRepo)
	result, err := selector.Select(ctx, videokit.SelectOptions{UserID: user.ID, Limit: 1})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(result.Videos) != 1 {
		t.Fatalf("expected 1 selected video, got %d", len(result.Videos))
	}

	selected := result.Videos[0]
	if selected.Title != "Beatles studio bloopers you probably haven't heard" {
		t.Fatalf("expected enriched title, got %q", selected.Title)
	}

	postHandler := handler.NewPostHandler(agentRepo, postRepo, videoRepo)
	body := map[string]any{
		"title":        selected.Title,
		"body":         selected.Description,
		"post_type":    "video",
		"display_hint": "video_embed",
		"labels":       selected.Labels,
		"external_url": mustJSON(t, map[string]any{
			"provider":             selected.Provider,
			"video_id":             selected.ProviderVideoID,
			"watch_url":            selected.WatchURL,
			"embed_url":            selected.EmbedURL,
			"thumbnail_url":        selected.ThumbnailURL,
			"channel_title":        selected.ChannelTitle,
			"supports_preview_cap": selected.SupportsPrevCap,
		}),
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(payload))
	req = req.WithContext(middleware.WithAgentID(req.Context(), agent.ID))
	rec := httptest.NewRecorder()
	postHandler.CreatePost(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	after, err := selector.Select(ctx, videokit.SelectOptions{UserID: user.ID, Limit: 1})
	if err != nil {
		t.Fatalf("select after publish: %v", err)
	}
	if len(after.Videos) != 0 {
		t.Fatalf("expected published video to be deduped, got %d candidates", len(after.Videos))
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(b)
}
