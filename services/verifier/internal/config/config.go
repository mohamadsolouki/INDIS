// Package config provides configuration loading for the verifier service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the runtime configuration for the verifier service.
type Config struct {
	// GRPCPort is the TCP port on which the gRPC server listens.
	GRPCPort int
	// MetricsPort is the port for the Prometheus metrics endpoint.
	MetricsPort int
	// DatabaseURL is the PostgreSQL connection string (libpq format).
	DatabaseURL string
	// ZKProofURL is the base URL of the zkproof HTTP verification endpoint.
	// The service will POST to ZKProofURL + "/verify".
	ZKProofURL string
}

// Load reads configuration from environment variables and applies documented defaults.
// This follows 12-factor app methodology (https://12factor.net/config).
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:    9110,
		MetricsPort: 9111,
		DatabaseURL: "postgres://indis:indis_dev_password@localhost:5432/indis_verifier?sslmode=disable",
		ZKProofURL:  "http://localhost:8080",
	}

	if v := os.Getenv("GRPC_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("config: GRPC_PORT must be an integer: %w", err)
		}
		cfg.GRPCPort = p
	}

	if v := os.Getenv("METRICS_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("config: METRICS_PORT must be an integer: %w", err)
		}
		cfg.MetricsPort = p
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}

	if v := os.Getenv("ZK_PROOF_URL"); v != "" {
		cfg.ZKProofURL = v
	}

	return cfg, nil
}
