package ranking

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
)

// checkpoint is the JSON serialisation of a trained two-tower model.
// user_weights and post_weights are [repr_dim][input_dim] projection matrices.
type checkpoint struct {
	InputDim    int         `json:"input_dim"`
	ReprDim     int         `json:"repr_dim"`
	UserWeights [][]float32 `json:"user_weights"`
	PostWeights [][]float32 `json:"post_weights"`
}

// Ranker scores (user, post) embedding pairs using a two-tower dot-product model.
// Each tower is a single learned linear projection followed by L2 normalisation.
// The dot product of the two unit-norm representations is mapped to [0, 1].
//
// For inference at scale:
//   - user repr is computed once per request
//   - post reprs are cached and reused across users
type Ranker struct {
	userW    [][]float32
	postW    [][]float32
	inputDim int
	reprDim  int
}

// NewRanker loads a checkpoint from the JSON file at path and returns a ready
// Ranker. Returns a descriptive error if the file is missing or malformed.
func NewRanker(path string) (*Ranker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ranker: read %s: %w", path, err)
	}
	var cp checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("ranker: unmarshal checkpoint: %w", err)
	}
	if err := validateCheckpoint(&cp); err != nil {
		return nil, fmt.Errorf("ranker: invalid checkpoint: %w", err)
	}
	return &Ranker{
		userW:    cp.UserWeights,
		postW:    cp.PostWeights,
		inputDim: cp.InputDim,
		reprDim:  cp.ReprDim,
	}, nil
}

// InputDim returns the expected dimension of user and post input vectors.
func (r *Ranker) InputDim() int { return r.inputDim }

// Score returns a relevance score in [0, 1] for a single (user, post) pair.
// Returns 0.0 (not an error) if either vector contains NaN so that callers
// can safely use the score in feed ranking without propagating NaN.
// Returns an error if either vector has the wrong dimension.
func (r *Ranker) Score(userVec, postVec []float32) (float32, error) {
	if containsNaN(userVec) || containsNaN(postVec) {
		slog.Warn("ranker: NaN in input vector, returning 0.0")
		return 0.0, nil
	}
	if err := r.checkDims(len(userVec), len(postVec)); err != nil {
		return 0, err
	}
	uRepr := l2norm(project(r.userW, userVec))
	pRepr := l2norm(project(r.postW, postVec))
	return clamp01((dot(uRepr, pRepr) + 1) / 2), nil
}

// ScoreBatch scores one user against multiple post candidates in a single call.
// The user vector is projected once; post vectors are projected individually.
// Returns one score per candidate in the same order as postVecs.
// NaN candidates receive score 0.0. Wrong-dimension candidates receive score 0.0.
// Returns an error only if userVec has the wrong dimension.
func (r *Ranker) ScoreBatch(userVec []float32, postVecs [][]float32) ([]float32, error) {
	scores := make([]float32, len(postVecs))
	if len(postVecs) == 0 {
		return scores, nil
	}
	if containsNaN(userVec) {
		slog.Warn("ranker: NaN in user vector, returning all-zero batch scores")
		return scores, nil
	}
	if len(userVec) != r.inputDim {
		return nil, fmt.Errorf("ranker: user vec dim %d != input_dim %d",
			len(userVec), r.inputDim)
	}
	uRepr := l2norm(project(r.userW, userVec))
	for i, pv := range postVecs {
		if containsNaN(pv) || len(pv) != r.inputDim {
			scores[i] = 0.0
			continue
		}
		pRepr := l2norm(project(r.postW, pv))
		scores[i] = clamp01((dot(uRepr, pRepr) + 1) / 2)
	}
	return scores, nil
}

// ProjectUser returns the L2-normalised representation for a user vector.
// Pre-computing this once per request and reusing it with ScoreBatchFromRepr
// avoids redundant projection work across many candidates.
func (r *Ranker) ProjectUser(userVec []float32) ([]float32, error) {
	if containsNaN(userVec) {
		return make([]float32, r.reprDim), nil
	}
	if len(userVec) != r.inputDim {
		return nil, fmt.Errorf("ranker: user vec dim %d != input_dim %d",
			len(userVec), r.inputDim)
	}
	return l2norm(project(r.userW, userVec)), nil
}

// ProjectPost returns the L2-normalised representation for a post vector.
// Cache this per post to avoid recomputing across user requests.
func (r *Ranker) ProjectPost(postVec []float32) ([]float32, error) {
	if containsNaN(postVec) {
		return make([]float32, r.reprDim), nil
	}
	if len(postVec) != r.inputDim {
		return nil, fmt.Errorf("ranker: post vec dim %d != input_dim %d",
			len(postVec), r.inputDim)
	}
	return l2norm(project(r.postW, postVec)), nil
}

// ScoreBatchFromReprs scores a pre-projected user repr against pre-projected post
// reprs. This is the fast path for feed ranking when post reprs are cached.
func (r *Ranker) ScoreBatchFromReprs(userRepr []float32, postReprs [][]float32) []float32 {
	scores := make([]float32, len(postReprs))
	for i, pr := range postReprs {
		scores[i] = clamp01((dot(userRepr, pr) + 1) / 2)
	}
	return scores
}

// --- helpers ---

func validateCheckpoint(cp *checkpoint) error {
	if cp.InputDim <= 0 || cp.ReprDim <= 0 {
		return fmt.Errorf("input_dim and repr_dim must be positive")
	}
	if len(cp.UserWeights) != cp.ReprDim {
		return fmt.Errorf("user_weights rows %d != repr_dim %d",
			len(cp.UserWeights), cp.ReprDim)
	}
	if len(cp.PostWeights) != cp.ReprDim {
		return fmt.Errorf("post_weights rows %d != repr_dim %d",
			len(cp.PostWeights), cp.ReprDim)
	}
	for i, row := range cp.UserWeights {
		if len(row) != cp.InputDim {
			return fmt.Errorf("user_weights[%d] has %d cols, want %d",
				i, len(row), cp.InputDim)
		}
	}
	for i, row := range cp.PostWeights {
		if len(row) != cp.InputDim {
			return fmt.Errorf("post_weights[%d] has %d cols, want %d",
				i, len(row), cp.InputDim)
		}
	}
	return nil
}

func (r *Ranker) checkDims(userLen, postLen int) error {
	if userLen != r.inputDim || postLen != r.inputDim {
		return fmt.Errorf("ranker: expected input_dim=%d, got user=%d post=%d",
			r.inputDim, userLen, postLen)
	}
	return nil
}

// project computes the matrix-vector product w @ v.
func project(w [][]float32, v []float32) []float32 {
	out := make([]float32, len(w))
	for i, row := range w {
		var s float32
		for j, wij := range row {
			s += wij * v[j]
		}
		out[i] = s
	}
	return out
}

func l2norm(v []float32) []float32 {
	var mag float64
	for _, x := range v {
		mag += float64(x) * float64(x)
	}
	mag = math.Sqrt(mag)
	out := make([]float32, len(v))
	if mag < 1e-10 {
		return out
	}
	for i, x := range v {
		out[i] = float32(float64(x) / mag)
	}
	return out
}

func dot(a, b []float32) float32 {
	var s float32
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func clamp01(x float32) float32 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func containsNaN(v []float32) bool {
	for _, x := range v {
		if math.IsNaN(float64(x)) {
			return true
		}
	}
	return false
}
