// Package config provides configuration loading for the gateway service.
package config

// Config holds the configuration for the gateway service.
type Config struct {
	// GRPCPort is the port for the gRPC server.
	GRPCPort int `env:"GRPC_PORT" envDefault:"50051"`

	// HTTPPort is the port for the HTTP/REST server.
	HTTPPort int `env:"HTTP_PORT" envDefault:"8080"`

	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string `env:"DATABASE_URL" envDefault:"postgres://indis:indis_dev_password@localhost:5432/indis_identity?sslmode=disable"`

	// RedisURL is the Redis connection string.
	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
}
