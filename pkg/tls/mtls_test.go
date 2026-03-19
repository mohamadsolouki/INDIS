package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

// selfSignedCert generates a self-signed ECDSA P-256 certificate for testing.
// Returns certPEM and keyPEM.
func selfSignedCert(t *testing.T, commonName string, isCA bool) (certPEM, keyPEM []byte) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		IsCA:         isCA,
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})
	return
}

// ---------------------------------------------------------------------------
// MutualTLSConfig
// ---------------------------------------------------------------------------

func TestMutualTLSConfig_ReturnsValidConfig(t *testing.T) {
	certPEM, keyPEM := selfSignedCert(t, "test-client", false)
	caCertPEM, _ := selfSignedCert(t, "test-ca", true)

	cfg, err := MutualTLSConfig(certPEM, keyPEM, caCertPEM)
	if err != nil {
		t.Fatalf("MutualTLSConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("MutualTLSConfig returned nil config")
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
	if cfg.RootCAs == nil {
		t.Error("RootCAs should not be nil")
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %d, want TLS 1.3 (%d)", cfg.MinVersion, tls.VersionTLS13)
	}
}

func TestMutualTLSConfig_RejectsInvalidCert(t *testing.T) {
	_, err := MutualTLSConfig([]byte("not-a-cert"), []byte("not-a-key"), []byte("not-a-ca"))
	if err == nil {
		t.Error("MutualTLSConfig should return error for invalid cert/key PEM")
	}
}

func TestMutualTLSConfig_RejectsInvalidCACert(t *testing.T) {
	certPEM, keyPEM := selfSignedCert(t, "client", false)
	_, err := MutualTLSConfig(certPEM, keyPEM, []byte("not-a-cert"))
	if err == nil {
		t.Error("MutualTLSConfig should return error for invalid CA PEM")
	}
}

func TestMutualTLSConfig_KeyMismatchReturnsError(t *testing.T) {
	certPEM, _ := selfSignedCert(t, "c1", false)
	_, keyPEM2 := selfSignedCert(t, "c2", false)
	caCertPEM, _ := selfSignedCert(t, "ca", true)
	_, err := MutualTLSConfig(certPEM, keyPEM2, caCertPEM)
	if err == nil {
		t.Error("MutualTLSConfig should return error when cert and key do not match")
	}
}

// ---------------------------------------------------------------------------
// ServerMutualTLSConfig
// ---------------------------------------------------------------------------

func TestServerMutualTLSConfig_ReturnsValidConfig(t *testing.T) {
	certPEM, keyPEM := selfSignedCert(t, "test-server", false)
	caCertPEM, _ := selfSignedCert(t, "test-ca", true)

	cfg, err := ServerMutualTLSConfig(certPEM, keyPEM, caCertPEM)
	if err != nil {
		t.Fatalf("ServerMutualTLSConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("ServerMutualTLSConfig returned nil config")
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
	if cfg.ClientCAs == nil {
		t.Error("ClientCAs should not be nil")
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuth = %v, want RequireAndVerifyClientCert", cfg.ClientAuth)
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("MinVersion = %d, want TLS 1.3 (%d)", cfg.MinVersion, tls.VersionTLS13)
	}
}

func TestServerMutualTLSConfig_RejectsInvalidCert(t *testing.T) {
	_, err := ServerMutualTLSConfig([]byte("bad"), []byte("bad"), []byte("bad"))
	if err == nil {
		t.Error("ServerMutualTLSConfig should return error for invalid cert/key PEM")
	}
}

func TestServerMutualTLSConfig_RejectsInvalidCACert(t *testing.T) {
	certPEM, keyPEM := selfSignedCert(t, "server", false)
	_, err := ServerMutualTLSConfig(certPEM, keyPEM, []byte("not-a-cert"))
	if err == nil {
		t.Error("ServerMutualTLSConfig should return error for invalid CA PEM")
	}
}

func TestServerMutualTLSConfig_KeyMismatchReturnsError(t *testing.T) {
	certPEM, _ := selfSignedCert(t, "s1", false)
	_, keyPEM2 := selfSignedCert(t, "s2", false)
	caCertPEM, _ := selfSignedCert(t, "ca", true)
	_, err := ServerMutualTLSConfig(certPEM, keyPEM2, caCertPEM)
	if err == nil {
		t.Error("ServerMutualTLSConfig should return error when cert and key do not match")
	}
}
