// Package circuitbreaker implements a stateful circuit breaker with a
// sliding-window failure counter, following the classic three-state machine
// (Closed → Open → HalfOpen) from Google SRE / Resilience4j.
//
// The breaker is local to each instance (fast, no coordination latency).
// It is safe for concurrent use.
package circuitbreaker

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ErrCircuitOpen is returned when the breaker is open and rejects requests
// without attempting them.
var ErrCircuitOpen = errors.New("circuitbreaker: circuit is open")

// State is the breaker state.
type State int32

const (
	// StateClosed allows requests and counts failures.
	StateClosed State = iota
	// StateOpen rejects all requests (fast fail) until the cool-down elapses.
	StateOpen
	// StateHalfOpen allows a limited number of probe requests to test recovery.
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return fmt.Sprintf("unknown(%d)", int32(s))
	}
}

// Callback reports the outcome of an allowed request back to the breaker.
// Call exactly once with success=true/false after the request completes.
type Callback func(success bool)

// StateChangeFunc is invoked whenever the breaker transitions state.
type StateChangeFunc func(old, new State)

// Breaker is a circuit breaker guarding a downstream service or dependency.
type Breaker struct {
	mu sync.Mutex

	// Configuration.
	maxRequests   int           // probe requests allowed in HalfOpen
	interval      time.Duration // Closed counting window
	timeout       time.Duration // Open cool-down before HalfOpen
	failureCount  int           // consecutive-failure threshold (0 disables)
	failureRatio  float64       // failure-ratio threshold (0 disables)
	minRequests   int           // minimum samples before ratio trips
	onStateChange StateChangeFunc
	now           func() time.Time

	// State.
	state     State
	openedAt  time.Time
	halfOpens int

	// Sliding-window statistics.
	window *slidingWindow
}

// BreakerOption configures a Breaker.
type BreakerOption func(*Breaker)

// New creates a circuit breaker with sensible defaults:
// 5 consecutive failures or 50% failure over the window trips it open.
func New(opts ...BreakerOption) *Breaker {
	b := &Breaker{
		maxRequests:  1,
		interval:     10 * time.Second,
		timeout:      30 * time.Second,
		failureCount: 5,
		failureRatio: 0,
		minRequests:  20,
		state:        StateClosed,
		window:       newSlidingWindow(10*time.Second, time.Second),
		now:          time.Now,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// WithMaxRequests sets how many probe requests are allowed in HalfOpen state.
func WithMaxRequests(n int) BreakerOption {
	return func(b *Breaker) { b.maxRequests = n }
}

// WithInterval sets the Closed-state counting window.
func WithInterval(d time.Duration) BreakerOption {
	return func(b *Breaker) { b.interval = d }
}

// WithTimeout sets how long the breaker stays Open before probing.
func WithTimeout(d time.Duration) BreakerOption {
	return func(b *Breaker) { b.timeout = d }
}

// WithFailureCount sets the consecutive-failure threshold. A window with
// failureCount consecutive failures trips the breaker. 0 disables this policy.
func WithFailureCount(n int) BreakerOption {
	return func(b *Breaker) { b.failureCount = n }
}

// WithFailureRatio sets the failure-ratio threshold (0..1) over the window.
// The breaker trips when failure/total >= ratio AND total >= MinRequests.
// 0 disables this policy.
func WithFailureRatio(r float64) BreakerOption {
	return func(b *Breaker) { b.failureRatio = r }
}

// WithMinRequests sets the minimum request count before the ratio policy can
// trip the breaker (guards against cold-start false positives).
func WithMinRequests(n int) BreakerOption {
	return func(b *Breaker) { b.minRequests = n }
}

// WithOnStateChange registers a callback invoked on every state transition.
func WithOnStateChange(fn StateChangeFunc) BreakerOption {
	return func(b *Breaker) { b.onStateChange = fn }
}

// WithNow overrides the clock (for testing).
func WithNow(now func() time.Time) BreakerOption {
	return func(b *Breaker) { b.now = now }
}

// State returns the current breaker state.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// Allow checks whether a request may proceed. When nil error, the returned
// Callback MUST be invoked after the request completes to feed the breaker.
func (b *Breaker) Allow() (Callback, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		// Fall through: allow.
	case StateHalfOpen:
		if b.halfOpens >= b.maxRequests {
			return nil, ErrCircuitOpen
		}
		b.halfOpens++
	case StateOpen:
		if b.now().Sub(b.openedAt) < b.timeout {
			return nil, ErrCircuitOpen
		}
		// Cool-down elapsed: move to HalfOpen and allow a probe.
		b.toState(StateHalfOpen)
		b.halfOpens++
	}
	return b.callback, nil
}

// callback reports a request outcome and drives the state machine.
func (b *Breaker) callback(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		b.window.record(success, b.now())
		if !success && b.shouldTrip() {
			b.toState(StateOpen)
		}
	case StateHalfOpen:
		if success {
			// Probe succeeded: service recovered.
			b.toState(StateClosed)
		} else {
			b.toState(StateOpen)
		}
	case StateOpen:
		// Outcomes of pre-open requests are ignored.
	}
}

// shouldTrip decides whether the Closed-state statistics breach a threshold.
func (b *Breaker) shouldTrip() bool {
	success, failure := b.window.snapshot(b.now())
	total := success + failure

	if b.failureCount > 0 && failure >= uint64(b.failureCount) {
		return true
	}
	if b.failureRatio > 0 && total >= uint64(b.minRequests) {
		if float64(failure)/float64(total) >= b.failureRatio {
			return true
		}
	}
	return false
}

// toState transitions the breaker and fires the state-change callback.
func (b *Breaker) toState(s State) {
	old := b.state
	b.state = s
	switch s {
	case StateOpen:
		b.openedAt = b.now()
	case StateHalfOpen:
		b.halfOpens = 0
	}
	b.window.reset()
	if b.onStateChange != nil {
		b.onStateChange(old, s)
	}
}

// bucket is one time slice of the sliding window.
type bucket struct {
	ts      int64 // window-slot number (now / bucketDur)
	success atomic.Uint64
	failure atomic.Uint64
}

// slidingWindow counts successes/failures over a fixed interval using a
// ring of time-sliced buckets. Buckets are reset on first write of a new
// slot; reads filter out slots outside the window.
type slidingWindow struct {
	buckets   []bucket
	bucketDur time.Duration
}

func newSlidingWindow(interval, bucketDur time.Duration) *slidingWindow {
	n := int(interval / bucketDur)
	if n < 1 {
		n = 1
	}
	return &slidingWindow{
		buckets:   make([]bucket, n),
		bucketDur: bucketDur,
	}
}

func (w *slidingWindow) slot(now time.Time) int64 {
	return now.UnixNano() / w.bucketDur.Nanoseconds()
}

// record counts an outcome into the bucket for the current slot.
func (w *slidingWindow) record(success bool, now time.Time) {
	slot := w.slot(now)
	idx := int(slot) % len(w.buckets)
	if w.buckets[idx].ts != slot {
		// First write of this slot: reset the bucket.
		w.buckets[idx].ts = slot
		w.buckets[idx].success.Store(0)
		w.buckets[idx].failure.Store(0)
	}
	if success {
		w.buckets[idx].success.Add(1)
	} else {
		w.buckets[idx].failure.Add(1)
	}
}

// snapshot sums the counters of all slots inside the window.
func (w *slidingWindow) snapshot(now time.Time) (success, failure uint64) {
	current := w.slot(now)
	for i := range w.buckets {
		slot := w.buckets[i].ts
		if slot == 0 {
			continue
		}
		// Keep only slots within the ring (i.e. younger than one full cycle).
		if current-slot < int64(len(w.buckets)) {
			success += w.buckets[i].success.Load()
			failure += w.buckets[i].failure.Load()
		}
	}
	return success, failure
}

// reset clears all buckets.
func (w *slidingWindow) reset() {
	for i := range w.buckets {
		w.buckets[i].ts = 0
		w.buckets[i].success.Store(0)
		w.buckets[i].failure.Store(0)
	}
}
