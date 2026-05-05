// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	stderrors "errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// TestTokenBucketLimiter_Basic tests basic token bucket functionality.
func TestTokenBucketLimiter_Basic(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 5) // 10 req/sec, burst 5

	// Should allow burst
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("expected request %d to be allowed", i)
		}
	}

	// Should be denied after burst
	if limiter.Allow() {
		t.Error("expected request after burst to be denied")
	}
}

// TestTokenBucketLimiter_Wait tests that Wait blocks until tokens are available.
func TestTokenBucketLimiter_Wait(t *testing.T) {
	// Use high rate for fast test
	limiter := NewTokenBucketLimiter(100, 1) // 100 req/sec, burst 1

	ctx := context.Background()

	// First request should succeed immediately
	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Second request should block and succeed after token refill
	start := time.Now()
	err = limiter.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have waited at least ~10ms (1/100 sec)
	if elapsed < 5*time.Millisecond {
		t.Errorf("expected wait of at least 5ms, got %v", elapsed)
	}
}

// TestTokenBucketLimiter_WaitContextCancel tests that Wait respects context cancellation.
func TestTokenBucketLimiter_WaitContextCancel(t *testing.T) {
	limiter := NewTokenBucketLimiter(1, 1) // 1 req/sec, burst 1

	// Consume the initial token
	limiter.Allow()

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
	if !stderrors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestTokenBucketLimiter_AllowN tests AllowN with multiple tokens.
func TestTokenBucketLimiter_AllowN(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 10) // 10 req/sec, burst 10

	// Should allow 10 requests at once
	if !limiter.AllowN(10) {
		t.Error("expected AllowN(10) to be allowed")
	}

	// Should be denied after
	if limiter.Allow() {
		t.Error("expected request after burst to be denied")
	}
}

// TestTokenBucketLimiter_Reserve tests reservation functionality.
func TestTokenBucketLimiter_Reserve(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 5)

	// Exhaust tokens
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	// Reserve should fail (no tokens available)
	r := limiter.Reserve()
	if r.OK() {
		t.Error("expected reservation to fail when no tokens available")
	}
}

// TestLeakyBucketLimiter_Basic tests basic leaky bucket functionality.
func TestLeakyBucketLimiter_Basic(t *testing.T) {
	limiter := NewLeakyBucketLimiter(10, 5) // 10 req/sec processed, capacity 5

	// Should allow up to capacity
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("expected request %d to be allowed", i)
		}
	}

	// Should be denied after reaching capacity
	if limiter.Allow() {
		t.Error("expected request after capacity to be denied")
	}
}

// TestLeakyBucketLimiter_Refill tests that the leaky bucket refills over time.
func TestLeakyBucketLimiter_Refill(t *testing.T) {
	// Use mocked time for precise control
	now := time.Now()
	limiter := &LeakyBucketLimiter{
		rate:     100, // 100 req/sec
		capacity: 2,
		water:    0,
		lastLeak: now,
		now:      func() time.Time { return now },
	}

	// Fill the bucket
	if !limiter.AllowN(2) {
		t.Error("expected AllowN(2) to be allowed")
	}

	// Should be denied now (bucket full)
	if limiter.Allow() {
		t.Error("expected request to be denied when bucket is full")
	}

	// Advance time by 10ms (should allow 1 request to leak)
	now = now.Add(10 * time.Millisecond)
	limiter.now = func() time.Time { return now }

	// Should allow 1 request now
	if !limiter.Allow() {
		t.Error("expected request to be allowed after leak")
	}
}

// TestLeakyBucketLimiter_Wait tests that Wait blocks until requests are allowed.
func TestLeakyBucketLimiter_Wait(t *testing.T) {
	limiter := NewLeakyBucketLimiter(100, 1) // 100 req/sec, capacity 1

	ctx := context.Background()

	// First request succeeds
	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Second request waits and succeeds
	start := time.Now()
	err = limiter.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have waited
	if elapsed < 5*time.Millisecond {
		t.Errorf("expected wait, got %v", elapsed)
	}
}

// TestRateLimitMiddleware_Global tests global rate limiting.
func TestRateLimitMiddleware_Global(t *testing.T) {
	// Create a limiter with burst 2
	limiter := NewTokenBucketLimiter(1, 2)

	// Create middleware with global limiter
	m := RateLimit(WithRateLimiter(limiter))

	// Create a handler that counts requests
	var count atomic.Int32
	handler := m(func(ctx context.Context, req any) (any, error) {
		count.Add(1)
		return "ok", nil
	})

	// Create context with transport
	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/test",
		requestHeader: newMockHeader(),
		replyHeader:   newMockHeader(),
	})

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		_, err := handler(ctx, nil)
		if err != nil {
			t.Errorf("unexpected error on request %d: %v", i, err)
		}
	}

	// Third request should be rate limited
	_, err := handler(ctx, nil)
	if err == nil {
		t.Error("expected rate limit error")
	}

	// Check error type
	if fwErr, ok := stderrors.AsType[*errors.Error](err); ok {
		if fwErr.Code != errors.CodeServiceUnavailable {
			t.Errorf("expected code %d, got %d", errors.CodeServiceUnavailable, fwErr.Code)
		}
		if fwErr.Reason != "RATE_LIMIT_EXCEEDED" {
			t.Errorf("expected reason RATE_LIMIT_EXCEEDED, got %s", fwErr.Reason)
		}
	} else {
		t.Errorf("expected *errors.Error, got %T", err)
	}

	// Verify only 2 requests were processed
	if count.Load() != 2 {
		t.Errorf("expected 2 requests processed, got %d", count.Load())
	}
}

// TestRateLimitMiddleware_PerKey tests per-key rate limiting.
func TestRateLimitMiddleware_PerKey(t *testing.T) {
	// Track limiter creation per key
	createdKeys := sync.Map{}

	// Create middleware with per-key limiter factory
	m := RateLimit(
		WithLimiterFactory(func(key string) RateLimiter {
			createdKeys.Store(key, true)
			return NewTokenBucketLimiter(1, 1) // 1 req/sec per key
		}),
	)

	// Create a handler that counts requests
	var count atomic.Int32
	handler := m(func(ctx context.Context, req any) (any, error) {
		count.Add(1)
		return "ok", nil
	})

	// Request from IP1
	ctx1 := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/test",
		requestHeader: &mockHeader{values: map[string]string{"X-Real-IP": "192.168.1.1"}},
		replyHeader:   newMockHeader(),
	})

	// Request from IP2
	ctx2 := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/test",
		requestHeader: &mockHeader{values: map[string]string{"X-Real-IP": "192.168.1.2"}},
		replyHeader:   newMockHeader(),
	})

	// First request from IP1 should succeed
	_, err := handler(ctx1, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Second request from IP1 should be rate limited
	_, err = handler(ctx1, nil)
	if err == nil {
		t.Error("expected rate limit error for second request from IP1")
	}

	// First request from IP2 should succeed (different key)
	_, err = handler(ctx2, nil)
	if err != nil {
		t.Errorf("unexpected error for request from IP2: %v", err)
	}

	// Verify limiters were created for both IPs
	if _, ok := createdKeys.Load("192.168.1.1"); !ok {
		t.Error("expected limiter to be created for 192.168.1.1")
	}
	if _, ok := createdKeys.Load("192.168.1.2"); !ok {
		t.Error("expected limiter to be created for 192.168.1.2")
	}

	// Verify 2 requests were processed
	if count.Load() != 2 {
		t.Errorf("expected 2 requests processed, got %d", count.Load())
	}
}

// TestRateLimitMiddleware_SkipPaths tests that skipped paths bypass rate limiting.
func TestRateLimitMiddleware_SkipPaths(t *testing.T) {
	// Create a limiter with burst 0
	limiter := NewTokenBucketLimiter(1, 0) // burst 0 means no requests allowed

	// Create middleware with skip paths
	m := RateLimit(
		WithRateLimiter(limiter),
		WithRateLimiterSkipPaths([]string{"/health", "/metrics"}),
	)

	// Create handler
	var count atomic.Int32
	handler := m(func(ctx context.Context, req any) (any, error) {
		count.Add(1)
		return "ok", nil
	})

	// Request to /health should bypass rate limit
	healthCtx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/health",
		requestHeader: newMockHeader(),
		replyHeader:   newMockHeader(),
	})
	_, err := handler(healthCtx, nil)
	if err != nil {
		t.Errorf("expected /health to bypass rate limit, got error: %v", err)
	}

	// Request to /metrics should bypass rate limit
	metricsCtx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/metrics",
		requestHeader: newMockHeader(),
		replyHeader:   newMockHeader(),
	})
	_, err = handler(metricsCtx, nil)
	if err != nil {
		t.Errorf("expected /metrics to bypass rate limit, got error: %v", err)
	}

	// Request to /api/test should be rate limited
	apiCtx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/test",
		requestHeader: newMockHeader(),
		replyHeader:   newMockHeader(),
	})
	_, err = handler(apiCtx, nil)
	if err == nil {
		t.Error("expected /api/test to be rate limited")
	}

	// Verify 2 requests were processed (skip paths)
	if count.Load() != 2 {
		t.Errorf("expected 2 requests processed, got %d", count.Load())
	}
}

// TestRateLimitMiddleware_RetryAfter tests the Retry-After metadata.
func TestRateLimitMiddleware_RetryAfter(t *testing.T) {
	limiter := NewTokenBucketLimiter(1, 0) // burst 0

	m := RateLimit(
		WithRateLimiter(limiter),
		WithRetryAfter(true),
	)

	handler := m(func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/test",
		requestHeader: newMockHeader(),
		replyHeader:   newMockHeader(),
	})

	_, err := handler(ctx, nil)
	if err == nil {
		t.Fatal("expected rate limit error")
	}

	if fwErr, ok := stderrors.AsType[*errors.Error](err); ok {
		if fwErr.Metadata == nil {
			t.Error("expected metadata to be set")
		} else if fwErr.Metadata["retry_after"] != "1" {
			t.Errorf("expected retry_after=1, got %s", fwErr.Metadata["retry_after"])
		}
	} else {
		t.Errorf("expected *errors.Error, got %T", err)
	}
}

// TestRateLimitMiddleware_NoLimiter tests that middleware passes through when no limiter configured.
func TestRateLimitMiddleware_NoLimiter(t *testing.T) {
	// Capture warnings
	var buf []byte
	logger := slog.New(slog.NewTextHandler(&testWriter{&buf}, nil))

	m := RateLimit(WithRateLimitLogger(logger))

	handler := m(func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/test",
		requestHeader: newMockHeader(),
		replyHeader:   newMockHeader(),
	})

	// Should pass through without rate limiting
	_, err := handler(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have logged a warning
	if len(buf) == 0 {
		t.Error("expected warning log about no limiter configured")
	}
}

// TestRateLimitMiddleware_CustomKeyExtractor tests custom key extraction.
func TestRateLimitMiddleware_CustomKeyExtractor(t *testing.T) {
	limiter := NewTokenBucketLimiter(1, 1)

	// Custom key extractor that uses a custom header
	m := RateLimit(
		WithRateLimiter(limiter),
		WithKeyExtractor(func(ctx context.Context, tr transport.Transporter) string {
			if tr.RequestHeader() != nil {
				return tr.RequestHeader().Get("X-User-ID")
			}
			return "unknown"
		}),
	)

	handler := m(func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	// Request with X-User-ID header
	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/test",
		requestHeader: &mockHeader{values: map[string]string{"X-User-ID": "user123"}},
		replyHeader:   newMockHeader(),
	})

	// First request should succeed
	_, err := handler(ctx, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Second request should be rate limited
	_, err = handler(ctx, nil)
	if err == nil {
		t.Error("expected rate limit error")
	}
}

// TestRateLimitMiddleware_NilTransport tests behavior with nil transport.
func TestRateLimitMiddleware_NilTransport(t *testing.T) {
	limiter := NewTokenBucketLimiter(1, 2)

	m := RateLimit(WithRateLimiter(limiter))

	handler := m(func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	// Context without transport
	ctx := context.Background()

	// Should still work with nil transport (use percentage limiter or nil check)
	_, err := handler(ctx, nil)
	if err != nil {
		t.Errorf("expected request to succeed even with nil transport: %v", err)
	}
}

// TestRateLimitMiddleware_Concurrent tests concurrent rate limiting.
func TestRateLimitMiddleware_Concurrent(t *testing.T) {
	limiter := NewTokenBucketLimiter(1000, 100) // 1000 req/sec, burst 100

	m := RateLimit(WithRateLimiter(limiter))

	var successCount atomic.Int32
	var rateLimitedCount atomic.Int32

	handler := m(func(ctx context.Context, req any) (any, error) {
		successCount.Add(1)
		return "ok", nil
	})

	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "/api/test",
		requestHeader: newMockHeader(),
		replyHeader:   newMockHeader(),
	})

	// Launch 200 concurrent requests (burst is 100)
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := handler(ctx, nil)
			if err != nil {
				rateLimitedCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// At least 100 should have succeeded (burst)
	if successCount.Load() < 100 {
		t.Errorf("expected at least 100 successful requests, got %d", successCount.Load())
	}

	// Some should have been rate limited (since we sent 200 with burst 100)
	if rateLimitedCount.Load() == 0 {
		t.Error("expected some requests to be rate limited")
	}
}

// testWriter is a simple io.Writer for testing logs.
type testWriter struct {
	buf *[]byte
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
