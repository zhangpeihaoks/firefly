package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// slidingWindowScript implements the sliding window log algorithm (the Redis
// official pattern, see https://redis.io/docs/latest/develop/use/patterns/rate-limiting/)
// atomically in Lua.
//
// Each request is a member of a sorted set scored by its arrival time (ms).
// Members outside the window are pruned, then the set is counted: if under
// `limit`, the request is recorded and allowed; otherwise denied.
var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local windowMs = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, 0, now - windowMs)
local count = redis.call('ZCARD', key)

if count < limit then
	redis.call('ZADD', key, now, member)
	redis.call('EXPIRE', key, math.ceil(windowMs / 1000) + 1)
	return 1
end
return 0
`)

// RedisSlidingWindowLimiter implements the sliding window log algorithm.
// Unlike the fixed window, the boundary has no burst hole: at most `Limit`
// requests per `Window` per key at any instant, shared across instances.
//
// Trade-off: each request stores one ZSET member, so it uses more memory than
// the fixed window and is best suited for moderate limits.
type RedisSlidingWindowLimiter struct {
	cmd        redis.Cmdable
	limit      int64
	window     time.Duration
	now        func() time.Time
	serial     atomic.Uint64
	failOpen   bool
}

// RedisSlidingWindowConfig configures a RedisSlidingWindowLimiter.
type RedisSlidingWindowConfig struct {
	// Limit is the maximum number of requests allowed within the window.
	Limit int
	// Window is the sliding window length (e.g. time.Minute).
	Window time.Duration
	// FailOpen allows the request when Redis is unreachable (default true).
	FailOpen bool
}

// NewRedisSlidingWindowLimiter creates a sliding window log limiter.
//
//	limiter := ratelimit.NewRedisSlidingWindowLimiter(rdb, ratelimit.RedisSlidingWindowConfig{
//	    Limit: 1000, Window: time.Minute, // 1000 req/min, no boundary burst
//	})
func NewRedisSlidingWindowLimiter(cmd redis.Cmdable, cfg RedisSlidingWindowConfig) *RedisSlidingWindowLimiter {
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	return &RedisSlidingWindowLimiter{
		cmd:      cmd,
		limit:    int64(cfg.Limit),
		window:   cfg.Window,
		now:      time.Now,
		failOpen: cfg.FailOpen,
	}
}

// Allow checks whether a single request for the key is allowed.
func (l *RedisSlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

// AllowN checks whether n requests for the key are allowed.
func (l *RedisSlidingWindowLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	now := l.now()
	nowMs := now.UnixMilli()

	// Reserve a unique member per request slot. The serial makes members
	// unique even when multiple requests arrive within the same millisecond.
	serial := l.serial.Add(uint64(n))
	member := fmt.Sprintf("%d-%d", nowMs, serial)

	allowed, err := evalScript(ctx, l.cmd, slidingWindowScript,
		[]string{key},
		strconv.FormatInt(nowMs, 10),
		strconv.FormatInt(l.window.Milliseconds(), 10),
		strconv.FormatInt(l.limit, 10),
		member,
	)
	if err != nil {
		if l.failOpen {
			return true, nil
		}
		return false, fmt.Errorf("%w: %v", ErrRedisUnavailable, err)
	}
	return allowed == 1, nil
}
