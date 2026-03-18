package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// revocationTTL is the 72-hour offline verifier cache window specified in PRD §4.6.
	revocationTTL = 72 * time.Hour

	// keyPrefix is prepended to every credential ID stored in Redis.
	keyPrefix = "indis:revoked:"
)

// RedisRevocationCache implements RevocationCache using Redis.
// Key format: "indis:revoked:<credentialID>"  (SET with EX 259200 = 72h)
type RedisRevocationCache struct {
	client *redis.Client
}

// NewRedisRevocationCache creates a RedisRevocationCache connected to addr.
// addr must be a host:port string (e.g. "localhost:6379").
func NewRedisRevocationCache(addr string) (*RedisRevocationCache, error) {
	if addr == "" {
		return nil, fmt.Errorf("cache: Redis address must not be empty")
	}
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisRevocationCache{client: client}, nil
}

// revocationKey returns the Redis key for credentialID.
func revocationKey(credentialID string) string {
	return keyPrefix + credentialID
}

// Revoke marks credentialID as revoked by writing key "indis:revoked:<credentialID>"
// with value "1" and an expiry of 72 hours (259200 seconds).
func (r *RedisRevocationCache) Revoke(ctx context.Context, credentialID string) error {
	if err := r.client.Set(ctx, revocationKey(credentialID), "1", revocationTTL).Err(); err != nil {
		return fmt.Errorf("cache: revoke %q: %w", credentialID, err)
	}
	return nil
}

// IsRevoked reports whether credentialID is present in the revocation cache.
// It returns false (not revoked) if the key has expired or was never set.
func (r *RedisRevocationCache) IsRevoked(ctx context.Context, credentialID string) (bool, error) {
	n, err := r.client.Exists(ctx, revocationKey(credentialID)).Result()
	if err != nil {
		return false, fmt.Errorf("cache: check revocation for %q: %w", credentialID, err)
	}
	return n > 0, nil
}

// Close releases the underlying Redis connection.
func (r *RedisRevocationCache) Close() error {
	return r.client.Close()
}
