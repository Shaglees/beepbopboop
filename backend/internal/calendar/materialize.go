package calendar

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

// MaterializeWorker generates personalized feed posts from interest calendar events.
// It runs on a configurable interval, scanning upcoming events for each user that
// has declared interests and publishing posts at appropriate time windows.
type MaterializeWorker struct {
	calendarRepo *repository.CalendarEventRepo
	postRepo     *repository.PostRepo
	userRepo     *repository.UserRepo
	interestRepo *repository.UserInterestRepo
	agentID      string
	interval     time.Duration
}

// NewMaterializeWorker creates a new MaterializeWorker.
func NewMaterializeWorker(
	calendarRepo *repository.CalendarEventRepo,
	postRepo *repository.PostRepo,
	userRepo *repository.UserRepo,
	interestRepo *repository.UserInterestRepo,
	agentID string,
) *MaterializeWorker {
	return &MaterializeWorker{
		calendarRepo: calendarRepo,
		postRepo:     postRepo,
		userRepo:     userRepo,
		interestRepo: interestRepo,
		agentID:      agentID,
		interval:     15 * time.Minute,
	}
}

// Run starts the materialization worker loop. It runs an immediate first cycle then
// repeats on the configured interval until ctx is cancelled.
func (w *MaterializeWorker) Run(ctx context.Context) {
	slog.Info("materialize worker started", "interval", w.interval)
	w.cycleOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("materialize worker stopped")
			return
		case <-ticker.C:
			w.cycleOnce(ctx)
		}
	}
}

// cycleOnce runs one full materialization pass across all users with interests.
func (w *MaterializeWorker) cycleOnce(ctx context.Context) {
	userIDs, err := w.interestRepo.UsersWithInterests()
	if err != nil {
		slog.Error("materialize worker: failed to list users with interests", "error", err)
		return
	}
	if len(userIDs) == 0 {
		slog.Debug("materialize worker: no users with interests, skipping")
		return
	}

	now := time.Now()
	var ok, failed int

	for _, userID := range userIDs {
		if ctx.Err() != nil {
			return
		}
		if err := w.processUser(userID, now); err != nil {
			slog.Warn("materialize worker: user failed", "user_id", userID, "error", err)
			failed++
		} else {
			ok++
		}
	}

	slog.Info("materialize worker: cycle complete",
		"users", len(userIDs), "ok", ok, "failed", failed)
}

// processUser materializes posts for a single user.
func (w *MaterializeWorker) processUser(userID string, now time.Time) error {
	interests, err := w.interestRepo.ListActive(userID)
	if err != nil {
		return fmt.Errorf("list active interests: %w", err)
	}
	if len(interests) == 0 {
		return nil
	}

	tags := interestTags(interests)

	// Query matching events in the next 24h window. Sports preview looks back 24h
	// from now to catch events in T-24h..T-12h; entertainment preview looks ahead 7d.
	// Use a wide window here so applicableWindows can filter precisely.
	from := now.Add(-24 * time.Hour)
	to := now.Add(7 * 24 * time.Hour)

	events, err := w.calendarRepo.ForUser(userID, tags, from, to)
	if err != nil {
		return fmt.Errorf("get calendar events for user: %w", err)
	}

	for _, event := range events {
		windows := w.applicableWindows(event, now)
		for _, window := range windows {
			if err := w.materializePost(userID, event, window, now); err != nil {
				slog.Warn("materialize worker: post failed",
					"user_id", userID,
					"event_key", event.EventKey,
					"window", window,
					"error", err,
				)
			}
		}
	}

	return nil
}

// materializePost renders and creates a post for one event/user/window, if not already done.
func (w *MaterializeWorker) materializePost(userID string, event model.InterestCalendarEvent, window string, now time.Time) error {
	published, err := w.calendarRepo.IsPublished(event.EventKey, userID, window)
	if err != nil {
		return fmt.Errorf("check published: %w", err)
	}
	if published {
		return nil
	}

	content, err := w.renderContent(event, window, now)
	if err != nil {
		return fmt.Errorf("render content: %w", err)
	}

	post, err := w.postRepo.Create(repository.CreatePostParams{
		AgentID:     w.agentID,
		UserID:      userID,
		Title:       content.Title,
		Body:        content.Body,
		PostType:    "discovery",
		Visibility:  "personal",
		DisplayHint: content.DisplayHint,
		ExternalURL: content.ExternalURL,
	})
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}

	if err := w.calendarRepo.LogPost(event.EventKey, userID, window, post.ID); err != nil {
		// Non-fatal: post was created, dedup log may be missing but we can retry safely
		// due to ON CONFLICT DO NOTHING in LogPost.
		slog.Warn("materialize worker: log post failed",
			"event_key", event.EventKey,
			"user_id", userID,
			"post_id", post.ID,
			"error", err,
		)
	}

	slog.Info("materialize worker: post created",
		"event_key", event.EventKey,
		"user_id", userID,
		"window", window,
		"post_id", post.ID,
	)

	return nil
}

// renderContent selects the appropriate rendering function based on event domain.
func (w *MaterializeWorker) renderContent(event model.InterestCalendarEvent, window string, now time.Time) (*postContent, error) {
	switch event.Domain {
	case "sports":
		return renderSportsPost(event.Payload, event.StartTime, window)
	case "entertainment":
		return renderEntertainmentPost(event.Payload, event.StartTime, window)
	default:
		// Fallback for unknown domains: produce a minimal card post.
		return &postContent{
			Title:       event.Title,
			Body:        fmt.Sprintf("Coming up: %s", event.StartTime.Format("Mon Jan 2 at 3:04 PM")),
			DisplayHint: "card",
			ExternalURL: string(event.Payload),
		}, nil
	}
}

// applicableWindows returns the list of window names that apply to this event at the given time.
//
// Sports windows:
//   - "preview":  T-24h to T-12h before start
//   - "imminent": T-2h to T+0 (up to start time)
//
// Entertainment windows:
//   - "preview":     T-7d to T-3d before start
//   - "release_day": T-24h to T+24h around start
func (w *MaterializeWorker) applicableWindows(event model.InterestCalendarEvent, now time.Time) []string {
	start := event.StartTime
	var windows []string

	switch event.Domain {
	case "sports":
		// Preview: T-24h <= now < T-12h
		if !now.Before(start.Add(-24*time.Hour)) && now.Before(start.Add(-12*time.Hour)) {
			windows = append(windows, "preview")
		}
		// Imminent: T-2h <= now <= T
		if !now.Before(start.Add(-2*time.Hour)) && !now.After(start) {
			windows = append(windows, "imminent")
		}

	case "entertainment":
		// Preview: T-7d <= now < T-3d
		if !now.Before(start.Add(-7*24*time.Hour)) && now.Before(start.Add(-3*24*time.Hour)) {
			windows = append(windows, "preview")
		}
		// Release day: T-24h <= now <= T+24h
		if !now.Before(start.Add(-24*time.Hour)) && !now.After(start.Add(24*time.Hour)) {
			windows = append(windows, "release_day")
		}

	default:
		// For unknown domains, use a generic "day" window (T-24h to T)
		if !now.Before(start.Add(-24*time.Hour)) && !now.After(start) {
			windows = append(windows, "day")
		}
	}

	return windows
}

// interestTags extracts topic strings from a list of UserInterest records.
func interestTags(interests []model.UserInterest) []string {
	tags := make([]string, 0, len(interests))
	for _, i := range interests {
		tags = append(tags, i.Topic)
	}
	return tags
}
