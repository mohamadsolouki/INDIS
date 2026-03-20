// Package circuitbreaker provides a minimal in-process circuit-breaker for
// protecting gateway→backend gRPC calls.
//
// State machine:
//
//	Closed ──(5 consecutive failures)──► Open ──(30 s timeout)──► HalfOpen
//	  ▲                                                                │
//	  └──────────────────(probe success)────────────────────────────►─┘
//	                           │
//	                    (probe failure)
//	                           │
//	                         Open (reopened)
package circuitbreaker

import (
	"sync"
	"time"
)

// State represents the circuit-breaker state.
type State int

const (
	// StateClosed means the backend is considered healthy; all calls pass through.
	StateClosed State = iota
	// StateOpen means the backend is failing; calls are rejected immediately.
	StateOpen
	// StateHalfOpen means one probe request is allowed to test backend recovery.
	StateHalfOpen
)

const (
	defaultFailureThreshold = 5
	defaultOpenTimeout      = 30 * time.Second
)

// CircuitBreaker tracks the health of a single backend service and prevents
// requests from reaching a repeatedly-failing backend.
//
// Each backend service should own its own CircuitBreaker instance.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            State
	consecutiveFails int
	openedAt         time.Time

	failureThreshold int
	openTimeout      time.Duration
}

// New returns a new CircuitBreaker in the Closed state with default thresholds
// (5 consecutive failures, 30 s open timeout).
func New() *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: defaultFailureThreshold,
		openTimeout:      defaultOpenTimeout,
	}
}

// Allow reports whether the next call should be forwarded to the backend.
//
//   - Closed    → true (always allowed)
//   - Open      → false, unless the open timeout has elapsed, in which case
//     the breaker transitions to HalfOpen and returns true for exactly one
//     probe request.
//   - HalfOpen  → false (probe already in flight; subsequent calls are blocked)
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		if time.Since(cb.openedAt) >= cb.openTimeout {
			// Transition to HalfOpen to allow one probe.
			cb.state = StateHalfOpen
			return true
		}
		return false

	case StateHalfOpen:
		// Only the single probe is allowed; subsequent callers are blocked.
		return false

	default:
		return false
	}
}

// RecordSuccess records a successful downstream call.
// If the breaker is HalfOpen (probe succeeded), it transitions back to Closed.
// In the Closed state it resets the consecutive failure counter.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		// Probe succeeded — backend is healthy again.
		cb.state = StateClosed
		cb.consecutiveFails = 0
	case StateClosed:
		cb.consecutiveFails = 0
	}
	// In Open state a success is unexpected (Allow returned false), ignore.
}

// RecordFailure records a failed downstream call.
// In the Closed state it increments the consecutive failure counter and opens the
// breaker after reaching the threshold. In HalfOpen (probe failed), it immediately
// reopens the breaker.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.consecutiveFails++
		if cb.consecutiveFails >= cb.failureThreshold {
			cb.state = StateOpen
			cb.openedAt = time.Now()
		}
	case StateHalfOpen:
		// Probe failed — reopen immediately.
		cb.state = StateOpen
		cb.openedAt = time.Now()
	}
}

// State returns the current state of the circuit breaker (for observability).
func (cb *CircuitBreaker) CurrentState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
