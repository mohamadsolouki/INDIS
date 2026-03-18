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

	// Backend gRPC transport security mode.
	// Supported values:
	//   - plaintext: no TLS (local/dev compatibility)
	//   - tls: verify backend certs using BackendCAFile
	//   - tls_insecure_skip_verify: TLS encryption without cert verification (dev only)
	BackendTLSMode        string
	BackendCAFile         string
	BackendClientCertFile string
	BackendClientKeyFile  string
}

// Load reads configuration from environment variables with sane defaults.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:              envInt("HTTP_PORT", 8080),
		IdentityAddr:          envStr("IDENTITY_ADDR", "localhost:50051"),
		CredentialAddr:        envStr("CREDENTIAL_ADDR", "localhost:50052"),
		EnrollmentAddr:        envStr("ENROLLMENT_ADDR", "localhost:50053"),
		BiometricAddr:         envStr("BIOMETRIC_ADDR", "localhost:50054"),
		AuditAddr:             envStr("AUDIT_ADDR", "localhost:50055"),
		NotificationAddr:      envStr("NOTIFICATION_ADDR", "localhost:50056"),
		ElectoralAddr:         envStr("ELECTORAL_ADDR", "localhost:50057"),
		JusticeAddr:           envStr("JUSTICE_ADDR", "localhost:50058"),
		RateLimitRPS:          envInt("RATE_LIMIT_RPS", 100),
		BackendTLSMode:        envStr("BACKEND_TLS_MODE", "plaintext"),
		BackendCAFile:         envStr("BACKEND_CA_FILE", ""),
		BackendClientCertFile: envStr("BACKEND_CLIENT_CERT_FILE", ""),
		BackendClientKeyFile:  envStr("BACKEND_CLIENT_KEY_FILE", ""),
	}

	if cfg.RateLimitRPS <= 0 {
		return nil, fmt.Errorf("RATE_LIMIT_RPS must be positive, got %d", cfg.RateLimitRPS)
	}

	switch cfg.BackendTLSMode {
	case "plaintext", "tls", "tls_insecure_skip_verify":
		// valid
	default:
		return nil, fmt.Errorf("BACKEND_TLS_MODE must be one of plaintext|tls|tls_insecure_skip_verify, got %q", cfg.BackendTLSMode)
	}

	if cfg.BackendTLSMode == "tls" && cfg.BackendCAFile == "" {
		return nil, fmt.Errorf("BACKEND_CA_FILE is required when BACKEND_TLS_MODE=tls")
	}

	if (cfg.BackendClientCertFile == "") != (cfg.BackendClientKeyFile == "") {
		return nil, fmt.Errorf("BACKEND_CLIENT_CERT_FILE and BACKEND_CLIENT_KEY_FILE must be set together")
	}
	if cfg.BackendTLSMode != "tls" && cfg.BackendClientCertFile != "" {
		return nil, fmt.Errorf("BACKEND_CLIENT_CERT_FILE requires BACKEND_TLS_MODE=tls")
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
