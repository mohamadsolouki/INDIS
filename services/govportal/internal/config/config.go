// Package config provides configuration loading for the govportal service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the runtime configuration for the govportal service.
type Config struct {
	// HTTPPort is the TCP port on which the HTTP server listens.
	HTTPPort int
	// MetricsPort is the port for the Prometheus metrics endpoint.
	MetricsPort int
	// DatabaseURL is the PostgreSQL connection string (libpq format).
	DatabaseURL string
	// IdentityGRPCAddr is the host:port of the identity gRPC service.
	IdentityGRPCAddr string
	// CredentialGRPCAddr is the host:port of the credential gRPC service.
	CredentialGRPCAddr string
	// AuditGRPCAddr is the host:port of the audit gRPC service.
	AuditGRPCAddr string
	// JWTSecret is the HMAC-SHA256 key used to validate ministry user tokens.
	JWTSecret string
}

// Load reads configuration from environment variables and applies documented defaults.
// This follows 12-factor app methodology (https://12factor.net/config).
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:           8200,
		MetricsPort:        9112,
		DatabaseURL:        "postgres://indis:indis_dev_password@localhost:5432/indis_govportal?sslmode=disable",
		IdentityGRPCAddr:   "localhost:50051",
		CredentialGRPCAddr: "localhost:50052",
		AuditGRPCAddr:      "localhost:50056",
		JWTSecret:          "indis-govportal-dev-secret",
	}

	if v := os.Getenv("HTTP_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("config: HTTP_PORT must be an integer: %w", err)
		}
		cfg.HTTPPort = p
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
	if v := os.Getenv("IDENTITY_GRPC_ADDR"); v != "" {
		cfg.IdentityGRPCAddr = v
	}
	if v := os.Getenv("CREDENTIAL_GRPC_ADDR"); v != "" {
		cfg.CredentialGRPCAddr = v
	}
	if v := os.Getenv("AUDIT_GRPC_ADDR"); v != "" {
		cfg.AuditGRPCAddr = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}

	return cfg, nil
}
