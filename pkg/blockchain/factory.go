// Package blockchain — factory for creating the correct BlockchainAdapter from environment.
package blockchain

import (
	"encoding/base64"
	"fmt"
	"os"
)

// NewAdapter creates a BlockchainAdapter driven entirely by environment variables.
// This is the canonical entry point for application services; no service should
// call NewMockAdapter or NewFabricAdapter directly in production code.
//
// Environment variables:
//
//	BLOCKCHAIN_TYPE           — "mock" (default) or "fabric"
//
// When BLOCKCHAIN_TYPE=fabric, the following variables are read:
//
//	FABRIC_GATEWAY_URL        — required; base URL of the Fabric peer gateway REST API
//	                            e.g. "http://peer0.org1.indis.ir:7080"
//	FABRIC_CHANNEL_ID         — optional default channel ID (default: "did-registry-channel")
//	FABRIC_MSP_ID             — optional MSP identifier (default: "niaMSP")
//	FABRIC_CERT_PEM           — optional; base64-encoded PEM client certificate for mTLS
//	FABRIC_KEY_PEM            — optional; base64-encoded PEM private key for mTLS
//	FABRIC_TLS_CA_CERT_PEM    — optional; base64-encoded PEM TLS CA certificate
//
// PEM values are accepted as base64 to allow safe embedding in container environment
// variables without newline escaping issues.
//
// Panics if BLOCKCHAIN_TYPE=fabric and FABRIC_GATEWAY_URL is not set, or if TLS
// configuration is invalid. This is intentional: a misconfigured blockchain adapter
// must prevent service startup rather than silently falling back to mock behaviour.
func NewAdapter() BlockchainAdapter {
	adapterType := os.Getenv("BLOCKCHAIN_TYPE")
	if adapterType == "" {
		adapterType = "mock"
	}

	switch adapterType {
	case "mock":
		return NewMockAdapter()

	case "fabric":
		cfg, err := fabricConfigFromEnv()
		if err != nil {
			panic(fmt.Sprintf("blockchain.NewAdapter: invalid Fabric configuration: %v", err))
		}
		adapter, err := NewFabricAdapter(cfg)
		if err != nil {
			panic(fmt.Sprintf("blockchain.NewAdapter: failed to create Fabric adapter: %v", err))
		}
		return adapter

	default:
		panic(fmt.Sprintf("blockchain.NewAdapter: unknown BLOCKCHAIN_TYPE %q; valid values are: mock, fabric", adapterType))
	}
}

// fabricConfigFromEnv reads Fabric adapter configuration from environment variables.
// PEM values may be provided as raw PEM or as base64-encoded PEM strings; the
// function auto-detects by attempting base64 decoding first.
func fabricConfigFromEnv() (FabricConfig, error) {
	gatewayURL := os.Getenv("FABRIC_GATEWAY_URL")
	if gatewayURL == "" {
		return FabricConfig{}, fmt.Errorf("FABRIC_GATEWAY_URL must be set when BLOCKCHAIN_TYPE=fabric")
	}

	channelID := os.Getenv("FABRIC_CHANNEL_ID")
	if channelID == "" {
		channelID = "did-registry-channel"
	}

	mspID := os.Getenv("FABRIC_MSP_ID")
	if mspID == "" {
		mspID = "niaMSP"
	}

	certPEM, err := decodePEMEnvVar("FABRIC_CERT_PEM")
	if err != nil {
		return FabricConfig{}, fmt.Errorf("FABRIC_CERT_PEM: %w", err)
	}

	keyPEM, err := decodePEMEnvVar("FABRIC_KEY_PEM")
	if err != nil {
		return FabricConfig{}, fmt.Errorf("FABRIC_KEY_PEM: %w", err)
	}

	caCertPEM, err := decodePEMEnvVar("FABRIC_TLS_CA_CERT_PEM")
	if err != nil {
		return FabricConfig{}, fmt.Errorf("FABRIC_TLS_CA_CERT_PEM: %w", err)
	}

	return FabricConfig{
		GatewayURL:   gatewayURL,
		ChannelID:    channelID,
		MSPId:        mspID,
		CertPEM:      certPEM,
		KeyPEM:       keyPEM,
		TLSCACertPEM: caCertPEM,
	}, nil
}

// decodePEMEnvVar reads an environment variable and returns its value as a string.
// If the value is valid base64, it is decoded first (to support base64-encoded PEM
// blocks stored in container secrets). If decoding fails or the value is empty,
// the raw value is returned as-is.
func decodePEMEnvVar(envKey string) (string, error) {
	raw := os.Getenv(envKey)
	if raw == "" {
		return "", nil
	}

	// Attempt base64 decode; if it succeeds and the result looks like PEM, use it.
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil && len(decoded) > 0 {
		return string(decoded), nil
	}

	// Fall back to treating the raw value as a PEM string.
	return raw, nil
}
