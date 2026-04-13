package database

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// OpenTestDB starts a Postgres container and returns a connected *sql.DB.
// The container is automatically terminated when the test finishes.
// If TEST_DATABASE_URL is set, it connects to that instead of starting a container.
// Skips the test if Docker is unavailable.
func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Allow override for CI or local Postgres
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		db, err := Open(url)
		if err != nil {
			t.Fatalf("open test database: %v", err)
		}
		t.Cleanup(func() { db.Close() })
		return db
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Skipf("skipping: Docker unavailable (%v)", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("get connection string: %v", err)
	}

	db, err := Open(connStr)
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		pgContainer.Terminate(ctx)
	})

	return db
}
