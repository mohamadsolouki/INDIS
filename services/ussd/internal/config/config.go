// Package config provides configuration loading for the USSD/SMS gateway service.
package config

import (
	"fmt"
	"os"
)

// Config holds the configuration for the USSD service.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string
	// HTTPPort is the TCP address to listen on (e.g. ":8300").
	HTTPPort string
	// GatewayURL is the base URL of the INDIS API gateway for upstream verification calls.
	GatewayURL string
}

// Load reads configuration from environment variables with hardcoded defaults.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: "postgres://indis:indis_dev_password@localhost:5432/indis_ussd?sslmode=disable",
		HTTPPort:    ":8300",
		GatewayURL:  "http://localhost:8080",
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("HTTP_PORT"); v != "" {
		// Accept with or without leading colon.
		if len(v) > 0 && v[0] != ':' {
			v = ":" + v
		}
		cfg.HTTPPort = v
	}
	if v := os.Getenv("GATEWAY_URL"); v != "" {
		cfg.GatewayURL = v
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL must not be empty")
	}
	return cfg, nil
}
