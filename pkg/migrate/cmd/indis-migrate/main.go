package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	indismigrate "github.com/IranProsperityProject/INDIS/pkg/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var (
		databaseURL   = flag.String("database-url", envOr("DATABASE_URL", ""), "PostgreSQL connection string")
		migrationsDir = flag.String("migrations-dir", "", "Path to SQL migrations directory (optional; defaults to MIGRATIONS_DIR or auto-discovery)")
		timeout       = flag.Duration("timeout", 2*time.Minute, "Migration execution timeout")
	)
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("missing database URL: set --database-url or DATABASE_URL")
	}

	resolvedDir, err := indismigrate.ResolveMigrationsDir(*migrationsDir)
	if err != nil {
		log.Fatalf("failed to resolve migrations directory: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	db, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	if err := indismigrate.Migrate(ctx, db, resolvedDir); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	fmt.Fprintf(os.Stdout, "migrations applied successfully from %s\n", resolvedDir)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
