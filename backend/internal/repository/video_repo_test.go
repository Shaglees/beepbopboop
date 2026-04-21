package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// --- fixtures -----------------------------------------------------------------

func sampleVideo(overrides func(*model.Video)) model.Video {
	v := model.Video{
		Provider:        "youtube",
		ProviderVideoID: "dQw4w9WgXcQ",
		WatchURL:        "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		EmbedURL:        "https://www.youtube.com/embed/dQw4w9WgXcQ",
		Title:           "Never Gonna Give You Up",
		Description:     "Classic 80s hit.",
		ChannelTitle:    "Rick Astley",
		ThumbnailURL:    "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
		DurationSec:     213,
		PublishedAt:     ptrTime(time.Date(2009, 10, 25, 0, 0, 0, 0, time.UTC)),
		SourceURL:       "https://wimp.com/never-gonna-give-you-up/",
		SourceDesc:      "Wimp.com feature: the meme that keeps on giving.",
		Labels:          []string{"music", "meme", "classic"},
		SupportsPrevCap: true,
		EmbedHealth:     "unknown",
	}
	if overrides != nil {
		overrides(&v)
	}
	return v
}

func ptrTime(t time.Time) *time.Time { return &t }

// --- video_catalog ------------------------------------------------------------

func TestVideoRepo_UpsertCatalog_CreatesNew(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	v := sampleVideo(nil)
	got, err := repo.UpsertCatalog(v)
	if err != nil {
		t.Fatalf("UpsertCatalog: %v", err)
	}
	if got.ID == "" {
		t.Fatalf("expected non-empty ID from upsert")
	}
	if got.Provider != "youtube" || got.ProviderVideoID != "dQw4w9WgXcQ" {
		t.Errorf("provider keys not persisted: %+v", got)
	}
	if len(got.Labels) != 3 || got.Labels[0] != "music" {
		t.Errorf("labels not persisted: %+v", got.Labels)
	}
	if got.EmbedHealth != "unknown" {
		t.Errorf("expected default embed_health=unknown, got %q", got.EmbedHealth)
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("expected CreatedAt to be set")
	}
}

func TestVideoRepo_UpsertCatalog_UpdatesOnConflict(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	first, err := repo.UpsertCatalog(sampleVideo(nil))
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	updated, err := repo.UpsertCatalog(sampleVideo(func(v *model.Video) {
		v.Title = "Never Gonna Give You Up (Remastered)"
		v.Labels = []string{"music", "meme", "classic", "remaster"}
	}))
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	if first.ID != updated.ID {
		t.Fatalf("expected same catalog ID across upserts, got %s vs %s", first.ID, updated.ID)
	}
	if updated.Title != "Never Gonna Give You Up (Remastered)" {
		t.Errorf("title not updated: %s", updated.Title)
	}
	if len(updated.Labels) != 4 {
		t.Errorf("expected 4 labels after update, got %v", updated.Labels)
	}
}

func TestVideoRepo_GetByProviderID(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	_, err := repo.UpsertCatalog(sampleVideo(nil))
	if err != nil {
		t.Fatalf("seed upsert: %v", err)
	}

	got, err := repo.GetByProviderID("youtube", "dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("GetByProviderID: %v", err)
	}
	if got == nil {
		t.Fatalf("expected a row, got nil")
	}
	if got.Title != "Never Gonna Give You Up" {
		t.Errorf("unexpected title %q", got.Title)
	}

	miss, err := repo.GetByProviderID("youtube", "does-not-exist")
	if err != nil {
		t.Fatalf("GetByProviderID(miss): %v", err)
	}
	if miss != nil {
		t.Errorf("expected nil on miss, got %+v", miss)
	}
}

func TestVideoRepo_UpdateEmbedHealth(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	v, err := repo.UpsertCatalog(sampleVideo(nil))
	if err != nil {
		t.Fatalf("seed upsert: %v", err)
	}
	if err := repo.UpdateEmbedHealth(v.ID, "blocked"); err != nil {
		t.Fatalf("UpdateEmbedHealth: %v", err)
	}

	got, err := repo.GetByID(v.ID)
	if err != nil || got == nil {
		t.Fatalf("GetByID: %v, row=%v", err, got)
	}
	if got.EmbedHealth != "blocked" {
		t.Errorf("expected embed_health=blocked, got %q", got.EmbedHealth)
	}
	if got.EmbedCheckedAt == nil {
		t.Errorf("expected embed_checked_at to be stamped")
	}
}

func TestVideoRepo_ListForEmbedHealthCheck_PrioritizesUnknownThenStale(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	unknown, err := repo.UpsertCatalog(sampleVideo(func(v *model.Video) {
		v.ProviderVideoID = "unknown-priority"
		v.EmbedURL = "https://www.youtube.com/embed/unknown-priority"
		v.WatchURL = "https://www.youtube.com/watch?v=unknown-priority"
		v.EmbedHealth = "unknown"
	}))
	if err != nil {
		t.Fatalf("seed unknown: %v", err)
	}
	stale, err := repo.UpsertCatalog(sampleVideo(func(v *model.Video) {
		v.ProviderVideoID = "stale-priority"
		v.EmbedURL = "https://www.youtube.com/embed/stale-priority"
		v.WatchURL = "https://www.youtube.com/watch?v=stale-priority"
		v.EmbedHealth = "ok"
	}))
	if err != nil {
		t.Fatalf("seed stale: %v", err)
	}
	fresh, err := repo.UpsertCatalog(sampleVideo(func(v *model.Video) {
		v.ProviderVideoID = "fresh-last"
		v.EmbedURL = "https://www.youtube.com/embed/fresh-last"
		v.WatchURL = "https://www.youtube.com/watch?v=fresh-last"
		v.EmbedHealth = "ok"
	}))
	if err != nil {
		t.Fatalf("seed fresh: %v", err)
	}

	if _, err := db.Exec(`UPDATE video_catalog SET embed_checked_at = NOW() - INTERVAL '10 days' WHERE id = $1`, stale.ID); err != nil {
		t.Fatalf("mark stale: %v", err)
	}
	if _, err := db.Exec(`UPDATE video_catalog SET embed_checked_at = NOW() - INTERVAL '1 day' WHERE id = $1`, fresh.ID); err != nil {
		t.Fatalf("mark fresh: %v", err)
	}

	got, err := repo.ListForEmbedHealthCheck(context.Background(), 7*24*time.Hour, 10)
	if err != nil {
		t.Fatalf("ListForEmbedHealthCheck: %v", err)
	}
	if len(got) < 3 {
		t.Fatalf("expected at least 3 rows, got %d", len(got))
	}
	if got[0].ID != unknown.ID {
		t.Fatalf("expected unknown video first, got %q", got[0].ProviderVideoID)
	}
	if got[1].ID != stale.ID {
		t.Fatalf("expected stale video second, got %q", got[1].ProviderVideoID)
	}
}

// --- video_embeddings ---------------------------------------------------------

func TestVideoRepo_UpsertAndGetEmbedding(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	v, err := repo.UpsertCatalog(sampleVideo(nil))
	if err != nil {
		t.Fatalf("seed upsert: %v", err)
	}

	vec := make([]float32, 1536)
	for i := range vec {
		vec[i] = float32(i%7) * 0.01
	}
	if err := repo.UpsertEmbedding(v.ID, vec, "text-embedding-3-small"); err != nil {
		t.Fatalf("UpsertEmbedding: %v", err)
	}

	got, err := repo.GetEmbedding(v.ID)
	if err != nil {
		t.Fatalf("GetEmbedding: %v", err)
	}
	if len(got) != 1536 {
		t.Fatalf("expected 1536 dims back, got %d", len(got))
	}
	// Sample a few values to ensure round-trip fidelity within float32 precision.
	for _, idx := range []int{0, 1, 500, 1535} {
		if diff := float32abs(got[idx] - vec[idx]); diff > 1e-3 {
			t.Errorf("dim %d drift too high: got=%f want=%f", idx, got[idx], vec[idx])
		}
	}
}

func TestVideoRepo_GetEmbedding_Missing(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	got, err := repo.GetEmbedding("no-such-id")
	if err != nil {
		t.Fatalf("GetEmbedding(miss): %v", err)
	}
	if got != nil {
		t.Errorf("expected nil embedding on miss, got %d dims", len(got))
	}
}

// --- video_post_history -------------------------------------------------------

func TestVideoRepo_InsertPostHistory_AndDedupWindow(t *testing.T) {
	db := database.OpenTestDB(t)

	userRepo := repository.NewUserRepo(db)
	user, err := userRepo.FindOrCreateByFirebaseUID("firebase-video-history")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	agentRepo := repository.NewAgentRepo(db)
	agent, err := agentRepo.Create(user.ID, "Video Bot")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	postRepo := repository.NewPostRepo(db)
	post, err := postRepo.Create(repository.CreatePostParams{
		AgentID: agent.ID,
		UserID:  user.ID,
		Title:   "Me at the zoo",
		Body:    "The first YouTube upload, shared by an agent.",
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	repo := repository.NewVideoRepo(db)
	v, err := repo.UpsertCatalog(sampleVideo(nil))
	if err != nil {
		t.Fatalf("seed upsert: %v", err)
	}

	if err := repo.InsertPostHistory(post.ID, v.ID, user.ID); err != nil {
		t.Fatalf("InsertPostHistory: %v", err)
	}

	recent, err := repo.ListPostHistoryForUserSince(user.ID, time.Now().Add(-180*24*time.Hour))
	if err != nil {
		t.Fatalf("ListPostHistoryForUserSince: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(recent))
	}
	if recent[0].VideoID != v.ID || recent[0].PostID != post.ID || recent[0].UserID != user.ID {
		t.Errorf("unexpected history row: %+v", recent[0])
	}

	future, err := repo.ListPostHistoryForUserSince(user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("ListPostHistoryForUserSince(future): %v", err)
	}
	if len(future) != 0 {
		t.Errorf("expected 0 history rows after future `since`, got %d", len(future))
	}
}

// --- video_source_ingest ------------------------------------------------------

func TestVideoRepo_RecordAndGetIngest(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)

	if err := repo.RecordIngest("wimp.com", "cursor-page-12"); err != nil {
		t.Fatalf("RecordIngest: %v", err)
	}

	got, err := repo.GetIngest("wimp.com")
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if got == nil {
		t.Fatalf("expected ingest row, got nil")
	}
	if got.Source != "wimp.com" || got.LastCursor != "cursor-page-12" {
		t.Errorf("unexpected ingest row: %+v", got)
	}

	// Overwrite updates cursor.
	if err := repo.RecordIngest("wimp.com", "cursor-page-13"); err != nil {
		t.Fatalf("RecordIngest(update): %v", err)
	}
	got, err = repo.GetIngest("wimp.com")
	if err != nil {
		t.Fatalf("GetIngest(after update): %v", err)
	}
	if got.LastCursor != "cursor-page-13" {
		t.Errorf("expected cursor-page-13, got %q", got.LastCursor)
	}
}

// --- helpers ------------------------------------------------------------------

func float32abs(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}
