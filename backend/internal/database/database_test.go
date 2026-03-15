package database_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
)

func TestOpenAndMigrate(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tables := []string{"users", "agents", "agent_tokens", "posts"}
	for _, table := range tables {
		_, err := db.Exec("SELECT count(*) FROM " + table)
		if err != nil {
			t.Errorf("table %s does not exist: %v", table, err)
		}
	}
}
