package config

import "os"

type Config struct {
	Port                    string
	DatabaseURL             string
	FirebaseProjectID       string
	FirebaseCredentialsFile string
	CreatorsAPIKey          string // optional: Serp API key for richer creator search results
}

func Load() Config {
	return Config{
		Port:                    envOr("PORT", "8080"),
		DatabaseURL:             envOr("DATABASE_URL", "postgres://beepbopboop:beepbopboop@localhost:5432/beepbopboop?sslmode=disable"),
		FirebaseProjectID:       os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseCredentialsFile: os.Getenv("FIREBASE_CREDENTIALS_FILE"),
		CreatorsAPIKey:          os.Getenv("CREATORS_API_KEY"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
