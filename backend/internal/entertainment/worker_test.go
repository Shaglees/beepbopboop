package entertainment_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
	"github.com/shanegleeson/beepbopboop/backend/internal/entertainment"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

func TestWorker_Cycle_IngestsMovie(t *testing.T) {
	// Set up a mock TMDB server.
	mux := http.NewServeMux()

	mux.HandleFunc("/3/genre/movie/list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"genres": []map[string]interface{}{
				{"id": 28, "name": "Action"},
				{"id": 12, "name": "Adventure"},
				{"id": 878, "name": "Science Fiction"},
			},
		})
	})

	mux.HandleFunc("/3/movie/upcoming", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		releaseDate := time.Now().UTC().Add(30 * 24 * time.Hour).Format("2006-01-02")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"id":           12345,
					"title":        "Galactic Odyssey",
					"overview":     "A thrilling space adventure.",
					"poster_path":  "/poster.jpg",
					"genre_ids":    []int{12, 878},
					"vote_average": 7.8,
					"release_date": releaseDate,
				},
			},
		})
	})

	mux.HandleFunc("/3/tv/on_the_air", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Set up a real test DB and repo.
	db := database.OpenTestDB(t)
	repo := repository.NewCalendarEventRepo(db)

	// Create the worker pointing at the mock server.
	worker := entertainment.NewWorker(repo, server.URL, "test-api-key", "US")

	ctx := context.Background()

	// Run the worker cycle (not the full Run loop).
	worker.RunCycle(ctx)

	// Verify the event was upserted.
	from := time.Now().UTC().Add(-time.Hour)
	to := time.Now().UTC().Add(60 * 24 * time.Hour)
	events, err := repo.Upcoming("entertainment", from, to)
	if err != nil {
		t.Fatalf("Upcoming: %v", err)
	}

	var found *struct {
		domain     string
		entityType string
		title      string
	}
	for _, e := range events {
		if e.EventKey == "tmdb:movie:12345:release:US" {
			found = &struct {
				domain     string
				entityType string
				title      string
			}{
				domain:     e.Domain,
				entityType: e.EntityType,
				title:      e.Title,
			}
			break
		}
	}

	if found == nil {
		t.Fatal("expected to find event with key tmdb:movie:12345:release:US, but it was not found")
	}
	if found.domain != "entertainment" {
		t.Errorf("domain = %q, want %q", found.domain, "entertainment")
	}
	if found.entityType != "movie_release" {
		t.Errorf("entity_type = %q, want %q", found.entityType, "movie_release")
	}
	if found.title != "Galactic Odyssey" {
		t.Errorf("title = %q, want %q", found.title, "Galactic Odyssey")
	}
}
