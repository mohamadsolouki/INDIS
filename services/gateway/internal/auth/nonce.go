// Package auth — nonce cache for JWT jti replay protection.
//
// NonceCache stores seen jti values until their associated JWT expires.
// A background goroutine calls GC every 5 minutes to evict expired entries,
// keeping memory usage bounded.
package auth

import (
	"sync"
	"time"
)

// NonceCache is a thread-safe store of JWT IDs (jti claims) used to detect
// token replay attacks. Each entry is retained until the token's exp time
// has passed, after which GC removes it.
type NonceCache struct {
	mu      sync.Mutex
	entries map[string]time.Time // jti → expiry time
}

// NewNonceCache creates an initialised NonceCache and starts the background GC
// goroutine. The goroutine runs for the lifetime of the process (no stop hook
// is needed: it holds no resources beyond memory).
func NewNonceCache() *NonceCache {
	nc := &NonceCache{
		entries: make(map[string]time.Time),
	}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			nc.GC()
		}
	}()
	return nc
}

// Check reports whether jti may be used. It returns false (replay detected) if
// the jti has already been recorded and its stored expiry has not yet passed.
// When true is returned the jti is recorded with the supplied exp so that
// subsequent calls with the same jti are rejected.
func (nc *NonceCache) Check(jti string, exp time.Time) bool {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if storedExp, seen := nc.entries[jti]; seen {
		if time.Now().Before(storedExp) {
			// Token still valid and already seen — replay.
			return false
		}
		// Stored entry has expired; the jti may be reused (new token).
	}
	nc.entries[jti] = exp
	return true
}

// GC removes entries whose expiry time is in the past.
func (nc *NonceCache) GC() {
	now := time.Now()
	nc.mu.Lock()
	defer nc.mu.Unlock()
	for jti, exp := range nc.entries {
		if now.After(exp) {
			delete(nc.entries, jti)
		}
	}
}
