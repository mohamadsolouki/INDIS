// Package crypto provides shared cryptographic utilities for INDIS services.
//
// Supported standards (PRD §4.3):
//   - Ed25519 / ECDSA P-256 (digital signatures)
//   - AES-256-GCM (data at rest)
//   - CRYSTALS-Dilithium (post-quantum, long-term credentials)
//
// All cryptographic libraries used MUST be audited open-source.
package crypto
