package migrate

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplyStartupMigrations resolves the migrations directory and applies all
// pending SQL migrations for a service startup path.
func ApplyStartupMigrations(ctx context.Context, db *pgxpool.Pool, explicitDir string) error {
	migrationsDir, err := ResolveMigrationsDir(explicitDir)
	if err != nil {
		return fmt.Errorf("resolve migrations dir: %w", err)
	}

	if err := Migrate(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
