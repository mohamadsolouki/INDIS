// Package config provides configuration loading for the electoral service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the configuration for the electoral service.
type Config struct {
	GRPCPort                 int
	MetricsPort              int
	DatabaseURL              string
	ZKProofURL               string
	RemoteNonceWindowMinutes int
}

// Load reads configuration from environment variables with hardcoded defaults.
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:                 50057,
		MetricsPort:              9107,
		DatabaseURL:              "postgres://indis:indis_dev_password@localhost:5432/indis_electoral?sslmode=disable",
		ZKProofURL:               "http://localhost:8088",
		RemoteNonceWindowMinutes: 60,
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
	if v := os.Getenv("ZKPROOF_URL"); v != "" {
		cfg.ZKProofURL = v
	}
	if v := os.Getenv("REMOTE_NONCE_WINDOW_MINUTES"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("config: REMOTE_NONCE_WINDOW_MINUTES: %w", err)
		}
		if p <= 0 {
			return nil, fmt.Errorf("config: REMOTE_NONCE_WINDOW_MINUTES must be > 0")
		}
		cfg.RemoteNonceWindowMinutes = p
	}
	return cfg, nil
}
