package video

import (
	"context"

	"github.com/shanegleeson/beepbopboop/backend/internal/model"
	"github.com/shanegleeson/beepbopboop/backend/internal/repository"
)

const defaultDedupWindowDays = 180

// SelectOptions configures an on-demand historical video pick.
type SelectOptions struct {
	UserID        string
	Limit         int
	Seed          *int64
	IncludeLabels []string
	ExcludeLabels []string
}

// SelectionDiagnostics describes how the selector ran so callers can handle
// low-inventory cases gracefully and log useful traces.
type SelectionDiagnostics struct {
	RequestedLimit   int
	ReturnedCount    int
	HadUserEmbedding bool
	IncludeLabels    []string
	ExcludeLabels    []string
	DedupWindowDays  int
}

// SelectionResult is ready for post generation: the caller can choose a video
// and map its fields into a `video_embed` post payload.
type SelectionResult struct {
	Videos      []model.Video
	Diagnostics SelectionDiagnostics
}

// Selector composes the video catalog repo with the user embedding repo.
type Selector struct {
	videoRepo         *repository.VideoRepo
	userEmbeddingRepo *repository.UserEmbeddingRepo
}

func NewSelector(videoRepo *repository.VideoRepo, userEmbeddingRepo *repository.UserEmbeddingRepo) *Selector {
	return &Selector{
		videoRepo:         videoRepo,
		userEmbeddingRepo: userEmbeddingRepo,
	}
}

// Select chooses candidate videos using hard filters first (dedup, labels,
// embed health), then ranking by user-embedding similarity when present,
// freshness, and exploration.
func (s *Selector) Select(ctx context.Context, opts SelectOptions) (SelectionResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 1
	}

	var userEmbedding []float32
	if s.userEmbeddingRepo != nil {
		if ue, err := s.userEmbeddingRepo.Get(ctx, opts.UserID); err != nil {
			return SelectionResult{}, err
		} else if ue != nil {
			userEmbedding = ue.Embedding
		}
	}

	videos, err := s.videoRepo.SelectCandidates(ctx, repository.VideoSelectionParams{
		UserID:          opts.UserID,
		Limit:           opts.Limit,
		DedupWindowDays: defaultDedupWindowDays,
		IncludeLabels:   opts.IncludeLabels,
		ExcludeLabels:   opts.ExcludeLabels,
		Seed:            opts.Seed,
		UserEmbedding:   userEmbedding,
	})
	if err != nil {
		return SelectionResult{}, err
	}

	return SelectionResult{
		Videos: videos,
		Diagnostics: SelectionDiagnostics{
			RequestedLimit:   opts.Limit,
			ReturnedCount:    len(videos),
			HadUserEmbedding: len(userEmbedding) > 0,
			IncludeLabels:    append([]string(nil), opts.IncludeLabels...),
			ExcludeLabels:    append([]string(nil), opts.ExcludeLabels...),
			DedupWindowDays:  defaultDedupWindowDays,
		},
	}, nil
}
