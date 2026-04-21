package videohealth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
	"github.com/shanegleeson/beepbopboop/backend/internal/videohealth"
)

type stubChecker struct {
	statuses map[string]string
	errFor   map[string]error
}

func (s stubChecker) CheckEmbed(ctx context.Context, v model.Video) (string, error) {
	if err := s.errFor[v.ID]; err != nil {
		return "", err
	}
	if status, ok := s.statuses[v.ID]; ok {
		return status, nil
	}
	return "unknown", nil
}

func TestWorker_RunOnce_TransitionsStatuses(t *testing.T) {
	db := database.OpenTestDB(t)
	ctx := context.Background()
	repo := repository.NewVideoRepo(db)

	unknown, err := repo.UpsertCatalog(fixtureVideo("unknown-to-ok", "unknown"))
	if err != nil {
		t.Fatalf("seed unknown: %v", err)
	}
	blocked, err := repo.UpsertCatalog(fixtureVideo("unknown-to-blocked", "unknown"))
	if err != nil {
		t.Fatalf("seed blocked: %v", err)
	}
	gone, err := repo.UpsertCatalog(fixtureVideo("ok-to-gone", "ok"))
	if err != nil {
		t.Fatalf("seed gone: %v", err)
	}
	if _, err := db.Exec(`UPDATE video_catalog SET embed_checked_at = NOW() - INTERVAL '14 days' WHERE id = $1`, gone.ID); err != nil {
		t.Fatalf("mark gone stale: %v", err)
	}

	worker := videohealth.NewWorker(repo, stubChecker{statuses: map[string]string{
		unknown.ID: "ok",
		blocked.ID: "blocked",
		gone.ID:    "gone",
	}})

	stats, err := worker.RunOnce(ctx, 10, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if stats.Checked != 3 {
		t.Fatalf("expected 3 checked, got %d", stats.Checked)
	}
	if stats.OK != 1 || stats.Blocked != 1 || stats.Gone != 1 {
		t.Errorf("unexpected stats: %+v", stats)
	}

	assertHealth(t, repo, unknown.ID, "ok")
	assertHealth(t, repo, blocked.ID, "blocked")
	assertHealth(t, repo, gone.ID, "gone")
}

func TestWorker_RunOnce_ProviderErrorsAreCountedButDoNotAbort(t *testing.T) {
	db := database.OpenTestDB(t)
	ctx := context.Background()
	repo := repository.NewVideoRepo(db)

	good, err := repo.UpsertCatalog(fixtureVideo("good-after-error", "unknown"))
	if err != nil {
		t.Fatalf("seed good: %v", err)
	}
	bad, err := repo.UpsertCatalog(fixtureVideo("fails-check", "unknown"))
	if err != nil {
		t.Fatalf("seed bad: %v", err)
	}

	worker := videohealth.NewWorker(repo, stubChecker{
		statuses: map[string]string{good.ID: "ok"},
		errFor:   map[string]error{bad.ID: fmt.Errorf("temporary upstream failure")},
	})

	stats, err := worker.RunOnce(ctx, 10, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if stats.Checked != 2 || stats.Failures != 1 || stats.OK != 1 {
		t.Errorf("unexpected stats: %+v", stats)
	}
	assertHealth(t, repo, good.ID, "ok")
	got, err := repo.GetByID(bad.ID)
	if err != nil {
		t.Fatalf("GetByID(%s): %v", bad.ID, err)
	}
	if got == nil {
		t.Fatalf("missing video %s", bad.ID)
	}
	if got.EmbedHealth != "unknown" {
		t.Fatalf("embed_health for %s: got %q want unknown", bad.ID, got.EmbedHealth)
	}
	if got.EmbedCheckedAt != nil {
		t.Fatalf("expected failed check to leave embed_checked_at unset, got %v", got.EmbedCheckedAt)
	}
}

func assertHealth(t *testing.T, repo *repository.VideoRepo, id, want string) {
	t.Helper()
	got, err := repo.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID(%s): %v", id, err)
	}
	if got == nil {
		t.Fatalf("missing video %s", id)
	}
	if got.EmbedHealth != want {
		t.Fatalf("embed_health for %s: got %q want %q", id, got.EmbedHealth, want)
	}
	if got.EmbedCheckedAt == nil {
		t.Fatalf("expected embed_checked_at for %s", id)
	}
}

func fixtureVideo(id, health string) model.Video {
	publishedAt := time.Now().Add(-48 * time.Hour)
	return model.Video{
		Provider:        "youtube",
		ProviderVideoID: id,
		WatchURL:        "https://www.youtube.com/watch?v=" + id,
		EmbedURL:        "https://www.youtube.com/embed/" + id,
		Title:           "Fixture " + id,
		Description:     "fixture",
		ThumbnailURL:    "https://example.com/" + id + ".jpg",
		PublishedAt:     &publishedAt,
		Labels:          []string{"wimp"},
		EmbedHealth:     health,
	}
}
