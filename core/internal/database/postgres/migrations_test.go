package postgres

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/lookitval/nabu/core/internal/testutils"
)

func postgresPackageDir(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err == nil {
		if _, statErr := os.Stat(filepath.Join(wd, "migrations.go")); statErr == nil {
			return wd
		}
	}

	_, fileName, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve package directory")
	}

	return filepath.Dir(fileName)
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change directory to %q: %v", dir, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("failed to restore working directory to %q: %v", oldWD, err)
		}
	})
}

func mustConnectPostgres(t *testing.T) *sql.DB {
	t.Helper()

	db, err := Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db
}

func resetSchemaMigrationsTable(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`DROP TABLE IF EXISTS schema_migrations`); err != nil {
		t.Fatalf("failed to drop schema_migrations: %v", err)
	}
}

func createExpectedSchemaMigrationsTable(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		CREATE TABLE schema_migrations (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("failed to create schema_migrations: %v", err)
	}
}

func createTempMigrationsDir(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	dir := filepath.Join(root, "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create migrations directory: %v", err)
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write migration file %q: %v", name, err)
		}
	}

	return root
}

func TestGetMigrations_ReturnsSortedSQLFiles(t *testing.T) {
	migrations, err := getMigrations()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(migrations) == 0 {
		t.Fatal("expected embedded migrations to be discovered, got none")
	}

	wantFirst := "000001_create_schema_migrations.sql"
	if migrations[0] != wantFirst {
		t.Fatalf("expected first migration to be %q, got %q", wantFirst, migrations[0])
	}

	sorted := append([]string(nil), migrations...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(migrations, sorted) {
		t.Fatalf("expected migrations list to be sorted, got %#v", migrations)
	}
}

func TestGetMigrations_UsesFallbackDirectoryWhenLocalMissing(t *testing.T) {
	withWorkingDir(t, t.TempDir())

	migrations, err := getMigrations()
	if err != nil {
		t.Fatalf("expected nil error using fallback directory, got %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected fallback migrations to be discovered, got none")
	}
}

func TestMustReadSQLQuery_PanicsForUnknownPath(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unknown embedded SQL path, got none")
		}
	}()

	_ = mustReadSQLQuery("sqlutils/does_not_exist.sql")
}

func TestMigrationTableExists_ReturnsFalseWhenMissing(t *testing.T) {
	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)

	exists, err := migrationTableExists(context.Background(), db)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if exists {
		t.Fatal("expected table to not exist")
	}
}

func TestMigrationTableExists_ReturnsTrueWhenPresent(t *testing.T) {
	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)
	createExpectedSchemaMigrationsTable(t, db)
	t.Cleanup(func() { resetSchemaMigrationsTable(t, db) })

	exists, err := migrationTableExists(context.Background(), db)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !exists {
		t.Fatal("expected table to exist")
	}
}

func TestMigrationTableExists_ReturnsErrorOnClosedDB(t *testing.T) {
	db := mustConnectPostgres(t)
	_ = db.Close()

	_, err := migrationTableExists(context.Background(), db)
	if err == nil {
		t.Fatal("expected error on closed database, got nil")
	}
}

func TestMigrationTableHasExpectedSchema_ReturnsFalseWhenInvalid(t *testing.T) {
	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)

	if _, err := db.Exec(`CREATE TABLE schema_migrations (name TEXT)`); err != nil {
		t.Fatalf("failed to create invalid schema_migrations table: %v", err)
	}
	t.Cleanup(func() { resetSchemaMigrationsTable(t, db) })

	valid, err := migrationTableHasExpectedSchema(context.Background(), db)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if valid {
		t.Fatal("expected invalid schema result")
	}
}

func TestMigrationTableHasExpectedSchema_ReturnsTrueWhenValid(t *testing.T) {
	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)
	createExpectedSchemaMigrationsTable(t, db)
	t.Cleanup(func() { resetSchemaMigrationsTable(t, db) })

	valid, err := migrationTableHasExpectedSchema(context.Background(), db)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !valid {
		t.Fatal("expected valid schema result")
	}
}

func TestMigrationTableHasExpectedSchema_ReturnsErrorOnClosedDB(t *testing.T) {
	db := mustConnectPostgres(t)
	_ = db.Close()

	_, err := migrationTableHasExpectedSchema(context.Background(), db)
	if err == nil {
		t.Fatal("expected error on closed database, got nil")
	}
}

func TestGetAppliedMigrations_ReturnsEmptyAndWarnsWhenTableMissing(t *testing.T) {
	packageDir := postgresPackageDir(t)
	withWorkingDir(t, packageDir)

	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)

	testutils.CaptureAndWaitForOutput(t, "does not exist", 500*time.Millisecond, func() {
		migrations, err := getAppliedMigrations(context.Background(), db)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(migrations) != 0 {
			t.Fatalf("expected empty migration list, got %#v", migrations)
		}
	})
}

func TestGetAppliedMigrations_ReturnsEmptyAndWarnsWhenSchemaInvalid(t *testing.T) {
	packageDir := postgresPackageDir(t)
	withWorkingDir(t, packageDir)

	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)
	if _, err := db.Exec(`CREATE TABLE schema_migrations (name TEXT)`); err != nil {
		t.Fatalf("failed to create invalid schema table: %v", err)
	}
	t.Cleanup(func() { resetSchemaMigrationsTable(t, db) })

	testutils.CaptureAndWaitForOutput(t, "does not match expected shape", 500*time.Millisecond, func() {
		migrations, err := getAppliedMigrations(context.Background(), db)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(migrations) != 0 {
			t.Fatalf("expected empty migration list, got %#v", migrations)
		}
	})
}

func TestGetAppliedMigrations_ReturnsNamesWhenValidTableExists(t *testing.T) {
	packageDir := postgresPackageDir(t)
	withWorkingDir(t, packageDir)

	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)
	createExpectedSchemaMigrationsTable(t, db)
	t.Cleanup(func() { resetSchemaMigrationsTable(t, db) })

	if _, err := db.Exec(`INSERT INTO schema_migrations (name) VALUES ('000002_b.sql'), ('000001_a.sql')`); err != nil {
		t.Fatalf("failed to seed schema_migrations: %v", err)
	}

	migrations, err := getAppliedMigrations(context.Background(), db)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := []string{"000001_a.sql", "000002_b.sql"}
	if !reflect.DeepEqual(migrations, want) {
		t.Fatalf("unexpected applied migrations: got %#v want %#v", migrations, want)
	}
}

func TestGetAppliedMigrations_ReturnsErrorWhenDBQueryFails(t *testing.T) {
	db := mustConnectPostgres(t)
	_ = db.Close()

	_, err := getAppliedMigrations(context.Background(), db)
	if err == nil {
		t.Fatal("expected error on closed database, got nil")
	}
}

func TestApplyMigration_ReturnsErrorWhenFileMissing(t *testing.T) {
	db := mustConnectPostgres(t)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when embedded migration file is missing, got none")
		}
	}()

	_ = applyMigration(context.Background(), db, "000001_missing.sql")
}

func TestApplyMigration_PanicsWhenMigrationIsNotEmbedded(t *testing.T) {
	db := mustConnectPostgres(t)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when migration is not embedded, got none")
		}
	}()

	_ = applyMigration(context.Background(), db, "000001_invalid.sql")
}

func TestApplyMigration_ReturnsErrorWhenTrackingInsertFails(t *testing.T) {
	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)
	if _, err := db.Exec(`CREATE TABLE schema_migrations (id SERIAL PRIMARY KEY, version VARCHAR(255) NOT NULL)`); err != nil {
		t.Fatalf("failed to create malformed schema_migrations table: %v", err)
	}
	t.Cleanup(func() { resetSchemaMigrationsTable(t, db) })

	err := applyMigration(context.Background(), db, "000001_create_schema_migrations.sql")
	if err == nil {
		t.Fatal("expected tracking insert error, got nil")
	}
}

func TestApplyMigration_AppliesSQLAndTracksMigration(t *testing.T) {
	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)
	t.Cleanup(func() { resetSchemaMigrationsTable(t, db) })

	err := applyMigration(context.Background(), db, "000001_create_schema_migrations.sql")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = $1`, "000001_create_schema_migrations.sql").Scan(&count); err != nil {
		t.Fatalf("failed to verify tracked migration row: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one tracked migration row, got %d", count)
	}
}

func TestApplyMigration_ReturnsErrorWhenBeginTxFails(t *testing.T) {
	db := mustConnectPostgres(t)
	_ = db.Close()

	err := applyMigration(context.Background(), db, "000001_create_schema_migrations.sql")
	if err == nil {
		t.Fatal("expected begin transaction error on closed db, got nil")
	}
}

func TestApplyMigrations_UsesFallbackMigrationsDirectory(t *testing.T) {
	withWorkingDir(t, t.TempDir())

	db := mustConnectPostgres(t)
	err := ApplyMigrations(context.Background(), db)
	if err != nil {
		t.Fatalf("expected fallback migrations to apply successfully, got %v", err)
	}
}

func TestApplyMigrations_ReturnsErrorWhenGetAppliedFails(t *testing.T) {
	packageDir := postgresPackageDir(t)
	withWorkingDir(t, packageDir)

	db := mustConnectPostgres(t)
	_ = db.Close()

	err := ApplyMigrations(context.Background(), db)
	if err == nil {
		t.Fatal("expected error with closed database, got nil")
	}
}

func TestApplyMigrations_AppliesPendingAndSkipsApplied(t *testing.T) {
	packageDir := postgresPackageDir(t)
	withWorkingDir(t, packageDir)

	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)
	t.Cleanup(func() { resetSchemaMigrationsTable(t, db) })

	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("expected first ApplyMigrations to succeed, got %v", err)
	}

	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("expected second ApplyMigrations to succeed, got %v", err)
	}
}

func TestMustReadSQLQuery_ReturnsEmbeddedSQLContent(t *testing.T) {
	query := mustReadSQLQuery(getAppliedMigrationsQueryFile)
	if !strings.Contains(query, "FROM schema_migrations") {
		t.Fatalf("unexpected query contents: %q", query)
	}
}

func TestApplyMigrations_ReturnsErrorWhenApplyMigrationFails(t *testing.T) {
	db := mustConnectPostgres(t)
	resetSchemaMigrationsTable(t, db)
	if _, err := db.Exec(`CREATE TABLE schema_migrations (id SERIAL PRIMARY KEY, version VARCHAR(255) NOT NULL)`); err != nil {
		t.Fatalf("failed to create malformed schema_migrations table: %v", err)
	}
	t.Cleanup(func() { resetSchemaMigrationsTable(t, db) })

	err := ApplyMigrations(context.Background(), db)
	if err == nil {
		t.Fatal("expected applyMigration error, got nil")
	}
}
