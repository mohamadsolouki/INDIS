package auth

import (
	"testing"
	"time"
)

func TestNonceCache_FirstUseAllowed(t *testing.T) {
	nc := &NonceCache{entries: make(map[string]time.Time)}
	exp := time.Now().Add(time.Hour)
	if !nc.Check("jti-abc", exp) {
		t.Fatal("expected first use of a jti to be allowed")
	}
}

func TestNonceCache_ReplayRejected(t *testing.T) {
	nc := &NonceCache{entries: make(map[string]time.Time)}
	exp := time.Now().Add(time.Hour)

	if !nc.Check("jti-xyz", exp) {
		t.Fatal("expected first use to be allowed")
	}
	// Second use with the same jti while the token is still valid → replay.
	if nc.Check("jti-xyz", exp) {
		t.Fatal("expected replay of the same jti to be rejected")
	}
}

func TestNonceCache_ExpiredEntryAllowsReuse(t *testing.T) {
	nc := &NonceCache{entries: make(map[string]time.Time)}

	// Store an already-expired jti directly.
	expiredExp := time.Now().Add(-time.Second)
	nc.entries["jti-old"] = expiredExp

	// A new token with the same jti but a future expiry should be allowed
	// because the old entry has expired.
	newExp := time.Now().Add(time.Hour)
	if !nc.Check("jti-old", newExp) {
		t.Fatal("expected reuse of an expired jti to be allowed")
	}
}
