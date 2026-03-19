// Package crypto — pqc.go
// Post-quantum cryptography migration utilities for INDIS.
// Ref: INDIS PRD §4.3 — NIST PQC, FIPS 203/204/205
package crypto

import "fmt"

// MigrationNeeded returns true if the key pair uses a pre-quantum algorithm
// and should be migrated to a post-quantum alternative.
//
// Ed25519 and ECDSA P-256 are considered pre-quantum because sufficiently
// large quantum computers running Shor's algorithm can break them.
// Dilithium3 is quantum-resistant per NIST FIPS 204.
func MigrationNeeded(keyType KeyType) bool {
	return keyType == KeyTypeEd25519 || keyType == KeyTypeECDSAP256
}

// RecommendedKeyType returns the recommended key type for a given use case.
//
// Policy (per INDIS PRD §4.3):
//   - "long-term" or "credential": Dilithium3 (post-quantum, for credentials
//     with lifetimes exceeding 5 years).
//   - "operational" or "session" or any other value: Ed25519 (fast, compact,
//     suitable for short-lived operational keys < 1 year).
//
// When the real Dilithium library is integrated, long-term keys should be
// re-issued using GenerateDilithiumKeyPair.
func RecommendedKeyType(useCase string) KeyType {
	switch useCase {
	case "long-term", "credential":
		return KeyTypeDilithium3
	default:
		return KeyTypeEd25519
	}
}

// MigrateKeyPair creates a new Dilithium3 key pair from an existing Ed25519
// key pair and returns it alongside the original, enabling a dual-signature
// transition period where both algorithms sign outgoing credentials until all
// verifiers have been upgraded to support Dilithium3.
//
// The caller is responsible for:
//  1. Issuing new credentials signed by both the returned Dilithium pair and
//     the existing Ed25519 key pair during the transition window.
//  2. Revoking the original Ed25519 key once all dependent verifiers support
//     Dilithium3.
//
// Returns an error if existing is nil or has an unsupported key type.
func MigrateKeyPair(existing *KeyPair) (*DilithiumKeyPair, error) {
	if existing == nil {
		return nil, fmt.Errorf("crypto: migrate key pair: existing key pair is nil")
	}
	if !MigrationNeeded(existing.Type) {
		return nil, fmt.Errorf("crypto: migrate key pair: key type %q does not require migration", existing.Type)
	}
	newPair, err := GenerateDilithiumKeyPair()
	if err != nil {
		return nil, fmt.Errorf("crypto: migrate key pair: %w", err)
	}
	return newPair, nil
}
