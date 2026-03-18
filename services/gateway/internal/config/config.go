// Package config provides configuration loading for the gateway service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the configuration for the gateway service.
type Config struct {
	HTTPPort int

	// Backend service addresses (host:port)
	IdentityAddr     string
	CredentialAddr   string
	EnrollmentAddr   string
	BiometricAddr    string
	AuditAddr        string
	NotificationAddr string
	ElectoralAddr    string
	JusticeAddr      string

	// Rate limit: max requests per second per IP
	RateLimitRPS int
}

// Load reads configuration from environment variables with sane defaults.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:         envInt("HTTP_PORT", 8080),
		IdentityAddr:     envStr("IDENTITY_ADDR", "localhost:50051"),
		CredentialAddr:   envStr("CREDENTIAL_ADDR", "localhost:50052"),
		EnrollmentAddr:   envStr("ENROLLMENT_ADDR", "localhost:50053"),
		BiometricAddr:    envStr("BIOMETRIC_ADDR", "localhost:50054"),
		AuditAddr:        envStr("AUDIT_ADDR", "localhost:50055"),
		NotificationAddr: envStr("NOTIFICATION_ADDR", "localhost:50056"),
		ElectoralAddr:    envStr("ELECTORAL_ADDR", "localhost:50057"),
		JusticeAddr:      envStr("JUSTICE_ADDR", "localhost:50058"),
		RateLimitRPS:     envInt("RATE_LIMIT_RPS", 100),
	}

	if cfg.RateLimitRPS <= 0 {
		return nil, fmt.Errorf("RATE_LIMIT_RPS must be positive, got %d", cfg.RateLimitRPS)
	}

	return cfg, nil
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
