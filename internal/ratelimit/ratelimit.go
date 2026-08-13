// Package ratelimit provides distributed rate limiters backed by Redis.
//
// Unlike the in-memory limiters in internal/middleware, these limiters are
// stateless on the application side — the counter/token state lives in Redis,
// so multiple instances of a service share the same limit.
//
// All limiters implement RedisLimiter, which takes a key (e.g. client IP,
// user ID, or route) so per-key limits are applied globally across instances.
package ratelimit

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// ErrRedisUnavailable is returned when the Redis backend cannot be reached
// while the limiter is configured to fail closed.
var ErrRedisUnavailable = errors.New("ratelimit: redis unavailable")

// RedisLimiter is the distributed rate limiter interface.
// Implementations must be safe for concurrent use.
type RedisLimiter interface {
	// Allow checks whether a request for the given key is allowed.
	Allow(ctx context.Context, key string) (bool, error)
	// AllowN checks whether n requests for the given key are allowed.
	AllowN(ctx context.Context, key string, n int) (bool, error)
}

// Options configures common behavior shared by all Redis limiters.
type Options struct {
	// FailOpen controls behavior when Redis is unreachable:
	// true  (default): allow the request (availability over strictness)
	// false:           deny with ErrRedisUnavailable
	FailOpen bool
}

// evalScript runs a Lua script with keys and args and decodes the integer result.
func evalScript(ctx context.Context, cmd redis.Cmdable, script *redis.Script, keys []string, args ...any) (int64, error) {
	res, err := script.Run(ctx, cmd, keys, args...).Result()
	if err != nil {
		return 0, err
	}
	n, ok := res.(int64)
	if !ok {
		return 0, errors.New("ratelimit: unexpected script result")
	}
	return n, nil
}
