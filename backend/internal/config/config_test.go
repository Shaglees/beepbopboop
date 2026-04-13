package config_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("FIREBASE_PROJECT_ID", "")
	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://beepbopboop:beepbopboop@localhost:5432/beepbopboop?sslmode=disable" {
		t.Errorf("expected default database url, got %s", cfg.DatabaseURL)
	}
	if cfg.FirebaseProjectID != "" {
		t.Errorf("expected empty firebase project id, got %s", cfg.FirebaseProjectID)
	}
}

func TestLoadFirebaseCredentials(t *testing.T) {
	t.Setenv("FIREBASE_CREDENTIALS_FILE", "/tmp/creds.json")

	cfg := config.Load()
	if cfg.FirebaseCredentialsFile != "/tmp/creds.json" {
		t.Errorf("expected /tmp/creds.json, got %s", cfg.FirebaseCredentialsFile)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://user:pass@host:5432/mydb")
	t.Setenv("FIREBASE_PROJECT_ID", "my-project")

	cfg := config.Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://user:pass@host:5432/mydb" {
		t.Errorf("expected custom database url, got %s", cfg.DatabaseURL)
	}
	if cfg.FirebaseProjectID != "my-project" {
		t.Errorf("expected firebase project id my-project, got %s", cfg.FirebaseProjectID)
	}
}
