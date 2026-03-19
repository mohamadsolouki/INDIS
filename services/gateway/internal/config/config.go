// Package config provides configuration loading for the gateway service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the configuration for the gateway service.
type Config struct {
	HTTPPort    int
	MetricsPort int

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

	// JWT authentication (HS256).
	// JWTSecret is the shared secret used to verify incoming Bearer tokens.
	JWTSecret string
	// APIKeys is the raw value of the API_KEYS env var: comma-separated "keyID:sha256hex" pairs.
	APIKeys string

	// Gateway's own Postgres DB for consent rules and data-export requests (PRD FR-008).
	DatabaseURL string

	// HTTP URLs for services that expose REST rather than gRPC.
	VerifierHTTPURL   string
	CardHTTPURL       string
	USSDHTTPURL       string
	GovPortalHTTPURL  string

	// CORS configuration.
	// CORSAllowedOrigins is a comma-separated list of allowed origins.
	// Use "*" to allow all origins (development only).
	CORSAllowedOrigins string
}

// Load reads configuration from environment variables with sane defaults.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:              envInt("HTTP_PORT", 8080),
		MetricsPort:           envInt("METRICS_PORT", 9109),
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
		JWTSecret:             envStr("JWT_SECRET", ""),
		APIKeys:               envStr("API_KEYS", ""),
		DatabaseURL:           envStr("DATABASE_URL", "postgres://indis:indis_dev_password@localhost:5432/indis_gateway?sslmode=disable"),
		VerifierHTTPURL:       envStr("VERIFIER_HTTP_URL", "http://localhost:9110"),
		CardHTTPURL:           envStr("CARD_HTTP_URL", "http://localhost:8400"),
		USSDHTTPURL:           envStr("USSD_HTTP_URL", "http://localhost:8300"),
		GovPortalHTTPURL:      envStr("GOV_PORTAL_HTTP_URL", "http://localhost:8200"),
		CORSAllowedOrigins:    envStr("CORS_ALLOWED_ORIGINS", "*"),
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
