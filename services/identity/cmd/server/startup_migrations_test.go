package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	indismigrate "github.com/mohamadsolouki/INDIS/pkg/migrate"
	"github.com/mohamadsolouki/INDIS/services/identity/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdentityStartupAppliesMigrationsOnCleanSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn := os.Getenv("MIGRATE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set MIGRATE_TEST_DATABASE_URL to run startup migration integration test")
	}

	adminPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("new admin pool: %v", err)
	}
	defer adminPool.Close()

	schema := fmt.Sprintf("identity_startup_it_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA %s CASCADE", schema))
	}()

	testDSN, err := dsnWithSearchPath(dsn, schema)
	if err != nil {
		t.Fatalf("build schema-scoped dsn: %v", err)
	}

	pool, err := repository.NewPool(ctx, testDSN)
	if err != nil {
		t.Fatalf("repository.NewPool: %v", err)
	}
	defer pool.Close()

	if err := indismigrate.ApplyStartupMigrations(ctx, pool, repoMigrationsDir(t)); err != nil {
		t.Fatalf("apply startup migrations: %v", err)
	}

	var identitiesTable string
	if err := pool.QueryRow(ctx, "SELECT COALESCE(to_regclass('identities')::text, '')").Scan(&identitiesTable); err != nil {
		t.Fatalf("check identities table: %v", err)
	}
	if identitiesTable == "" {
		t.Fatal("expected identities table to exist after startup migrations")
	}

	var appliedCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&appliedCount); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if appliedCount == 0 {
		t.Fatal("expected at least one applied migration")
	}
}

func repoMigrationsDir(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../../"))
	return filepath.Join(repoRoot, "db", "migrations")
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
