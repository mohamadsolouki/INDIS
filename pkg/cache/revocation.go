// Package cache provides a Redis-backed revocation cache for INDIS credentials.
// Revoked credential IDs are stored with a 72-hour TTL so that offline verifiers
// can operate without a live connection to the revocation service (PRD §4.6).
package cache

import "context"

// RevocationCache tracks revoked credential IDs.
// Revoked status is set with a TTL of 72 hours (offline verifier cache window,
// PRD §4.6).
type RevocationCache interface {
	// Revoke marks credentialID as revoked for 72 hours.
	Revoke(ctx context.Context, credentialID string) error

	// IsRevoked reports whether credentialID is currently in the revocation cache.
	IsRevoked(ctx context.Context, credentialID string) (bool, error)

	// Close releases resources held by the cache implementation.
	Close() error
}
