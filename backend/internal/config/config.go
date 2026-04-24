package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                    string
	DatabaseURL             string
	FirebaseProjectID       string
	FirebaseCredentialsFile string
	// RankerModelPath is the JSON checkpoint from ml/train.py export (two-tower).
	// Empty disables ML scoring in the ForYou feed.
	RankerModelPath string
	// MLRankBlend is the weight on the learned score vs normalised rule-based score in [0,1]
	// (same meaning as ranking.RankerConfig.MLWeight).
	MLRankBlend float64

	EmbeddingProvider             string
	EmbeddingFallbackProvider     string
	EmbeddingModel                string
	EmbeddingOutputDimensionality int
	EmbeddingAllowImageURLParts   bool
	GoogleAPIKey                  string
}

func Load() Config {
	return Config{
		Port:                          envOr("PORT", "8080"),
		DatabaseURL:                   envOr("DATABASE_URL", "postgres://beepbopboop:beepbopboop@localhost:5432/beepbopboop?sslmode=disable"),
		FirebaseProjectID:             os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseCredentialsFile:       os.Getenv("FIREBASE_CREDENTIALS_FILE"),
		RankerModelPath:               os.Getenv("RANKER_MODEL_PATH"),
		MLRankBlend:                   envFloat("ML_RANK_BLEND", 0.35),
		EmbeddingProvider:             envOr("EMBEDDING_PROVIDER", "gemini"),
		EmbeddingFallbackProvider:     envOr("EMBEDDING_FALLBACK_PROVIDER", "hash"),
		EmbeddingModel:                envOr("EMBEDDING_MODEL", "gemini-embedding-002"),
		EmbeddingOutputDimensionality: envInt("EMBEDDING_OUTPUT_DIM", 1536),
		EmbeddingAllowImageURLParts:   envBool("EMBEDDING_ALLOW_IMAGE_URL_PARTS", false),
		GoogleAPIKey:                  os.Getenv("GOOGLE_API_KEY"),
	}
}

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
