// Package config provides configuration loading for the electoral service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the configuration for the electoral service.
type Config struct {
	GRPCPort    int
	DatabaseURL string
}

// Load reads configuration from environment variables with hardcoded defaults.
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:    50057,
		DatabaseURL: "postgres://indis:indis_dev_password@localhost:5432/indis_electoral?sslmode=disable",
	}
	if v := os.Getenv("GRPC_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("config: GRPC_PORT: %w", err)
		}
		cfg.GRPCPort = p
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	return cfg, nil
}
