package tls

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServerOptionsFromEnv_DefaultsToPlaintext(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "")

	opts, err := ServerOptionsFromEnv()
	if err != nil {
		t.Fatalf("ServerOptionsFromEnv returned error: %v", err)
	}
	if len(opts) != 0 {
		t.Fatalf("expected no options in plaintext mode, got %d", len(opts))
	}
}

func TestServerOptionsFromEnv_RejectsInvalidMode(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "invalid")

	_, err := ServerOptionsFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid GRPC_TLS_MODE")
	}
}

func TestServerOptionsFromEnv_RequiresCertAndKeyInTLSMode(t *testing.T) {
	t.Setenv("GRPC_TLS_MODE", "tls")
	t.Setenv("TLS_CERT_FILE", "")
	t.Setenv("TLS_KEY_FILE", "")

	_, err := ServerOptionsFromEnv()
	if err == nil {
		t.Fatal("expected error when TLS_CERT_FILE/TLS_KEY_FILE are missing")
	}
}

func TestLoadClientMTLS_RequiresReadableFiles(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "missing.pem")

	_, err := LoadClientMTLS(missing, missing, missing)
	if err == nil {
		t.Fatal("expected error when CA/cert/key files are missing")
	}
}

func TestLoadClientTLSInsecureSkipVerify_ReturnsCredentials(t *testing.T) {
	creds := LoadClientTLSInsecureSkipVerify()
	if creds == nil {
		t.Fatal("expected non-nil transport credentials")
	}
}

func TestLoadClientTLS_RejectsInvalidCAData(t *testing.T) {
	tmp := t.TempDir()
	caFile := filepath.Join(tmp, "ca.pem")
	if err := os.WriteFile(caFile, []byte("not-a-certificate"), 0o600); err != nil {
		t.Fatalf("write temp ca file: %v", err)
	}

	_, err := LoadClientTLS(caFile)
	if err == nil {
		t.Fatal("expected error for invalid CA data")
	}
}
