package ranking

import "fmt"

// StubRanker is a fast scorer for latency and shape tests (issue #44 TDD).
// It does not load checkpoints; scores are in [0, 1] without file I/O.
type StubRanker struct {
	dim int
}

// NewStubRanker returns a stub with the given input vector dimension.
func NewStubRanker(dim int) *StubRanker {
	if dim < 1 {
		dim = 1
	}
	return &StubRanker{dim: dim}
}

// InputDim returns the configured dimension.
func (s *StubRanker) InputDim() int { return s.dim }

// ScoreBatch returns one score per post vector. Wrong-dimension post rows score 0.
func (s *StubRanker) ScoreBatch(userVec []float32, postVecs [][]float32) ([]float32, error) {
	out := make([]float32, len(postVecs))
	if len(userVec) != s.dim {
		return nil, fmt.Errorf("stub ranker: user vec dim %d want %d", len(userVec), s.dim)
	}
	for i, pv := range postVecs {
		if len(pv) != s.dim {
			out[i] = 0
			continue
		}
		// Deterministic [0,1) — cheap hot loop for latency tests.
		out[i] = float32((i*17 + len(pv)*3) % 1000) / 1000.0
		if out[i] >= 1 {
			out[i] = 0.999
		}
	}
	return out, nil
}
