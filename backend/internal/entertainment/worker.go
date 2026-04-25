package entertainment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// TMDB response types.

type tmdbGenreList struct {
	Genres []tmdbGenre `json:"genres"`
}

type tmdbGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tmdbMovieList struct {
	Results []tmdbMovie `json:"results"`
}

type tmdbMovie struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Overview    string  `json:"overview"`
	PosterPath  string  `json:"poster_path"`
	GenreIDs    []int   `json:"genre_ids"`
	VoteAverage float64 `json:"vote_average"`
	ReleaseDate string  `json:"release_date"`
}

type tmdbTVList struct {
	Results []tmdbTV `json:"results"`
}

type tmdbTV struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"poster_path"`
	GenreIDs     []int   `json:"genre_ids"`
	VoteAverage  float64 `json:"vote_average"`
	FirstAirDate string  `json:"first_air_date"`
}

// Worker periodically fetches upcoming movies and on-air TV from TMDB
// and upserts them into the interest_calendar_events table.
type Worker struct {
	calendarRepo *repository.CalendarEventRepo
	baseURL      string
	apiKey       string
	region       string
	interval     time.Duration
	genres       map[int]string
}

// NewWorker creates a new entertainment Worker with a 24h polling interval.
func NewWorker(calendarRepo *repository.CalendarEventRepo, baseURL, apiKey, region string) *Worker {
	return &Worker{
		calendarRepo: calendarRepo,
		baseURL:      baseURL,
		apiKey:       apiKey,
		region:       region,
		interval:     24 * time.Hour,
		genres:       make(map[int]string),
	}
}

// Run starts the entertainment worker loop. It loads genres, runs an immediate
// first cycle, then repeats on the configured interval until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("entertainment worker started", "interval", w.interval)

	if err := w.loadGenres(ctx); err != nil {
		slog.Warn("entertainment worker: failed to load genres", "error", err)
	}

	w.cycle(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("entertainment worker stopped")
			return
		case <-ticker.C:
			// Reload genres on each cycle in case they change.
			if err := w.loadGenres(ctx); err != nil {
				slog.Warn("entertainment worker: failed to reload genres", "error", err)
			}
			w.cycle(ctx)
		}
	}
}

// loadGenres fetches the TMDB movie genre list and populates the genre map.
func (w *Worker) loadGenres(ctx context.Context) error {
	url := fmt.Sprintf("%s/3/genre/movie/list?api_key=%s", w.baseURL, w.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build genre list request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch genre list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("genre list: unexpected status %d", resp.StatusCode)
	}

	var list tmdbGenreList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return fmt.Errorf("decode genre list: %w", err)
	}

	genres := make(map[int]string, len(list.Genres))
	for _, g := range list.Genres {
		genres[g.ID] = g.Name
	}
	w.genres = genres
	slog.Debug("entertainment worker: genres loaded", "count", len(genres))
	return nil
}

// RunCycle loads genres and runs one ingest cycle. Useful for testing.
func (w *Worker) RunCycle(ctx context.Context) {
	if err := w.loadGenres(ctx); err != nil {
		slog.Warn("entertainment worker: failed to load genres", "error", err)
	}
	w.cycle(ctx)
}

// cycle fetches upcoming movies and on-air TV, then upserts each as a calendar event.
func (w *Worker) cycle(ctx context.Context) {
	var ok, failed int

	movies, err := w.fetchUpcomingMovies(ctx)
	if err != nil {
		slog.Error("entertainment worker: fetch movies failed", "error", err)
	} else {
		for _, m := range movies {
			if err := w.ingestMovie(m); err != nil {
				slog.Warn("entertainment worker: ingest movie failed", "id", m.ID, "title", m.Title, "error", err)
				failed++
			} else {
				ok++
			}
		}
	}

	shows, err := w.fetchOnAirTV(ctx)
	if err != nil {
		slog.Error("entertainment worker: fetch TV failed", "error", err)
	} else {
		for _, s := range shows {
			if err := w.ingestTV(s); err != nil {
				slog.Warn("entertainment worker: ingest TV failed", "id", s.ID, "name", s.Name, "error", err)
				failed++
			} else {
				ok++
			}
		}
	}

	slog.Info("entertainment worker: cycle complete", "movies", len(movies), "shows", len(shows), "ok", ok, "failed", failed)
}

// fetchUpcomingMovies calls GET /3/movie/upcoming.
func (w *Worker) fetchUpcomingMovies(ctx context.Context) ([]tmdbMovie, error) {
	url := fmt.Sprintf("%s/3/movie/upcoming?api_key=%s&region=%s", w.baseURL, w.apiKey, w.region)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build upcoming movies request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch upcoming movies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upcoming movies: unexpected status %d", resp.StatusCode)
	}

	var list tmdbMovieList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decode upcoming movies: %w", err)
	}
	return list.Results, nil
}

// fetchOnAirTV calls GET /3/tv/on_the_air.
func (w *Worker) fetchOnAirTV(ctx context.Context) ([]tmdbTV, error) {
	url := fmt.Sprintf("%s/3/tv/on_the_air?api_key=%s", w.baseURL, w.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build on-air TV request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch on-air TV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("on-air TV: unexpected status %d", resp.StatusCode)
	}

	var list tmdbTVList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decode on-air TV: %w", err)
	}
	return list.Results, nil
}

// ingestMovie converts a TMDB movie to an InterestCalendarEvent and upserts it.
func (w *Worker) ingestMovie(m tmdbMovie) error {
	eventKey := fmt.Sprintf("tmdb:movie:%d:release:%s", m.ID, w.region)

	tags := w.genreNames(m.GenreIDs)

	posterURL := ""
	if m.PosterPath != "" {
		posterURL = "https://image.tmdb.org/t/p/w500" + m.PosterPath
	}

	startTime, err := parseDate(m.ReleaseDate)
	if err != nil {
		// Fall back to now if date is missing/invalid.
		startTime = time.Now().UTC()
	}

	payload, err := json.Marshal(map[string]interface{}{
		"title":        m.Title,
		"overview":     m.Overview,
		"poster_url":   posterURL,
		"genres":       tags,
		"vote_average": m.VoteAverage,
		"release_date": m.ReleaseDate,
	})
	if err != nil {
		return fmt.Errorf("marshal movie payload: %w", err)
	}

	entityIDs, err := json.Marshal(map[string]interface{}{
		"tmdb_id": m.ID,
	})
	if err != nil {
		return fmt.Errorf("marshal movie entity_ids: %w", err)
	}

	event := model.InterestCalendarEvent{
		EventKey:     eventKey,
		Domain:       "entertainment",
		Title:        m.Title,
		StartTime:    startTime,
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "movie_release",
		EntityIDs:    entityIDs,
		InterestTags: tags,
		Payload:      payload,
	}

	return w.calendarRepo.Upsert(event)
}

// ingestTV converts a TMDB TV show to an InterestCalendarEvent and upserts it.
func (w *Worker) ingestTV(s tmdbTV) error {
	eventKey := fmt.Sprintf("tmdb:tv:%d:premiere:%s", s.ID, w.region)

	tags := w.genreNames(s.GenreIDs)

	posterURL := ""
	if s.PosterPath != "" {
		posterURL = "https://image.tmdb.org/t/p/w500" + s.PosterPath
	}

	startTime, err := parseDate(s.FirstAirDate)
	if err != nil {
		startTime = time.Now().UTC()
	}

	payload, err := json.Marshal(map[string]interface{}{
		"title":          s.Name,
		"overview":       s.Overview,
		"poster_url":     posterURL,
		"genres":         tags,
		"vote_average":   s.VoteAverage,
		"first_air_date": s.FirstAirDate,
	})
	if err != nil {
		return fmt.Errorf("marshal TV payload: %w", err)
	}

	entityIDs, err := json.Marshal(map[string]interface{}{
		"tmdb_id": s.ID,
	})
	if err != nil {
		return fmt.Errorf("marshal TV entity_ids: %w", err)
	}

	event := model.InterestCalendarEvent{
		EventKey:     eventKey,
		Domain:       "entertainment",
		Title:        s.Name,
		StartTime:    startTime,
		Timezone:     "UTC",
		Status:       "scheduled",
		EntityType:   "tv_premiere",
		EntityIDs:    entityIDs,
		InterestTags: tags,
		Payload:      payload,
	}

	return w.calendarRepo.Upsert(event)
}

// genreNames resolves genre IDs to lowercase genre name strings.
func (w *Worker) genreNames(ids []int) []string {
	var names []string
	for _, id := range ids {
		if name, ok := w.genres[id]; ok {
			names = append(names, strings.ToLower(name))
		}
	}
	return names
}

// parseDate parses a TMDB date string (YYYY-MM-DD) into a UTC time.Time.
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date %q: %w", s, err)
	}
	return t.UTC(), nil
}
