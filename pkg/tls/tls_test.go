package tls

import (
	"os"
	"path/filepath"
	"testing"
)

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
