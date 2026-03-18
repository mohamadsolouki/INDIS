package config

import "testing"

func TestLoad_RejectsPartialClientCertConfig(t *testing.T) {
	t.Setenv("BACKEND_TLS_MODE", "tls")
	t.Setenv("BACKEND_CA_FILE", "/tmp/ca.pem")
	t.Setenv("BACKEND_CLIENT_CERT_FILE", "/tmp/client.crt")
	t.Setenv("BACKEND_CLIENT_KEY_FILE", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for partial client cert/key config")
	}
}

func TestLoad_RejectsClientCertOutsideTLSMode(t *testing.T) {
	t.Setenv("BACKEND_TLS_MODE", "plaintext")
	t.Setenv("BACKEND_CLIENT_CERT_FILE", "/tmp/client.crt")
	t.Setenv("BACKEND_CLIENT_KEY_FILE", "/tmp/client.key")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when client cert config is used outside tls mode")
	}
}

func TestLoad_AcceptsTLSWithClientCertPair(t *testing.T) {
	t.Setenv("BACKEND_TLS_MODE", "tls")
	t.Setenv("BACKEND_CA_FILE", "/tmp/ca.pem")
	t.Setenv("BACKEND_CLIENT_CERT_FILE", "/tmp/client.crt")
	t.Setenv("BACKEND_CLIENT_KEY_FILE", "/tmp/client.key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.BackendClientCertFile == "" || cfg.BackendClientKeyFile == "" {
		t.Fatal("expected client cert and key to be set")
	}
}
