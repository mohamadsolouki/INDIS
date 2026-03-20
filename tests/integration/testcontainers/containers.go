// Package testcontainers provides shared test infrastructure helpers that spin
// up real Postgres, Redis, and Kafka containers for INDIS integration tests.
//
// Usage:
//
//	func TestMain(m *testing.M) {
//	    ctx := context.Background()
//	    pg, dsn, err := testcontainers.StartPostgres(ctx)
//	    // ...
//	    os.Setenv("MIGRATE_TEST_DATABASE_URL", dsn)
//	    code := m.Run()
//	    pg.Terminate(ctx)
//	    os.Exit(code)
//	}
package testcontainers

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	tcKafka "github.com/testcontainers/testcontainers-go/modules/kafka"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
)

const (
	postgresImage = "postgres:16-alpine"
	redisImage    = "redis:7-alpine"
	kafkaImage    = "confluentinc/cp-kafka:7.6.1"

	defaultDBName   = "indis_test"
	defaultDBUser   = "indis"
	defaultDBPass   = "indis_test_secret"
)

// PostgresContainer wraps a testcontainers Postgres instance.
type PostgresContainer struct {
	Container testcontainers.Container
	DSN       string
}

// StartPostgres starts a Postgres 16 container and returns it with a ready DSN.
// The caller is responsible for calling Terminate when done.
func StartPostgres(ctx context.Context) (*PostgresContainer, error) {
	container, err := tcPostgres.RunContainer(ctx,
		testcontainers.WithImage(postgresImage),
		tcPostgres.WithDatabase(defaultDBName),
		tcPostgres.WithUsername(defaultDBUser),
		tcPostgres.WithPassword(defaultDBPass),
		testcontainers.WithWaitStrategy(
			tcPostgres.BasicWaitStrategies()...,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("postgres connection string: %w", err)
	}

	return &PostgresContainer{Container: container, DSN: dsn}, nil
}

// Terminate stops and removes the Postgres container.
func (p *PostgresContainer) Terminate(ctx context.Context) error {
	return p.Container.Terminate(ctx)
}

// RedisContainer wraps a testcontainers Redis instance.
type RedisContainer struct {
	Container testcontainers.Container
	Addr      string // host:port
}

// StartRedis starts a Redis 7 container and returns it with a ready address.
func StartRedis(ctx context.Context) (*RedisContainer, error) {
	container, err := tcRedis.RunContainer(ctx,
		testcontainers.WithImage(redisImage),
	)
	if err != nil {
		return nil, fmt.Errorf("start redis container: %w", err)
	}

	addr, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("redis connection string: %w", err)
	}

	return &RedisContainer{Container: container, Addr: addr}, nil
}

// Terminate stops and removes the Redis container.
func (r *RedisContainer) Terminate(ctx context.Context) error {
	return r.Container.Terminate(ctx)
}

// KafkaContainer wraps a testcontainers Kafka instance.
type KafkaContainer struct {
	Container *tcKafka.KafkaContainer
	Brokers   []string
}

// StartKafka starts a Kafka container and returns it with a bootstrap broker list.
func StartKafka(ctx context.Context) (*KafkaContainer, error) {
	container, err := tcKafka.RunContainer(ctx,
		testcontainers.WithImage(kafkaImage),
		tcKafka.WithClusterID("indis-test-cluster"),
	)
	if err != nil {
		return nil, fmt.Errorf("start kafka container: %w", err)
	}

	brokers, err := container.Brokers(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("kafka brokers: %w", err)
	}

	return &KafkaContainer{Container: container, Brokers: brokers}, nil
}

// Terminate stops and removes the Kafka container.
func (k *KafkaContainer) Terminate(ctx context.Context) error {
	return k.Container.Terminate(ctx)
}
