package embedding

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sync"

	"github.com/lib/pq"
)

// interestToLabels maps high-level onboarding interest names to the granular
// post labels used in the prototype computation. The first label in each slice
// is used when looking up a prototype (most representative label for the interest).
var interestToLabels = map[string][]string{
	"Sports":  {"sports", "basketball", "football", "baseball", "nfl", "nba"},
	"Fashion": {"fashion", "outfit", "style", "trending"},
	"Local":   {"event", "discovery", "place"},
	"News":    {"article", "hacker-news", "technology"},
	"Weather": {"weather"},
	"Music":   {"music"},
	"Food":    {"food"},
}

// PrototypeStore holds precomputed normalized embedding prototypes for each
// post label. Prototypes are the L2-normalised average of post embeddings for
// that label. Used by the cold-start path to seed new user vectors.
type PrototypeStore struct {
	db      *sql.DB
	vectors map[string][]float32
	mu      sync.RWMutex
}

func NewPrototypeStore(db *sql.DB) *PrototypeStore {
	return &PrototypeStore{db: db, vectors: make(map[string][]float32)}
}

// IsZero reports whether every element of v is zero (or v is empty).
func IsZero(v []float32) bool {
	for _, f := range v {
		if f != 0 {
			return false
		}
	}
	return true
}

// Compute queries post_embeddings grouped by label, averages element-wise,
// L2-normalises, and caches the result in memory. Safe to call concurrently;
// called on startup and on a nightly schedule.
func (ps *PrototypeStore) Compute(ctx context.Context) error {
	rows, err := ps.db.QueryContext(ctx, `
		SELECT label_val, pe.embedding
		FROM post_embeddings pe
		JOIN posts p ON p.id = pe.post_id
		CROSS JOIN LATERAL jsonb_array_elements_text(p.labels::jsonb) AS label_val
		WHERE p.status = 'published'
		  AND p.labels IS NOT NULL
		  AND p.labels != 'null'
		  AND p.created_at > NOW() - INTERVAL '30 days'`)
	if err != nil {
		return fmt.Errorf("prototype compute: %w", err)
	}
	defer rows.Close()

	sums := make(map[string][]float64)
	counts := make(map[string]int)

	for rows.Next() {
		var label string
		var f64 pq.Float64Array
		if err := rows.Scan(&label, &f64); err != nil {
			return fmt.Errorf("scan prototype row: %w", err)
		}
		if _, ok := sums[label]; !ok {
			sums[label] = make([]float64, len(f64))
		}
		for i, v := range f64 {
			sums[label][i] += v
		}
		counts[label]++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate prototype rows: %w", err)
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.vectors = make(map[string][]float32, len(sums))
	for label, sum := range sums {
		n := float64(counts[label])
		avg := make([]float64, len(sum))
		for i, v := range sum {
			avg[i] = v / n
		}
		ps.vectors[label] = l2normalize(avg)
	}
	return nil
}

// VectorFor returns the precomputed prototype for a label.
// Returns (nil, false) when no prototype exists for that label.
func (ps *PrototypeStore) VectorFor(label string) ([]float32, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	v, ok := ps.vectors[label]
	return v, ok
}

// CombineFor maps interest names (e.g. "Sports", "Fashion") to their primary
// label prototype, sums the matching vectors, and L2-normalises the result.
// Unknown interests and interests without a computed prototype are silently
// skipped. Returns an empty (zero) slice when no matching prototypes exist.
func (ps *PrototypeStore) CombineFor(interests []string) []float32 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var sum []float64
	count := 0
	for _, interest := range interests {
		labels, ok := interestToLabels[interest]
		if !ok {
			continue
		}
		// Use the first label prototype that exists for this interest.
		for _, label := range labels {
			if vec, ok := ps.vectors[label]; ok {
				if sum == nil {
					sum = make([]float64, len(vec))
				}
				for i, v := range vec {
					sum[i] += float64(v)
				}
				count++
				break
			}
		}
	}
	if count == 0 || sum == nil {
		return []float32{}
	}
	return l2normalize(sum)
}

// PopularityFallback returns the average embedding of the top-50 most-saved
// published posts in the last 7 days, L2-normalised. Returns an empty slice
// when no eligible posts exist — callers should treat this as "no signal".
func (ps *PrototypeStore) PopularityFallback(ctx context.Context) ([]float32, error) {
	rows, err := ps.db.QueryContext(ctx, `
		SELECT pe.embedding
		FROM post_embeddings pe
		JOIN posts p ON p.id = pe.post_id
		WHERE p.status = 'published'
		  AND p.save_count > 0
		  AND p.created_at > NOW() - INTERVAL '7 days'
		ORDER BY p.save_count DESC
		LIMIT 50`)
	if err != nil {
		return nil, fmt.Errorf("popularity fallback: %w", err)
	}
	defer rows.Close()

	var sum []float64
	count := 0
	for rows.Next() {
		var f64 pq.Float64Array
		if err := rows.Scan(&f64); err != nil {
			return nil, fmt.Errorf("scan fallback row: %w", err)
		}
		if sum == nil {
			sum = make([]float64, len(f64))
		}
		for i, v := range f64 {
			sum[i] += v
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fallback rows: %w", err)
	}
	if count == 0 || sum == nil {
		return []float32{}, nil
	}
	return l2normalize(sum), nil
}

// l2normalize divides v by its L2 norm and returns a []float32. Returns a
// zero slice of the same length when the magnitude is below 1e-10.
func l2normalize(v []float64) []float32 {
	var mag float64
	for _, x := range v {
		mag += x * x
	}
	mag = math.Sqrt(mag)
	result := make([]float32, len(v))
	if mag < 1e-10 {
		return result
	}
	for i, x := range v {
		result[i] = float32(x / mag)
	}
	return result
}
