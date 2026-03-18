// Package config provides configuration loading for the identity service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the configuration for the identity service.
type Config struct {
	// GRPCPort is the port for the gRPC server.
	GRPCPort int
	// HTTPPort is the port for the HTTP/REST server.
	HTTPPort int
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string
	// RedisURL is the Redis connection string.
	RedisURL string
}

// Load reads configuration from environment variables with hardcoded defaults.
// This follows 12-factor app methodology (https://12factor.net/config).
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:    50051,
		HTTPPort:    8080,
		DatabaseURL: "postgres://indis:indis_dev_password@localhost:5432/indis_identity?sslmode=disable",
		RedisURL:    "redis://localhost:6379/0",
	}

	if v := os.Getenv("GRPC_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("config: GRPC_PORT must be an integer: %w", err)
		}
		cfg.GRPCPort = p
	}

	if v := os.Getenv("HTTP_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("config: HTTP_PORT must be an integer: %w", err)
		}
		cfg.HTTPPort = p
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("REDIS_URL"); v != "" {
		cfg.RedisURL = v
	}

	return cfg, nil
}
