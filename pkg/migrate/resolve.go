package migrate

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveMigrationsDir returns an absolute path to the SQL migrations directory.
//
// Resolution order:
//  1. explicitDir if provided.
//  2. MIGRATIONS_DIR environment variable.
//  3. Auto-discovery by walking upward from the current working directory
//     looking for db/migrations.
func ResolveMigrationsDir(explicitDir string) (string, error) {
	if explicitDir != "" {
		return validateMigrationsDir(explicitDir)
	}

	if envDir := os.Getenv("MIGRATIONS_DIR"); envDir != "" {
		return validateMigrationsDir(envDir)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("migrate: getwd: %w", err)
	}

	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "db", "migrations")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return "", fmt.Errorf("migrate: could not find db/migrations from %q; set MIGRATIONS_DIR", cwd)
}

func validateMigrationsDir(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("migrate: resolve absolute path %q: %w", dir, err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("migrate: stat migrations directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("migrate: migrations path %q is not a directory", absDir)
	}

	return absDir, nil
}
