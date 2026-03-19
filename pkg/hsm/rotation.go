// Package hsm — rotation.go
// Key rotation policy utilities for INDIS.
//
// Ref: INDIS PRD §4.3 — key lifecycle management
package hsm

import "time"

// RotationPolicy defines the conditions under which a key should be rotated.
type RotationPolicy struct {
	// MaxAge is the maximum duration a key should be used before rotation.
	// A zero value disables age-based rotation.
	MaxAge time.Duration

	// MaxSignatures is the maximum number of signing operations before the key
	// must be rotated. A zero value disables count-based rotation.
	// Note: signature counting requires an external counter; this field is
	// informational and is not enforced by NeedsRotation.
	MaxSignatures int64
}

// DefaultRotationPolicy returns the INDIS recommended rotation policy for the
// given key type.
//
// Policy schedule:
//   - Ed25519 signing keys:    90 days, 1 000 000 signatures
//   - ECDSA P-256 signing keys: 90 days, 1 000 000 signatures
//   - Dilithium3 signing keys: 180 days, 5 000 000 signatures
//   - AES-256-GCM encryption:  365 days (1 year), no count limit
//
// For unknown key types the most conservative policy (90-day rotation) is
// returned.
func DefaultRotationPolicy(keyType KeyType) RotationPolicy {
	switch keyType {
	case KeyTypeEd25519, KeyTypeECDSAP256:
		return RotationPolicy{
			MaxAge:        90 * 24 * time.Hour,
			MaxSignatures: 1_000_000,
		}
	case KeyTypeDilithium3:
		return RotationPolicy{
			MaxAge:        180 * 24 * time.Hour,
			MaxSignatures: 5_000_000,
		}
	case KeyTypeAES256:
		return RotationPolicy{
			MaxAge:        365 * 24 * time.Hour,
			MaxSignatures: 0, // symmetric keys are not used for signing
		}
	default:
		// Conservative fallback.
		return RotationPolicy{
			MaxAge:        90 * 24 * time.Hour,
			MaxSignatures: 0,
		}
	}
}

// NeedsRotation reports whether a key created at createdAt should be rotated
// according to policy.
//
// Only age-based rotation is evaluated here; callers that track signature
// counts should additionally compare the count against policy.MaxSignatures.
//
// A policy with MaxAge == 0 never triggers age-based rotation.
func NeedsRotation(createdAt time.Time, policy RotationPolicy) bool {
	if policy.MaxAge == 0 {
		return false
	}
	return time.Since(createdAt) >= policy.MaxAge
}
