// Package config provides configuration loading for the biometric service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the configuration for the biometric service.
type Config struct {
	GRPCPort     int
	MetricsPort  int
	DatabaseURL  string
	AIServiceURL string
}

// Load reads configuration from environment variables with hardcoded defaults.
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:     50054,
		MetricsPort:  9104,
		DatabaseURL:  "postgres://indis:indis_dev_password@localhost:5432/indis_biometric?sslmode=disable",
		AIServiceURL: "http://localhost:8000",
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
	if v := os.Getenv("AI_SERVICE_URL"); v != "" {
		cfg.AIServiceURL = v
	}
	return cfg, nil
}
