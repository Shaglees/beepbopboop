package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// TestVideoRepo_ListCatalog_Filters covers the contract the /videos HTTP
// handler depends on: label include/exclude, provider whitelist, healthy-only
// toggle, and published_at-desc ordering.
func TestVideoRepo_ListCatalog_Filters(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)
	ctx := context.Background()

	seed := func(key, provider string, labels []string, health string, published time.Time) model.Video {
		v, err := repo.UpsertCatalog(model.Video{
			Provider:        provider,
			ProviderVideoID: key,
			WatchURL:        "https://example.test/watch/" + key,
			EmbedURL:        "https://example.test/embed/" + key,
			Title:           "Title " + key,
			Labels:          labels,
			PublishedAt:     &published,
			EmbedHealth:     health,
		})
		if err != nil {
			t.Fatalf("seed %s: %v", key, err)
		}
		return v
	}

	now := time.Now().UTC()
	newest := seed("newest", "youtube", []string{"dogs", "funny"}, "ok", now)
	middle := seed("middle", "vimeo", []string{"birds", "nature"}, "ok", now.Add(-24*time.Hour))
	oldest := seed("oldest", "youtube", []string{"music"}, "ok", now.Add(-48*time.Hour))
	dead := seed("dead", "youtube", []string{"dogs"}, "dead", now.Add(-1*time.Hour))
	_ = dead

	// Default list: no filters, healthy_only=true (dead rows excluded).
	got, err := repo.ListCatalog(ctx, repository.VideoCatalogListParams{HealthyOnly: true})
	if err != nil {
		t.Fatalf("ListCatalog default: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 healthy rows, got %d", len(got))
	}
	// Must come back published_at desc.
	if got[0].ID != newest.ID || got[1].ID != middle.ID || got[2].ID != oldest.ID {
		t.Errorf("unexpected ordering: %v", []string{got[0].ID, got[1].ID, got[2].ID})
	}

	// Include label filter: only "dogs".
	got, err = repo.ListCatalog(ctx, repository.VideoCatalogListParams{
		HealthyOnly:   true,
		IncludeLabels: []string{"dogs"},
	})
	if err != nil {
		t.Fatalf("ListCatalog include: %v", err)
	}
	if len(got) != 1 || got[0].ID != newest.ID {
		t.Errorf("expected only 'newest' for dogs filter, got %v", got)
	}

	// Exclude label filter: anything tagged "music".
	got, err = repo.ListCatalog(ctx, repository.VideoCatalogListParams{
		HealthyOnly:   true,
		ExcludeLabels: []string{"music"},
	})
	if err != nil {
		t.Fatalf("ListCatalog exclude: %v", err)
	}
	for _, v := range got {
		if v.ID == oldest.ID {
			t.Errorf("oldest (music) should have been excluded")
		}
	}

	// Provider whitelist: vimeo only.
	got, err = repo.ListCatalog(ctx, repository.VideoCatalogListParams{
		HealthyOnly: true,
		Providers:   []string{"vimeo"},
	})
	if err != nil {
		t.Fatalf("ListCatalog providers: %v", err)
	}
	if len(got) != 1 || got[0].ID != middle.ID {
		t.Errorf("expected only vimeo (middle), got %v", got)
	}

	// healthy_only=false still excludes 'dead' rows (per contract).
	got, err = repo.ListCatalog(ctx, repository.VideoCatalogListParams{HealthyOnly: false})
	if err != nil {
		t.Fatalf("ListCatalog all-health: %v", err)
	}
	for _, v := range got {
		if v.EmbedHealth == "dead" {
			t.Errorf("dead row leaked into results: %s", v.ID)
		}
	}
}

func TestVideoRepo_ListCatalog_LimitClamp(t *testing.T) {
	db := database.OpenTestDB(t)
	repo := repository.NewVideoRepo(db)
	ctx := context.Background()

	// Seed 3 healthy rows.
	for i, id := range []string{"a", "b", "c"} {
		_, err := repo.UpsertCatalog(model.Video{
			Provider:        "youtube",
			ProviderVideoID: id,
			WatchURL:        "https://x/" + id,
			EmbedURL:        "https://x/e/" + id,
			PublishedAt:     ptrTime(time.Now().Add(-time.Duration(i) * time.Hour)),
			EmbedHealth:     "ok",
		})
		if err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
	}

	// limit=0 should default to 20, not 0.
	got, _ := repo.ListCatalog(ctx, repository.VideoCatalogListParams{Limit: 0, HealthyOnly: true})
	if len(got) != 3 {
		t.Errorf("expected 3 rows under default limit, got %d", len(got))
	}

	// limit=1 returns only the newest.
	got, _ = repo.ListCatalog(ctx, repository.VideoCatalogListParams{Limit: 1, HealthyOnly: true})
	if len(got) != 1 {
		t.Errorf("expected 1 row under limit=1, got %d", len(got))
	}
}
