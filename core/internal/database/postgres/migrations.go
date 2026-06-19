package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"slices"
	"sort"
	"strings"
)

const migrationTable = "schema_migrations"

const (
	getAppliedMigrationsQueryFile = "sqlutils/getAppliedMigrationsQuery.sql"
	migrationTableExistsQueryFile = "sqlutils/migrationTableExistsQuery.sql"
	schemaCheckQueryFile          = "sqlutils/schemaCheckQuery.sql"
	constraintsCheckQueryFile     = "sqlutils/constraintsCheckQuery.sql"
	insertAppliedMigrationQuery   = "sqlutils/insertAppliedMigrationQuery.sql"
)

//go:embed sqlutils/*.sql
var sqlutilsFS embed.FS

//go:embed migrations/*.sql
var migrationsFS embed.FS

var (
	getAppliedMigrationsQuery = mustReadSQLQuery(getAppliedMigrationsQueryFile)
	migrationTableExistsQuery = mustReadSQLQuery(migrationTableExistsQueryFile)
	schemaCheckQuery          = mustReadSQLQuery(schemaCheckQueryFile)
	constraintsCheckQuery     = mustReadSQLQuery(constraintsCheckQueryFile)
	insertAppliedMigrationSQL = mustReadSQLQuery(insertAppliedMigrationQuery)
)

// ApplyMigrations applies all pending database migrations to ensure the schema is up to date.
func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	migrations, err := getMigrations()
	if err != nil {
		return err
	}

	appliedMigrations, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if slices.Contains(appliedMigrations, migration) {
			continue
		}
		if err := applyMigration(ctx, db, migration); err != nil {
			return err
		}
	}

	return nil
}

// getMigrations retrieves the list migration files that exist in the migrations directory
func getMigrations() ([]string, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	var migrations []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		migrations = append(migrations, entry.Name())
	}

	sort.Strings(migrations)
	return migrations, nil
}

// getAppliedMigrations queries the database and retrieves the list migration files that have been applied
func getAppliedMigrations(ctx context.Context, db *sql.DB) ([]string, error) {
	exists, err := migrationTableExists(ctx, db)
	if err != nil {
		return nil, err
	}
	if !exists {
		fmt.Printf("WARNING: migrations table %q does not exist; returning empty applied migration list\n", migrationTable)
		return []string{}, nil
	}

	validSchema, err := migrationTableHasExpectedSchema(ctx, db)
	if err != nil {
		return nil, err
	}
	if !validSchema {
		fmt.Printf("WARNING: migrations table %q schema does not match expected shape; returning empty applied migration list\n", migrationTable)
		return []string{}, nil
	}

	rows, err := db.QueryContext(ctx, getAppliedMigrationsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appliedMigrations []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		appliedMigrations = append(appliedMigrations, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return appliedMigrations, nil
}

// migrationTableExists reports whether the schema_migrations table exists in current schema.
func migrationTableExists(ctx context.Context, db *sql.DB) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, migrationTableExistsQuery).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// migrationTableHasExpectedSchema validates the table shape used by migration tracking.
func migrationTableHasExpectedSchema(ctx context.Context, db *sql.DB) (bool, error) {
	var hasExpectedColumns bool
	var hasID bool
	var hasName bool
	var hasAppliedAt bool

	err := db.QueryRowContext(ctx, schemaCheckQuery).Scan(
		&hasExpectedColumns,
		&hasID,
		&hasName,
		&hasAppliedAt,
	)
	if err != nil {
		return false, err
	}

	if !(hasExpectedColumns && hasID && hasName && hasAppliedAt) {
		return false, nil
	}

	var hasPK bool
	var hasUniqueName bool

	err = db.QueryRowContext(ctx, constraintsCheckQuery).Scan(&hasPK, &hasUniqueName)
	if err != nil {
		return false, err
	}

	return hasPK && hasUniqueName, nil
}

// applyMigration reads the SQL from the specified migration file and executes it against the database.
func applyMigration(ctx context.Context, db *sql.DB, migration string) error {
	migrationSQL := getMigrationString(migration)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, string(migrationSQL)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, insertAppliedMigrationSQL, migration); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// mustReadSQLQuery loads an embedded SQL file and panics when it cannot be read.
// Embedded SQL files are part of the binary, so a missing file is a build-time contract failure.
func mustReadSQLQuery(filePath string) string {
	b, err := sqlutilsFS.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("failed to read embedded SQL %q: %v", filePath, err))
	}

	return string(b)
}

// getMigrationString returns the full embedded content for a given migration file path.
func getMigrationString(migration string) string {
	b, err := migrationsFS.ReadFile(fmt.Sprintf("migrations/%s", migration))
	if err != nil {
		panic(fmt.Sprintf("failed to read embedded migration SQL for %q: %v", migration, err))
	}

	return string(b)
}
