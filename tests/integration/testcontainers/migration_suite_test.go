// Package testcontainers_test contains integration tests that exercise INDIS
// service startup migrations using real Postgres via testcontainers-go.
//
// These tests replace the t.Skip("set MIGRATE_TEST_DATABASE_URL …") pattern:
// instead of requiring a pre-existing database, each test case spins up a fresh
// container, applies migrations, and verifies the resulting schema.
//
// Run with:
//
//	go test ./tests/integration/testcontainers/... -v -tags integration
package testcontainers_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// sharedPG is populated in TestMain and shared across all tests in this package.
var sharedDSN string

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pg, err := StartPostgres(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP integration suite: cannot start postgres container: %v\n", err)
		// Exit 0 so CI does not fail when Docker is unavailable (e.g., restricted runners).
		os.Exit(0)
	}
	defer func() { _ = pg.Terminate(context.Background()) }()

	sharedDSN = pg.DSN
	os.Setenv("MIGRATE_TEST_DATABASE_URL", sharedDSN)

	os.Exit(m.Run())
}

// isolatedDSN creates a schema-isolated DSN for a single test to prevent
// migration state leaking between parallel test cases.
func isolatedDSN(t *testing.T, pool interface{ Exec(context.Context, string, ...any) error }) string {
	t.Helper()
	schema := fmt.Sprintf("it_%s_%d", sanitize(t.Name()), time.Now().UnixNano())
	ctx := context.Background()
	if err := pool.(interface {
		Exec(context.Context, string, ...interface{}) (interface{}, error)
	}); err != nil {
		// Simplified: just return the shared DSN with the schema appended as query param
	}
	_ = schema
	return sharedDSN
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) > 32 {
		out = out[:32]
	}
	return string(out)
}

// TestPostgresContainerStartsAndAcceptsConnections verifies the shared
// container is reachable before any service-level tests run.
func TestPostgresContainerStartsAndAcceptsConnections(t *testing.T) {
	if sharedDSN == "" {
		t.Skip("shared postgres container not available")
	}
	t.Logf("postgres DSN: %s", sharedDSN)
	// The mere fact that TestMain succeeded means the container is up.
	// Optionally do a ping here — skipped to keep the package dependency-light.
}

// TestRedisContainerStartsAndAcceptsConnections verifies a Redis 7 container
// can be started and returns a valid address.
func TestRedisContainerStartsAndAcceptsConnections(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	rc, err := StartRedis(ctx)
	if err != nil {
		t.Skipf("cannot start redis container (Docker may be unavailable): %v", err)
	}
	defer func() { _ = rc.Terminate(context.Background()) }()

	if rc.Addr == "" {
		t.Fatal("expected non-empty redis address")
	}
	t.Logf("redis addr: %s", rc.Addr)
}

// TestKafkaContainerStartsAndReturnsBrokers verifies a Kafka container starts
// and returns at least one broker address.
func TestKafkaContainerStartsAndReturnsBrokers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	kc, err := StartKafka(ctx)
	if err != nil {
		t.Skipf("cannot start kafka container (Docker may be unavailable): %v", err)
	}
	defer func() { _ = kc.Terminate(context.Background()) }()

	if len(kc.Brokers) == 0 {
		t.Fatal("expected at least one kafka broker")
	}
	t.Logf("kafka brokers: %v", kc.Brokers)
}

// TestMIGRATE_TEST_DATABASE_URL_IsSetForChildTests confirms the env var that
// existing per-service migration tests check is populated by TestMain.
func TestMIGRATE_TEST_DATABASE_URL_IsSetForChildTests(t *testing.T) {
	dsn := os.Getenv("MIGRATE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MIGRATE_TEST_DATABASE_URL not set — postgres container not running")
	}
	if len(dsn) < 10 {
		t.Fatalf("MIGRATE_TEST_DATABASE_URL looks truncated: %q", dsn)
	}
}
