package migrate

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrate_AppliesAllMigrationsOnCleanSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn := os.Getenv("MIGRATE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set MIGRATE_TEST_DATABASE_URL to run migration integration tests")
	}

	adminPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("new admin pool: %v", err)
	}
	defer adminPool.Close()

	schema := fmt.Sprintf("migrate_it_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA %s CASCADE", schema))
	}()

	testDSN, err := dsnWithSearchPath(dsn, schema)
	if err != nil {
		t.Fatalf("build schema-scoped DSN: %v", err)
	}

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("new test pool: %v", err)
	}
	defer pool.Close()

	migrationsDir := repoMigrationsDir(t)
	if err := Migrate(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	expectedFiles, err := migrationFiles(migrationsDir)
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}

	var appliedCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&appliedCount); err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}

	if appliedCount != len(expectedFiles) {
		t.Fatalf("expected %d applied migrations, got %d", len(expectedFiles), appliedCount)
	}

	var identitiesTable string
	if err := pool.QueryRow(ctx, "SELECT COALESCE(to_regclass('identities')::text, '')").Scan(&identitiesTable); err != nil {
		t.Fatalf("check identities table: %v", err)
	}
	if identitiesTable == "" {
		t.Fatal("expected identities table to exist after migrations")
	}
}

func TestMigrate_IsIdempotentOnSecondRun(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn := os.Getenv("MIGRATE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set MIGRATE_TEST_DATABASE_URL to run migration integration tests")
	}

	adminPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("new admin pool: %v", err)
	}
	defer adminPool.Close()

	schema := fmt.Sprintf("migrate_it_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA %s CASCADE", schema))
	}()

	testDSN, err := dsnWithSearchPath(dsn, schema)
	if err != nil {
		t.Fatalf("build schema-scoped DSN: %v", err)
	}

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("new test pool: %v", err)
	}
	defer pool.Close()

	migrationsDir := repoMigrationsDir(t)
	if err := Migrate(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("first migrate: %v", err)
	}

	var firstCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&firstCount); err != nil {
		t.Fatalf("first count: %v", err)
	}

	if err := Migrate(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	var secondCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&secondCount); err != nil {
		t.Fatalf("second count: %v", err)
	}

	if secondCount != firstCount {
		t.Fatalf("expected idempotent migration count %d, got %d", firstCount, secondCount)
	}
}

func repoMigrationsDir(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../.."))
	return filepath.Join(repoRoot, "db", "migrations")
}

func migrationFiles(migrationsDir string) ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, err
	}

	pattern := regexp.MustCompile(`^\d+_.*\.sql$`)
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if pattern.MatchString(entry.Name()) {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)
	return files, nil
}

func dsnWithSearchPath(dsn, schema string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	return u.String(), nil
}
