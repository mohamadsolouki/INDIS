// Package tls — mtls.go
// Pure-PEM mutual TLS helpers that work directly with tls.Config rather than
// file paths. These functions complement the file-based helpers in tls.go and
// are intended for services that receive certificate material from Vault, HSM,
// or Kubernetes secrets rather than from the local filesystem.
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

// MutualTLSConfig builds a tls.Config suitable for a gRPC (or plain TLS)
// client that presents a client certificate for mutual authentication.
//
// certPEM is the PEM-encoded client certificate.
// keyPEM is the PEM-encoded client private key matching certPEM.
// caCertPEM is the PEM-encoded CA certificate used to verify the server's
// certificate.
//
// The returned config enforces TLS 1.3 as the minimum version.
//
// Example:
//
//	cfg, err := tls.MutualTLSConfig(certPEM, keyPEM, caPEM)
//	creds := credentials.NewTLS(cfg)
//	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
func MutualTLSConfig(certPEM, keyPEM, caCertPEM []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("tls: mutual tls client: parse key pair: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("tls: mutual tls client: no valid certificates in CA PEM")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// ServerMutualTLSConfig builds a tls.Config for a server that requires
// clients to present a valid certificate signed by the given CA.
//
// certPEM is the PEM-encoded server certificate.
// keyPEM is the PEM-encoded server private key matching certPEM.
// caCertPEM is the PEM-encoded CA certificate used to verify connecting
// clients. Clients that do not present a certificate signed by this CA will
// have their connection rejected.
//
// The returned config enforces TLS 1.3 as the minimum version and sets
// ClientAuth to tls.RequireAndVerifyClientCert.
//
// Example:
//
//	cfg, err := tls.ServerMutualTLSConfig(certPEM, keyPEM, caPEM)
//	creds := credentials.NewTLS(cfg)
//	srv := grpc.NewServer(grpc.Creds(creds))
func ServerMutualTLSConfig(certPEM, keyPEM, caCertPEM []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("tls: mutual tls server: parse key pair: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("tls: mutual tls server: no valid certificates in CA PEM")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}, nil
}
