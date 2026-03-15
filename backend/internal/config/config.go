package config

import "os"

type Config struct {
	Port                   string
	DatabasePath           string
	FirebaseProjectID      string
	FirebaseCredentialsFile string
}

func Load() Config {
	return Config{
		Port:              envOr("PORT", "8080"),
		DatabasePath:      envOr("DATABASE_PATH", "beepbopboop.db"),
		FirebaseProjectID:      os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseCredentialsFile: os.Getenv("FIREBASE_CREDENTIALS_FILE"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
