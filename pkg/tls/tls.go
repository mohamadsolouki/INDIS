// Package tls provides mTLS certificate helpers for INDIS gRPC services.
// It wraps the standard crypto/tls and x509 packages to produce
// google.golang.org/grpc/credentials.TransportCredentials values that can be
// passed directly to grpc.NewServer or grpc.Dial.
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

// LoadServerTLS loads server-side mTLS credentials for a gRPC server.
//
// certFile is the path to the PEM-encoded server certificate.
// keyFile is the path to the PEM-encoded server private key.
// caFile is the path to the PEM-encoded CA certificate used to verify
// connecting clients. When caFile is empty, client certificates are not
// requested or verified — this mode must only be used in development
// environments.
//
// Example (production):
//
//	creds, err := tls.LoadServerTLS("server.crt", "server.key", "ca.crt")
func LoadServerTLS(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("tls: load server key pair: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("tls: read CA file %q: %w", caFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("tls: no valid certificates found in CA file %q", caFile)
		}
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return credentials.NewTLS(cfg), nil
}

// LoadClientTLS loads client-side TLS credentials for a gRPC client.
//
// caFile is the path to the PEM-encoded CA certificate used to verify the
// server's certificate. The returned credentials enforce server certificate
// verification.
//
// Example:
//
//	creds, err := tls.LoadClientTLS("ca.crt")
//	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
func LoadClientTLS(caFile string) (credentials.TransportCredentials, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("tls: read CA file %q: %w", caFile, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("tls: no valid certificates found in CA file %q", caFile)
	}

	cfg := &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS13,
	}
	return credentials.NewTLS(cfg), nil
}

// LoadClientMTLS loads client-side TLS credentials for a gRPC client and
// presents a client certificate for mutual authentication.
//
// caFile is the PEM-encoded CA certificate used to verify the server
// certificate. certFile and keyFile are the PEM-encoded client certificate and
// private key.
func LoadClientMTLS(caFile, certFile, keyFile string) (credentials.TransportCredentials, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("tls: read CA file %q: %w", caFile, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("tls: no valid certificates found in CA file %q", caFile)
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("tls: load client key pair: %w", err)
	}

	cfg := &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}
	return credentials.NewTLS(cfg), nil
}

// LoadClientTLSInsecureSkipVerify returns client TLS credentials that skip
// server certificate verification entirely.
//
// WARNING: This function must only be used in local development or testing
// environments. Using it in production exposes the connection to
// man-in-the-middle attacks.
func LoadClientTLSInsecureSkipVerify() credentials.TransportCredentials {
	cfg := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // development-only helper; intentional
		MinVersion:         tls.VersionTLS12,
	}
	return credentials.NewTLS(cfg)
}
