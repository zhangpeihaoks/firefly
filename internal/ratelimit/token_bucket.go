package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// tokenBucketScript implements a distributed token bucket atomically in Redis.
//
// Bucket state is stored in a hash: {tokens, last}. Tokens refill at `rate`
// per second up to `burst` capacity. Time is expressed as UNIX seconds
// (float) so refill stays accurate across long idle periods.
//
// Returns 1 when the request consumed a token, 0 when denied.
var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local tokens = tonumber(redis.call('HGET', key, 'tokens'))
local last = tonumber(redis.call('HGET', key, 'last') or '0')

if tokens == nil then
	tokens = burst
	last = now
end

-- Refill tokens based on elapsed time.
tokens = tokens + (now - last) * rate
if tokens > burst then
	tokens = burst
end

if tokens >= requested then
	tokens = tokens - requested
	redis.call('HSET', key, 'tokens', tokens, 'last', now)
	-- Expire the key shortly after the bucket would drain, so unused
	-- buckets do not accumulate in memory.
	redis.call('EXPIRE', key, math.ceil(burst / rate) + 1)
	return 1
end

-- Track last refill even on denial to avoid a burst after a long idle.
redis.call('HSET', key, 'last', now)
return 0
`)

// RedisTokenBucketLimiter is a distributed token bucket backed by Redis.
// It allows burst traffic up to burst capacity while keeping the average
// rate at rate tokens/second, shared across all service instances.
type RedisTokenBucketLimiter struct {
	cmd     redis.Cmdable
	rate    float64 // tokens per second
	burst   float64 // bucket capacity
	now     func() time.Time
	failOpen bool
}

// RedisTokenBucketConfig configures a RedisTokenBucketLimiter.
type RedisTokenBucketConfig struct {
	// Rate is the number of tokens added per second.
	Rate float64
	// Burst is the maximum bucket capacity (allowable burst size).
	Burst int
	// FailOpen allows the request when Redis is unreachable (default true).
	FailOpen bool
}

// NewRedisTokenBucketLimiter creates a distributed token bucket limiter.
//
//	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"}) // or firefly's database/redis Client()
//	limiter := ratelimit.NewRedisTokenBucketLimiter(rdb, ratelimit.RedisTokenBucketConfig{
//	    Rate: 100, Burst: 200, // 100 req/s, burst 200
//	})
func NewRedisTokenBucketLimiter(cmd redis.Cmdable, cfg RedisTokenBucketConfig) *RedisTokenBucketLimiter {
	return &RedisTokenBucketLimiter{
		cmd:      cmd,
		rate:     cfg.Rate,
		burst:    float64(cfg.Burst),
		now:      time.Now,
		failOpen: cfg.FailOpen,
	}
}

// Allow checks whether a single request for the key is allowed.
func (l *RedisTokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

// AllowN checks whether n requests for the key are allowed.
func (l *RedisTokenBucketLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	if n <= 0 {
		return true, nil
	}
	now := float64(l.now().UnixNano()) / 1e9

	allowed, err := evalScript(ctx, l.cmd, tokenBucketScript,
		[]string{key},
		strconv.FormatFloat(l.rate, 'f', -1, 64),
		strconv.FormatFloat(l.burst, 'f', -1, 64),
		strconv.FormatFloat(now, 'f', -1, 64),
		strconv.Itoa(n),
	)
	if err != nil {
		if l.failOpen {
			return true, nil
		}
		return false, fmt.Errorf("%w: %v", ErrRedisUnavailable, err)
	}
	return allowed == 1, nil
}
