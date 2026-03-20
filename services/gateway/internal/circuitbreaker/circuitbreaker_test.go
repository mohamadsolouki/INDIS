package circuitbreaker

import (
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedByDefault(t *testing.T) {
	cb := New()
	if !cb.Allow() {
		t.Fatal("expected Allow() == true for a new (Closed) circuit breaker")
	}
	if cb.CurrentState() != StateClosed {
		t.Fatalf("expected StateClosed, got %v", cb.CurrentState())
	}
}

func TestCircuitBreaker_OpensAfter5Failures(t *testing.T) {
	cb := New()

	for i := 0; i < 4; i++ {
		cb.Allow()
		cb.RecordFailure()
		if cb.CurrentState() != StateClosed {
			t.Fatalf("expected StateClosed after %d failures, got %v", i+1, cb.CurrentState())
		}
	}

	// 5th failure should open the breaker.
	cb.Allow()
	cb.RecordFailure()
	if cb.CurrentState() != StateOpen {
		t.Fatalf("expected StateOpen after 5 failures, got %v", cb.CurrentState())
	}

	// Allow() must return false when Open.
	if cb.Allow() {
		t.Fatal("expected Allow() == false when circuit is Open")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := New()
	// Override the open timeout to something tiny for the test.
	cb.openTimeout = 10 * time.Millisecond

	// Force the breaker open.
	for i := 0; i < 5; i++ {
		cb.Allow()
		cb.RecordFailure()
	}
	if cb.CurrentState() != StateOpen {
		t.Fatal("expected StateOpen")
	}

	// Before timeout: still blocked.
	if cb.Allow() {
		t.Fatal("expected Allow() == false before open timeout expires")
	}

	// Wait for the timeout.
	time.Sleep(20 * time.Millisecond)

	// After timeout: first Allow() should return true and transition to HalfOpen.
	if !cb.Allow() {
		t.Fatal("expected Allow() == true after open timeout (HalfOpen probe)")
	}
	if cb.CurrentState() != StateHalfOpen {
		t.Fatalf("expected StateHalfOpen, got %v", cb.CurrentState())
	}

	// Subsequent Allow() in HalfOpen state must return false.
	if cb.Allow() {
		t.Fatal("expected Allow() == false while HalfOpen (probe already in flight)")
	}
}

func TestCircuitBreaker_ClosesOnProbeSuccess(t *testing.T) {
	cb := New()
	cb.openTimeout = 10 * time.Millisecond

	// Force open.
	for i := 0; i < 5; i++ {
		cb.Allow()
		cb.RecordFailure()
	}

	// Wait for timeout → HalfOpen.
	time.Sleep(20 * time.Millisecond)
	cb.Allow() // transitions to HalfOpen

	// Successful probe closes the breaker.
	cb.RecordSuccess()
	if cb.CurrentState() != StateClosed {
		t.Fatalf("expected StateClosed after probe success, got %v", cb.CurrentState())
	}

	// Breaker should allow requests again.
	if !cb.Allow() {
		t.Fatal("expected Allow() == true after breaker closed via probe success")
	}
}
