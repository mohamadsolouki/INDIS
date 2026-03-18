// Package migrate applies numbered SQL migration files to a PostgreSQL database.
// It tracks applied migrations in a schema_migrations table so that each file
// is applied exactly once, in ascending numerical order.
package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationFilePattern matches files named like 001_create_users.sql.
var migrationFilePattern = regexp.MustCompile(`^\d+_.*\.sql$`)

// ensureSchemaTable creates the schema_migrations table if it does not exist.
func ensureSchemaTable(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	return err
}

// appliedMigrations returns the set of filenames already recorded in schema_migrations.
func appliedMigrations(ctx context.Context, db *pgxpool.Pool) (map[string]struct{}, error) {
	rows, err := db.Query(ctx, "SELECT filename FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate: query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("migrate: scan filename: %w", err)
		}
		applied[name] = struct{}{}
	}
	return applied, rows.Err()
}

// Migrate applies all pending SQL migrations from migrationsDir to the given database.
// It creates a schema_migrations table if it doesn't exist.
// Files must be named: 001_*.sql, 002_*.sql, etc.
// Already-applied migrations are skipped (tracked by filename in schema_migrations).
func Migrate(ctx context.Context, db *pgxpool.Pool, migrationsDir string) error {
	if err := ensureSchemaTable(ctx, db); err != nil {
		return fmt.Errorf("migrate: ensure schema table: %w", err)
	}

	applied, err := appliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("migrate: read directory %q: %w", migrationsDir, err)
	}

	// Collect and sort candidate files.
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if migrationFilePattern.MatchString(e.Name()) {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		if _, ok := applied[name]; ok {
			continue // already applied
		}

		path := filepath.Join(migrationsDir, name)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("migrate: read file %q: %w", name, err)
		}

		// Execute the migration and record it within a single transaction.
		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrate: begin transaction for %q: %w", name, err)
		}

		if _, err := tx.Exec(ctx, strings.TrimSpace(string(sql))); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate: apply %q: %w", name, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (filename) VALUES ($1)", name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate: record %q: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrate: commit %q: %w", name, err)
		}
	}

	return nil
}
