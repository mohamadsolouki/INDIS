// Package config provides configuration loading for the audit service.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the configuration for the audit service.
type Config struct {
	GRPCPort    int
	DatabaseURL string
	KafkaBrokers []string
	KafkaGroupID string
}

// Load reads configuration from environment variables with hardcoded defaults.
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:    50055,
		DatabaseURL: "postgres://indis:indis_dev_password@localhost:5432/indis_audit?sslmode=disable",
		KafkaBrokers: []string{"localhost:9092"},
		KafkaGroupID: "audit-service",
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
