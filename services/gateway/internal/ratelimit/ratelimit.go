// Package ratelimit implements a per-IP token bucket rate limiter.
package ratelimit

import (
	"sync"
	"time"
)

// bucket holds the token state for one client IP.
type bucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
}

// Limiter is a concurrent per-key token bucket limiter.
type Limiter struct {
	rps     float64 // tokens added per second
	burst   float64 // maximum token capacity
	buckets sync.Map
}

// New returns a Limiter allowing rps requests/second with a burst of rps*2.
func New(rps int) *Limiter {
	return &Limiter{
		rps:   float64(rps),
		burst: float64(rps) * 2,
	}
}

// Allow returns true if the given key (IP address) is within the rate limit.
func (l *Limiter) Allow(key string) bool {
	v, _ := l.buckets.LoadOrStore(key, &bucket{tokens: l.burst, last: time.Now()})
	b := v.(*bucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now

	// Refill tokens based on elapsed time.
	b.tokens += elapsed * l.rps
	if b.tokens > l.burst {
		b.tokens = l.burst
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
