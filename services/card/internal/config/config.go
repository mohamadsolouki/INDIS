// Package config provides configuration loading for the card service.
package config

import (
	"fmt"
	"os"
)

// Config holds the configuration for the card service.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string
	// HTTPPort is the TCP address to listen on (e.g. ":8400").
	HTTPPort string
	// CardIssuerSeed is the 32-byte hex-encoded Ed25519 seed for the issuer signing key.
	// A random seed is generated at startup when this is empty.
	CardIssuerSeed string
}

// Load reads configuration from environment variables with hardcoded defaults.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:    "postgres://indis:indis_dev_password@localhost:5432/indis_card?sslmode=disable",
		HTTPPort:       ":8400",
		CardIssuerSeed: "",
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("HTTP_PORT"); v != "" {
		if len(v) > 0 && v[0] != ':' {
			v = ":" + v
		}
		cfg.HTTPPort = v
	}
	if v := os.Getenv("CARD_ISSUER_SEED"); v != "" {
		cfg.CardIssuerSeed = v
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL must not be empty")
	}
	return cfg, nil
}
