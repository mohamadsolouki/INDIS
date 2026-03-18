// Package config provides configuration loading for the justice service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the configuration for the justice service.
type Config struct {
	GRPCPort    int
	MetricsPort int
	DatabaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:    50058,
		MetricsPort: 9108,
		DatabaseURL: "postgres://indis:indis_dev_password@localhost:5432/indis_justice?sslmode=disable",
	}
	if v := os.Getenv("GRPC_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("config: GRPC_PORT: %w", err)
		}
		cfg.GRPCPort = p
	}
	if v := os.Getenv("METRICS_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("config: METRICS_PORT: %w", err)
		}
		cfg.MetricsPort = p
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	return cfg, nil
}
