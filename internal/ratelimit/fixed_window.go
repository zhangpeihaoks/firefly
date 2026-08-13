package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// fixedWindowScript is the official Redis rate limiting pattern (see
// https://redis.io/commands/incr/ — "Rate limiting pattern") made atomic.
//
// The window key (key + ":" + window start) is INCRemented; the first request
// in a window sets its EXPIRE. Requests beyond `limit` within the window are
// denied. The window key embeds the window start timestamp, so a new window
// starts with a fresh counter.
var fixedWindowScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local count = redis.call('INCR', key)
if count == 1 then
	redis.call('EXPIRE', key, window)
end

if count > limit then
	return 0
end
return 1
`)

// RedisFixedWindowLimiter implements the official Redis fixed-window rate
// limiting pattern: at most `Limit` requests per `Window` per key, shared
// across all service instances.
//
// Note: fixed windows allow up to 2x the limit in a window boundary burst
// (e.g. a request just before the boundary and one just after). Use
// RedisSlidingWindowLimiter for stricter smoothing.
type RedisFixedWindowLimiter struct {
	cmd       redis.Cmdable
	limit     int64
	window    time.Duration
	now       func() time.Time
	failOpen  bool
}

// RedisFixedWindowConfig configures a RedisFixedWindowLimiter.
type RedisFixedWindowConfig struct {
	// Limit is the maximum number of requests allowed per window.
	Limit int
	// Window is the length of the counting window (e.g. time.Second).
	Window time.Duration
	// FailOpen allows the request when Redis is unreachable (default true).
	FailOpen bool
}

// NewRedisFixedWindowLimiter creates a fixed-window limiter using the official
// Redis INCR + EXPIRE pattern.
//
//	limiter := ratelimit.NewRedisFixedWindowLimiter(rdb, ratelimit.RedisFixedWindowConfig{
//	    Limit: 100, Window: time.Second, // 100 req/s per key
//	})
func NewRedisFixedWindowLimiter(cmd redis.Cmdable, cfg RedisFixedWindowConfig) *RedisFixedWindowLimiter {
	if cfg.Window <= 0 {
		cfg.Window = time.Second
	}
	return &RedisFixedWindowLimiter{
		cmd:      cmd,
		limit:    int64(cfg.Limit),
		window:   cfg.Window,
		now:      time.Now,
		failOpen: cfg.FailOpen,
	}
}

// Allow checks whether a single request for the key is allowed.
func (l *RedisFixedWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

// AllowN checks whether n requests for the key are allowed.
func (l *RedisFixedWindowLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	// Window key embeds the window start, so each window has its own counter.
	windowStart := l.now().Unix() / int64(l.window/time.Second)
	windowKey := fmt.Sprintf("%s:%d", key, windowStart)

	allowed, err := evalScript(ctx, l.cmd, fixedWindowScript,
		[]string{windowKey},
		strconv.FormatInt(l.limit, 10),
		strconv.FormatInt(int64(l.window/time.Second), 10),
	)
	if err != nil {
		if l.failOpen {
			return true, nil
		}
		return false, fmt.Errorf("%w: %v", ErrRedisUnavailable, err)
	}
	return allowed == 1, nil
}
