package api

import (
	"database/sql"
	"testing"
)

// resetMigrationState keeps API tests isolated when NewServer auto-runs migrations.
func resetMigrationState(t *testing.T, db *sql.DB) {
	t.Helper()
	if db == nil {
		return
	}

	if _, err := db.Exec(`DROP TABLE IF EXISTS schema_migrations`); err != nil {
		t.Fatalf("failed to reset schema_migrations table: %v", err)
	}
}
