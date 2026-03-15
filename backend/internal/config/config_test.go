package config_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("FIREBASE_PROJECT_ID", "")
	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.DatabasePath != "beepbopboop.db" {
		t.Errorf("expected default db path beepbopboop.db, got %s", cfg.DatabasePath)
	}
	if cfg.FirebaseProjectID != "" {
		t.Errorf("expected empty firebase project id, got %s", cfg.FirebaseProjectID)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_PATH", "/tmp/test.db")
	t.Setenv("FIREBASE_PROJECT_ID", "my-project")

	cfg := config.Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("expected db path /tmp/test.db, got %s", cfg.DatabasePath)
	}
	if cfg.FirebaseProjectID != "my-project" {
		t.Errorf("expected firebase project id my-project, got %s", cfg.FirebaseProjectID)
	}
}
