// Package config provides configuration loading for the credential service.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the configuration for the credential service.
type Config struct {
	GRPCPort    int
	HTTPPort    int
	DatabaseURL string
	RedisURL    string
	// IssuerDID is the DID used when issuing credentials as the system authority.
	IssuerDID string
	KafkaBrokers []string
	KafkaGroupID string
}

// Load reads configuration from environment variables with hardcoded defaults.
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:    50052,
		HTTPPort:    8081,
		DatabaseURL: "postgres://indis:indis_dev_password@localhost:5432/indis_credential?sslmode=disable",
		RedisURL:    "redis://localhost:6379/1",
		IssuerDID:   "did:indis:system",
		KafkaBrokers: []string{"localhost:9092"},
		KafkaGroupID: "credential-service",
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
	if v := os.Getenv("ISSUER_DID"); v != "" {
		cfg.IssuerDID = v
	}
	if v := os.Getenv("KAFKA_BROKERS"); v != "" {
		cfg.KafkaBrokers = splitAndTrim(v)
	}
	if v := os.Getenv("KAFKA_GROUP_ID"); v != "" {
		cfg.KafkaGroupID = v
	}
	return cfg, nil
}

func splitAndTrim(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
