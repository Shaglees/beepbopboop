package database_test

import (
	"testing"

	"github.com/shanegleeson/beepbopboop/backend/internal/database"
)

func TestOpenAndMigrate(t *testing.T) {
	db := database.OpenTestDB(t)

	tables := []string{"users", "agents", "agent_tokens", "posts"}
	for _, table := range tables {
		_, err := db.Exec("SELECT count(*) FROM " + table)
		if err != nil {
			t.Errorf("table %s does not exist: %v", table, err)
		}
	}
}

func TestLocalCreatorsTableExists(t *testing.T) {
	db := database.OpenTestDB(t)
	_, err := db.Exec(`INSERT INTO local_creators (name, designation, source) VALUES ('Test Creator', 'Painter', 'test')`)
	if err != nil {
		t.Fatalf("local_creators table missing or wrong schema: %v", err)
	}
}
