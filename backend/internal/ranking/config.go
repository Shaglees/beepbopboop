package ranking

// RankerConfig holds tunables for hybrid ML + rule ForYou ranking (issue #44 refactor).
// MLWeight is the same concept as ML_RANK_BLEND in server config: weight on the
// learned score after per-batch normalisation of rule scores [0, 1].
type RankerConfig struct {
	MLWeight float64
}

// DefaultRankerConfig matches server default ML_RANK_BLEND.
func DefaultRankerConfig() RankerConfig {
	return RankerConfig{MLWeight: 0.35}
}
