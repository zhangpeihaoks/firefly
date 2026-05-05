// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// RateLimiter is the interface for rate limiting strategies.
// Implementations can provide different algorithms:
// - Token Bucket
// - Leaky Bucket
// - Sliding Window
// - Fixed Window
type RateLimiter interface {
	// Allow checks if a request is allowed under the rate limit.
	// Returns true if the request should be allowed, false otherwise.
	Allow() bool
	// AllowN checks if n requests are allowed under the rate limit.
	AllowN(n int) bool
	// Wait blocks until a request is allowed or the context is cancelled.
	// Returns an error if the context is cancelled before a request is allowed.
	Wait(ctx context.Context) error
	// WaitN blocks until n requests are allowed or the context is cancelled.
	WaitN(ctx context.Context, n int) error
	// Reserve reserves a request and returns the time to wait before using it.
	Reserve() Reservation
	// ReserveN reserves n requests and returns the time to wait before using them.
	ReserveN(n int) Reservation
}

// Reservation represents a reservation for rate-limited requests.
type Reservation struct {
	// ok indicates whether the reservation was successful.
	ok bool
	// timeToWait is the duration to wait before the reservation can be used.
	timeToWait time.Duration
}

// OK returns whether the reservation was successful.
func (r Reservation) OK() bool {
	return r.ok
}

// Delay returns the duration to wait before the reservation can be used.
func (r Reservation) Delay() time.Duration {
	return r.timeToWait
}

// TokenBucketLimiter implements the Token Bucket rate limiting algorithm.
// The token bucket algorithm:
// 1. Tokens are added to the bucket at a fixed rate (Rate)
// 2. The bucket has a maximum capacity (Burst)
// 3. Each request consumes one token
// 4. If the bucket is empty, the request is denied
//
// This allows for burst traffic up to the bucket capacity while
// maintaining the average rate limit.
type TokenBucketLimiter struct {
	rate     float64          // tokens per second
	burst    int              // maximum bucket capacity
	tokens   float64          // current token count
	lastTime time.Time        // last time tokens were updated
	mu       sync.Mutex       // protects tokens and lastTime
	now      func() time.Time // for testing
}

// NewTokenBucketLimiter creates a new TokenBucketLimiter.
// rate: tokens added per second
// burst: maximum bucket capacity
func NewTokenBucketLimiter(rate float64, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:     rate,
		burst:    burst,
		tokens:   float64(burst), // start with a full bucket
		lastTime: time.Now(),
		now:      time.Now,
	}
}

// Allow checks if a single request is allowed.
func (l *TokenBucketLimiter) Allow() bool {
	return l.AllowN(1)
}

// AllowN checks if n requests are allowed.
func (l *TokenBucketLimiter) AllowN(n int) bool {
	return l.reserveN(n, 0).ok
}

// Wait blocks until a request is allowed or the context is cancelled.
func (l *TokenBucketLimiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// WaitN blocks until n requests are allowed or the context is cancelled.
func (l *TokenBucketLimiter) WaitN(ctx context.Context, n int) error {
	// Use a very large maxWait for Wait operations (effectively infinite waiting)
	r := l.reserveN(n, time.Duration(1<<62-1))
	if !r.ok {
		return errors.New(errors.CodeServiceUnavailable, "RATE_LIMIT_EXCEEDED", "请求频率超限")
	}

	delay := r.timeToWait
	if delay <= 0 {
		return nil
	}

	t := time.NewTimer(delay)
	defer t.Stop()

	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Reserve reserves a single request.
func (l *TokenBucketLimiter) Reserve() Reservation {
	return l.ReserveN(1)
}

// ReserveN reserves n requests.
func (l *TokenBucketLimiter) ReserveN(n int) Reservation {
	return l.reserveN(n, l.now().Sub(l.lastTime))
}

// reserveN is the internal reservation logic.
func (l *TokenBucketLimiter) reserveN(n int, maxWait time.Duration) Reservation {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	// Calculate elapsed time and add tokens
	elapsed := now.Sub(l.lastTime)
	newTokens := float64(elapsed.Nanoseconds()) * l.rate / 1e9
	l.tokens += newTokens

	// Cap at burst
	if l.tokens > float64(l.burst) {
		l.tokens = float64(l.burst)
	}

	l.lastTime = now

	var r Reservation
	if n <= 0 {
		r = Reservation{ok: true, timeToWait: 0}
		return r
	}

	tokensNeeded := float64(n)

	if l.tokens >= tokensNeeded {
		l.tokens -= tokensNeeded
		r = Reservation{ok: true, timeToWait: 0}
		return r
	}

	// Calculate wait time
	tokensShort := tokensNeeded - l.tokens
	waitDuration := time.Duration(float64(time.Second) * tokensShort / l.rate)

	if waitDuration > maxWait {
		r = Reservation{ok: false, timeToWait: 0}
		return r
	}

	l.tokens = 0
	l.lastTime = now.Add(waitDuration)
	r = Reservation{ok: true, timeToWait: waitDuration}
	return r
}

// LeakyBucketLimiter implements the Leaky Bucket rate limiting algorithm.
// The leaky bucket algorithm:
// 1. Requests are added to a queue (bucket)
// 2. The bucket "leaks" at a constant rate (processed requests)
// 3. If the bucket is full, the request is denied
//
// This provides smooth output rate regardless of input bursts.
type LeakyBucketLimiter struct {
	rate     float64          // requests per second to process
	capacity int              // maximum bucket capacity
	water    int              // current water level
	lastLeak time.Time        // last time water was leaked
	mu       sync.Mutex       // protects water and lastLeak
	now      func() time.Time // for testing
}

// NewLeakyBucketLimiter creates a new LeakyBucketLimiter.
// rate: requests processed per second
// capacity: maximum bucket capacity
func NewLeakyBucketLimiter(rate float64, capacity int) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		rate:     rate,
		capacity: capacity,
		water:    0,
		lastLeak: time.Now(),
		now:      time.Now,
	}
}

// Allow checks if a single request is allowed.
func (l *LeakyBucketLimiter) Allow() bool {
	return l.AllowN(1)
}

// AllowN checks if n requests are allowed.
func (l *LeakyBucketLimiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	// Calculate leaked water
	elapsed := now.Sub(l.lastLeak)
	leaked := int(float64(elapsed.Nanoseconds()) * l.rate / 1e9)
	l.water -= leaked
	if l.water < 0 {
		l.water = 0
	}

	l.lastLeak = now

	// Check if we can add the water
	if l.water+n <= l.capacity {
		l.water += n
		return true
	}

	return false
}

// Wait blocks until a request is allowed or the context is cancelled.
func (l *LeakyBucketLimiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// WaitN blocks until n requests are allowed or the context is cancelled.
func (l *LeakyBucketLimiter) WaitN(ctx context.Context, n int) error {
	for {
		if l.AllowN(n) {
			return nil
		}

		// Calculate wait time
		waitTime := time.Duration(float64(time.Second) / l.rate)

		select {
		case <-time.After(waitTime):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Reserve reserves a single request.
func (l *LeakyBucketLimiter) Reserve() Reservation {
	return l.ReserveN(1)
}

// ReserveN reserves n requests.
func (l *LeakyBucketLimiter) ReserveN(n int) Reservation {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	// Calculate leaked water
	elapsed := now.Sub(l.lastLeak)
	leaked := int(float64(elapsed.Nanoseconds()) * l.rate / 1e9)
	l.water -= leaked
	if l.water < 0 {
		l.water = 0
	}

	l.lastLeak = now

	// Check if we can add the water
	if l.water+n <= l.capacity {
		l.water += n
		return Reservation{ok: true, timeToWait: 0}
	}

	// Calculate wait time
	spaceNeeded := (l.water + n) - l.capacity
	waitDuration := time.Duration(float64(time.Second) * float64(spaceNeeded) / l.rate)

	return Reservation{ok: l.water+n <= l.capacity, timeToWait: waitDuration}
}

// KeyExtractor is a function that extracts a rate limit key from a request.
// This allows for per-user, per-IP, or other custom rate limiting strategies.
type KeyExtractor func(ctx context.Context, tr transport.Transporter) string

// ratelimitOptions holds the configuration for RateLimit middleware.
type ratelimitOptions struct {
	logger         *slog.Logger
	limiter        RateLimiter
	limiterFactory func(key string) RateLimiter
	keyExtractor   KeyExtractor
	skipPaths      map[string]bool
	retryAfter     bool
}

// RateLimitOption is a configuration option for RateLimit middleware.
type RateLimitOption func(*ratelimitOptions)

// WithRateLimitLogger sets a custom logger for the RateLimit middleware.
// If not set, the default slog logger is used.
func WithRateLimitLogger(logger *slog.Logger) RateLimitOption {
	return func(o *ratelimitOptions) {
		o.logger = logger
	}
}

// WithRateLimiter sets a global rate limiter.
// This limiter is shared across all requests.
func WithRateLimiter(limiter RateLimiter) RateLimitOption {
	return func(o *ratelimitOptions) {
		o.limiter = limiter
	}
}

// WithLimiterFactory sets a factory function for creating per-key rate limiters.
// This enables per-user or per-IP rate limiting.
// The factory is called with the key extracted from the request.
func WithLimiterFactory(factory func(key string) RateLimiter) RateLimitOption {
	return func(o *ratelimitOptions) {
		o.limiterFactory = factory
	}
}

// WithKeyExtractor sets the key extractor function.
// The extracted key is used for per-key rate limiting.
// Default: extracts client IP from transport.
func WithKeyExtractor(extractor KeyExtractor) RateLimitOption {
	return func(o *ratelimitOptions) {
		o.keyExtractor = extractor
	}
}

// WithRateLimiterSkipPaths sets paths that should skip rate limiting.
// These paths will not be rate limited.
func WithRateLimiterSkipPaths(paths []string) RateLimitOption {
	return func(o *ratelimitOptions) {
		o.skipPaths = make(map[string]bool)
		for _, path := range paths {
			o.skipPaths[path] = true
		}
	}
}

// WithRetryAfter enables or disables the Retry-After header in rate limit responses.
// When enabled, clients receive information about when to retry.
func WithRetryAfter(enabled bool) RateLimitOption {
	return func(o *ratelimitOptions) {
		o.retryAfter = enabled
	}
}

// RateLimit returns a middleware that enforces rate limiting.
// It supports both global and per-key rate limiting using various algorithms.
//
// The middleware:
//  1. Extracts the rate limit key (e.g., client IP)
//  2. Checks if the request is allowed under the rate limit
//  3. Returns 429 Too Many Requests if rate limit is exceeded
//
// Example (global rate limiting):
//
//	limiter := middleware.NewTokenBucketLimiter(100, 200) // 100 req/sec, burst 200
//	middleware.RateLimit(middleware.WithRateLimiter(limiter))
//
// Example (per-IP rate limiting):
//
//	middleware.RateLimit(
//	    middleware.WithLimiterFactory(func(key string) middleware.RateLimiter {
//	        return middleware.NewTokenBucketLimiter(10, 20) // 10 req/sec per IP
//	    }),
//	)
func RateLimit(opts ...RateLimitOption) Middleware {
	// Apply default options
	options := &ratelimitOptions{
		logger:       slog.Default(),
		keyExtractor: defaultKeyExtractor,
	}

	// Apply custom options
	for _, opt := range opts {
		opt(options)
	}

	// Create limiter store for per-key limiting
	var limiters sync.Map // map[string]RateLimiter

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Get transport info from context
			tr := transport.FromContext(ctx)

			// Check if path should skip rate limiting
			if tr != nil && options.skipPaths != nil {
				if options.skipPaths[tr.Operation()] {
					return next(ctx, req)
				}
			}

			var limiter RateLimiter
			var key string

			// Use global limiter or per-key limiter
			if options.limiter != nil {
				limiter = options.limiter
			} else if options.limiterFactory != nil {
				// Extract key for per-key limiting
				key = options.keyExtractor(ctx, tr)

				// Get or create limiter for this key
				if v, ok := limiters.Load(key); ok {
					limiter = v.(RateLimiter)
				} else {
					limiter = options.limiterFactory(key)
					limiters.Store(key, limiter)
				}
			} else {
				// No limiter configured
				options.logger.Warn("rate limit middleware: no limiter configured")
				return next(ctx, req)
			}

			// Check if request is allowed
			if !limiter.Allow() {
				// Log rate limit exceeded
				options.logger.Warn("rate limit exceeded",
					"key", key,
					"operation", func() string {
						if tr != nil {
							return tr.Operation()
						}
						return ""
					}(),
				)

				// Return rate limit error
				err := errors.New(errors.CodeServiceUnavailable, "RATE_LIMIT_EXCEEDED", "请求频率超限，请稍后重试")
				if options.retryAfter {
					err = err.WithMetadata(map[string]string{
						"retry_after": "1", // Suggest retry after 1 second
					})
				}
				return nil, err
			}

			// Call the next handler
			return next(ctx, req)
		}
	}
}

// defaultKeyExtractor extracts the client IP as the rate limit key.
func defaultKeyExtractor(ctx context.Context, tr transport.Transporter) string {
	if tr == nil {
		return "unknown"
	}

	// Try to get client IP from request header
	if tr.RequestHeader() != nil {
		// Check common headers for real IP
		if ip := tr.RequestHeader().Get("X-Real-IP"); ip != "" {
			return ip
		}
		if ip := tr.RequestHeader().Get("X-Forwarded-For"); ip != "" {
			// X-Forwarded-For may contain multiple IPs, use the first one
			return ip
		}
	}

	// Fall back to operation name (path) as key
	return tr.Operation()
}
