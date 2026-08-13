package circuitbreaker

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestBreaker_ConsecutiveFailures(t *testing.T) {
	breaker := New(WithFailureCount(3))

	// Failures below the threshold keep the breaker closed.
	for i := 0; i < 2; i++ {
		cb, err := breaker.Allow()
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
		cb(false)
		if breaker.State() != StateClosed {
			t.Fatalf("request %d: expected closed, got %s", i, breaker.State())
		}
	}

	// Third consecutive failure trips it open.
	cb, err := breaker.Allow()
	if err != nil {
		t.Fatalf("request 3: unexpected error: %v", err)
	}
	cb(false)
	if breaker.State() != StateOpen {
		t.Fatalf("expected open after 3 failures, got %s", breaker.State())
	}

	// Open rejects requests.
	if _, err := breaker.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestBreaker_CoolDownAndRecovery(t *testing.T) {
	now := time.Unix(1000, 0)
	breaker := New(
		WithFailureCount(1),
		WithTimeout(30*time.Second),
		WithNow(func() time.Time { return now }),
	)

	// Trip open.
	cb, _ := breaker.Allow()
	cb(false)
	if breaker.State() != StateOpen {
		t.Fatalf("expected open, got %s", breaker.State())
	}

	// Still in cool-down: rejected.
	if _, err := breaker.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen during cool-down, got %v", err)
	}

	// After cool-down: HalfOpen probe allowed.
	now = now.Add(31 * time.Second)
	cb, err := breaker.Allow()
	if err != nil {
		t.Fatalf("expected probe allowed after cool-down, got %v", err)
	}
	if breaker.State() != StateHalfOpen {
		t.Fatalf("expected half-open, got %s", breaker.State())
	}

	// Probe succeeds → recovered to Closed.
	cb(true)
	if breaker.State() != StateClosed {
		t.Fatalf("expected closed after successful probe, got %s", breaker.State())
	}
}

func TestBreaker_HalfOpenFailureReopens(t *testing.T) {
	now := time.Unix(1000, 0)
	breaker := New(
		WithFailureCount(1),
		WithTimeout(30*time.Second),
		WithNow(func() time.Time { return now }),
	)

	cb, _ := breaker.Allow()
	cb(false) // → Open
	now = now.Add(31 * time.Second)

	cb, err := breaker.Allow() // → HalfOpen probe
	if err != nil {
		t.Fatalf("expected probe allowed, got %v", err)
	}
	cb(false) // probe fails → back to Open

	if breaker.State() != StateOpen {
		t.Fatalf("expected open after failed probe, got %s", breaker.State())
	}
	if _, err := breaker.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen after failed probe, got %v", err)
	}
}

func TestBreaker_HalfOpenMaxRequests(t *testing.T) {
	now := time.Unix(1000, 0)
	breaker := New(
		WithFailureCount(1),
		WithTimeout(time.Second),
		WithMaxRequests(2),
		WithNow(func() time.Time { return now }),
	)

	cb, _ := breaker.Allow()
	cb(false) // → Open
	now = now.Add(2 * time.Second)

	// HalfOpen allows up to 2 probes.
	for i := 0; i < 2; i++ {
		if _, err := breaker.Allow(); err != nil {
			t.Fatalf("probe %d: unexpected error: %v", i, err)
		}
	}
	// Third request rejected while probes are in flight.
	if _, err := breaker.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen when probes exhausted, got %v", err)
	}
}

func TestBreaker_RatioThreshold(t *testing.T) {
	t.Run("trips above ratio with enough samples", func(t *testing.T) {
		breaker := New(
			WithFailureCount(0), // disable consecutive-failure policy
			WithFailureRatio(0.5),
			WithMinRequests(5),
		)

		// 3 successes + 3 failures: ratio reaches 50% only on the last
		// request (3/6), earlier prefixes stay below (2/5, 1/4).
		outcomes := []bool{true, true, true, false, false, false}
		for i, ok := range outcomes {
			cb, err := breaker.Allow()
			if err != nil {
				t.Fatalf("request %d: unexpected error: %v", i, err)
			}
			cb(ok)
		}

		if breaker.State() != StateOpen {
			t.Fatalf("expected open after 50%% failures, got %s", breaker.State())
		}
	})

	t.Run("min requests guards cold start", func(t *testing.T) {
		breaker := New(
			WithFailureCount(0),
			WithFailureRatio(0.5),
			WithMinRequests(10),
		)

		// 2 failures of 2 requests = 100% but below minRequests → stays closed.
		for i := 0; i < 2; i++ {
			cb, _ := breaker.Allow()
			cb(false)
		}
		if breaker.State() != StateClosed {
			t.Fatalf("expected closed below min requests, got %s", breaker.State())
		}
	})
}

func TestBreaker_StateChangeCallback(t *testing.T) {
	now := time.Unix(1000, 0)
	var transitions []string
	breaker := New(
		WithFailureCount(1),
		WithTimeout(time.Second),
		WithNow(func() time.Time { return now }),
		WithOnStateChange(func(old, new State) {
			transitions = append(transitions, old.String()+"->"+new.String())
		}),
	)

	cb, _ := breaker.Allow()
	cb(false) // closed -> open
	now = now.Add(2 * time.Second)
	cb, _ = breaker.Allow() // open -> half-open (probe)
	cb(true)                // half-open -> closed

	want := []string{"closed->open", "open->half-open", "half-open->closed"}
	if len(transitions) != len(want) {
		t.Fatalf("expected %d transitions, got %d: %v", len(want), len(transitions), transitions)
	}
	for i, w := range want {
		if transitions[i] != w {
			t.Errorf("transition %d = %q, want %q", i, transitions[i], w)
		}
	}
}

func TestBreaker_Concurrent(t *testing.T) {
	breaker := New(WithFailureCount(5))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cb, err := breaker.Allow()
			if err != nil {
				return // rejected, fine
			}
			cb(n%2 == 0) // alternate success/failure
		}(i)
	}
	wg.Wait()

	// No panic, and the state must be one of the three valid states.
	s := breaker.State()
	if s != StateClosed && s != StateOpen && s != StateHalfOpen {
		t.Fatalf("invalid state %s", s)
	}
}

func TestBreaker_WindowSlides(t *testing.T) {
	now := time.Unix(1000, 0)
	breaker := New(
		WithFailureCount(3),
		WithInterval(10*time.Second),
		WithNow(func() time.Time { return now }),
	)

	// Two failures, then the window slides past them.
	cb, _ := breaker.Allow()
	cb(false)
	cb, _ = breaker.Allow()
	cb(false)

	// Advance 11s: old failures fall out of the window.
	now = now.Add(11 * time.Second)
	cb, err := breaker.Allow()
	if err != nil {
		t.Fatalf("expected allowed after window slide, got %v", err)
	}
	cb(false) // one fresh failure, below threshold
	if breaker.State() != StateClosed {
		t.Fatalf("expected closed after window slide, got %s", breaker.State())
	}
}
