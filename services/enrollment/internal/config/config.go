// Package config provides configuration loading for the enrollment service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the configuration for the enrollment service.
type Config struct {
	GRPCPort    int
	HTTPPort    int
	DatabaseURL string
	RedisURL    string
}

// Load reads configuration from environment variables with hardcoded defaults.
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:    50053,
		HTTPPort:    8082,
		DatabaseURL: "postgres://indis:indis_dev_password@localhost:5432/indis_enrollment?sslmode=disable",
		RedisURL:    "redis://localhost:6379/2",
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
