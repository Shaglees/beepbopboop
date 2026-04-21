package video_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	videoselector "github.com/shanegleeson/beepbopboop/backend/internal/video"
)

func TestSelector_Select_DeterministicSeed(t *testing.T) {
	db := database.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("selector-seed-user")
	videoRepo := repository.NewVideoRepo(db)
	selector := videoselector.NewSelector(videoRepo, repository.NewUserEmbeddingRepo(db))

	seedVideos(t, videoRepo,
		videoFixture("video-a", []string{"wimp"}, 2),
		videoFixture("video-b", []string{"wimp"}, 2),
		videoFixture("video-c", []string{"wimp"}, 2),
	)

	seed := int64(42)
	first, err := selector.Select(ctx, videoselector.SelectOptions{UserID: user.ID, Limit: 3, Seed: &seed})
	if err != nil {
		t.Fatalf("first select: %v", err)
	}
	second, err := selector.Select(ctx, videoselector.SelectOptions{UserID: user.ID, Limit: 3, Seed: &seed})
	if err != nil {
		t.Fatalf("second select: %v", err)
	}

	assertVideoOrder(t, first.Videos, second.Videos)
}

func TestSelector_Select_IncludeExcludeLabels(t *testing.T) {
	db := database.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("selector-label-user")
	videoRepo := repository.NewVideoRepo(db)
	selector := videoselector.NewSelector(videoRepo, repository.NewUserEmbeddingRepo(db))

	seedVideos(t, videoRepo,
		videoFixture("cats-only", []string{"wimp", "cats"}, 5),
		videoFixture("dogs-only", []string{"wimp", "dogs"}, 5),
		videoFixture("cats-and-blocked", []string{"wimp", "cats", "blocked"}, 5),
	)

	result, err := selector.Select(ctx, videoselector.SelectOptions{
		UserID:        user.ID,
		Limit:         5,
		IncludeLabels: []string{"cats"},
		ExcludeLabels: []string{"blocked"},
	})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(result.Videos) != 1 {
		t.Fatalf("expected 1 video after include/exclude filtering, got %d", len(result.Videos))
	}
	if result.Videos[0].ProviderVideoID != "cats-only" {
		t.Errorf("expected cats-only, got %q", result.Videos[0].ProviderVideoID)
	}
}

func TestSelector_Select_ExcludesRecentlyPostedVideos(t *testing.T) {
	db := database.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepo(db)
	agentRepo := repository.NewAgentRepo(db)
	postRepo := repository.NewPostRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("selector-dedup-user")
	agent, _ := agentRepo.Create(user.ID, "Selector Agent")
	videoRepo := repository.NewVideoRepo(db)
	selector := videoselector.NewSelector(videoRepo, repository.NewUserEmbeddingRepo(db))

	videos := seedVideos(t, videoRepo,
		videoFixture("recently-posted", []string{"wimp"}, 7),
		videoFixture("still-eligible", []string{"wimp"}, 7),
	)
	post, err := postRepo.Create(repository.CreatePostParams{AgentID: agent.ID, UserID: user.ID, Title: "Posted", Body: "Already used."})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := videoRepo.InsertPostHistory(post.ID, videos[0].ID, user.ID); err != nil {
		t.Fatalf("insert post history: %v", err)
	}

	result, err := selector.Select(ctx, videoselector.SelectOptions{UserID: user.ID, Limit: 5})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(result.Videos) != 1 {
		t.Fatalf("expected 1 eligible video after dedup, got %d", len(result.Videos))
	}
	if result.Videos[0].ProviderVideoID != "still-eligible" {
		t.Errorf("expected still-eligible, got %q", result.Videos[0].ProviderVideoID)
	}
}

func TestSelector_Select_LowInventoryReturnsDiagnostics(t *testing.T) {
	db := database.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("selector-low-inventory-user")
	videoRepo := repository.NewVideoRepo(db)
	selector := videoselector.NewSelector(videoRepo, repository.NewUserEmbeddingRepo(db))

	seedVideos(t, videoRepo, videoFixture("only-one", []string{"rare"}, 1))

	result, err := selector.Select(ctx, videoselector.SelectOptions{
		UserID:        user.ID,
		Limit:         3,
		IncludeLabels: []string{"rare"},
	})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got, want := len(result.Videos), 1; got != want {
		t.Fatalf("returned videos: got %d want %d", got, want)
	}
	if result.Diagnostics.RequestedLimit != 3 || result.Diagnostics.ReturnedCount != 1 {
		t.Errorf("unexpected diagnostics: %+v", result.Diagnostics)
	}
}

func TestSelector_Select_PrefersCloserEmbeddingWhenAvailable(t *testing.T) {
	db := database.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepo(db)
	user, _ := userRepo.FindOrCreateByFirebaseUID("selector-embedding-user")
	videoRepo := repository.NewVideoRepo(db)
	userEmbeddingRepo := repository.NewUserEmbeddingRepo(db)
	selector := videoselector.NewSelector(videoRepo, userEmbeddingRepo)

	videos := seedVideos(t, videoRepo,
		videoFixture("close-match", []string{"wimp"}, 10),
		videoFixture("far-match", []string{"wimp"}, 10),
	)
	if err := videoRepo.UpsertEmbedding(videos[0].ID, vecWithHotIndex(0), "test-model"); err != nil {
		t.Fatalf("upsert close embedding: %v", err)
	}
	if err := videoRepo.UpsertEmbedding(videos[1].ID, vecWithHotIndex(1), "test-model"); err != nil {
		t.Fatalf("upsert far embedding: %v", err)
	}
	if err := userEmbeddingRepo.Upsert(ctx, user.ID, vecWithHotIndex(0), 3); err != nil {
		t.Fatalf("upsert user embedding: %v", err)
	}

	seed := int64(7)
	result, err := selector.Select(ctx, videoselector.SelectOptions{UserID: user.ID, Limit: 2, Seed: &seed})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(result.Videos) < 1 {
		t.Fatalf("expected at least one selected video")
	}
	if result.Videos[0].ProviderVideoID != "close-match" {
		t.Errorf("expected closest embedding first, got %q", result.Videos[0].ProviderVideoID)
	}
	if !result.Diagnostics.HadUserEmbedding {
		t.Errorf("expected diagnostics to note user embedding presence")
	}
}

func seedVideos(t *testing.T, repo *repository.VideoRepo, videos ...model.Video) []model.Video {
	t.Helper()
	out := make([]model.Video, 0, len(videos))
	for _, v := range videos {
		got, err := repo.UpsertCatalog(v)
		if err != nil {
			t.Fatalf("upsert catalog %s: %v", v.ProviderVideoID, err)
		}
		out = append(out, got)
	}
	return out
}

func videoFixture(id string, labels []string, ageDays int) model.Video {
	publishedAt := time.Now().Add(-time.Duration(ageDays) * 24 * time.Hour).UTC().Truncate(time.Second)
	return model.Video{
		Provider:        "youtube",
		ProviderVideoID: id,
		WatchURL:        "https://www.youtube.com/watch?v=" + id,
		EmbedURL:        "https://www.youtube.com/embed/" + id,
		Title:           fmt.Sprintf("Video %s", id),
		Description:     "selector test fixture",
		ThumbnailURL:    "https://example.com/" + id + ".jpg",
		PublishedAt:     &publishedAt,
		Labels:          labels,
		EmbedHealth:     "unknown",
	}
}

func vecWithHotIndex(idx int) []float32 {
	vec := make([]float32, 1536)
	vec[idx] = 1.0
	return vec
}

func assertVideoOrder(t *testing.T, a, b []model.Video) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("length mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatalf("order mismatch at %d: %q vs %q", i, a[i].ID, b[i].ID)
		}
	}
}
