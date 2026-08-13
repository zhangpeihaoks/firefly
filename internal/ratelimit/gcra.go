package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// gcraScript implements the Generic Cell Rate Algorithm (GCRA), a leaky-bucket
// variant used by Redis official rate limiting docs and redis-cell.
//
// State is a single STRING key holding the TAT (Theoretical Arrival Time),
// relative to Jan 1, 2017 to stay within 64-bit float precision. Time comes
// from the Redis server (TIME command), so client clock skew does not affect
// distributed limiting. The key TTL is derived from reset_after, so unused
// buckets expire automatically.
//
// Returns [allowed, remaining, retry_after, reset_after]:
//   - allowed:     cost consumed (0 when denied)
//   - remaining:   tokens remaining (float, may be fractional)
//   - retry_after: seconds until next allowed request; -1 when allowed
//   - reset_after: seconds until the bucket returns to full capacity
//
// Based on github.com/go-redis/redis_rate (BSD-2-Clause), which itself
// derives from github.com/rwz/redis-gcra.
var gcraScript = redis.NewScript(`
-- this script has side-effects, so it requires replicate commands mode
redis.replicate_commands()

local rate_limit_key = KEYS[1]
local burst = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local period = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])

local emission_interval = period / rate
local increment = emission_interval * cost
local burst_offset = emission_interval * burst

-- redis returns time as an array containing two integers: seconds of the
-- epoch time (10 digits) and microseconds (6 digits). convert them to a
-- floating point number relative to Jan 1, 2017 to avoid 64-bit double
-- precision issues. valid until Wed, 09 Sep 2048.
local jan_1_2017 = 1483228800
local now = redis.call("TIME")
now = (now[1] - jan_1_2017) + (now[2] / 1000000)

local tat = redis.call("GET", rate_limit_key)
if not tat then
  tat = now
else
  tat = tonumber(tat)
end

-- protect against clock going backwards or long idle periods
tat = math.max(tat, now)

local new_tat = tat + increment
local allow_at = new_tat - burst_offset

local diff = now - allow_at
local remaining = diff / emission_interval

if remaining < 0 then
  local reset_after = tat - now
  local retry_after = diff * -1
  return {
    0, -- allowed
    0, -- remaining
    tostring(retry_after),
    tostring(reset_after),
  }
end

local reset_after = new_tat - now
if reset_after > 0 then
  redis.call("SET", rate_limit_key, new_tat, "EX", math.ceil(reset_after))
end
local retry_after = -1
return {cost, remaining, tostring(retry_after), tostring(reset_after)}
`)

// RateResult is the outcome of a rate limit check with quota metadata,
// suitable for HTTP 429 headers (Retry-After / X-RateLimit-*).
type RateResult struct {
	// Allowed is the number of requests consumed; 0 when denied.
	Allowed int
	// Remaining is the number of requests still permitted in the window.
	Remaining int
	// RetryAfter is the duration to wait before the next request is allowed.
	// -1 means the request was allowed.
	RetryAfter time.Duration
	// ResetAfter is the duration until the limit fully resets.
	ResetAfter time.Duration
}

// GCRALimit defines the rate limit parameters for RedisGCRALimiter.
type GCRALimit struct {
	// Rate is the number of requests allowed per Period.
	Rate int
	// Burst is the burst capacity; defaults to Rate when zero.
	Burst int
	// Period is the time window; defaults to time.Second when zero.
	Period time.Duration
}

// RedisGCRALimiter implements the GCRA (leaky bucket) algorithm backed by a
// single Redis STRING key. Unlike the sliding window log, memory is O(1) per
// key; unlike the fixed window, the window is exact (no boundary burst).
type RedisGCRALimiter struct {
	cmd      redis.Cmdable
	limit    GCRALimit
	failOpen bool
}

// NewRedisGCRALimiter creates a GCRA rate limiter.
//
//	limiter := ratelimit.NewRedisGCRALimiter(rdb, ratelimit.GCRALimit{
//	    Rate: 100, Burst: 200, Period: time.Second, // 100 req/s, burst 200
//	})
func NewRedisGCRALimiter(cmd redis.Cmdable, limit GCRALimit) *RedisGCRALimiter {
	if limit.Rate <= 0 {
		limit.Rate = 1
	}
	if limit.Burst <= 0 {
		limit.Burst = limit.Rate
	}
	if limit.Period <= 0 {
		limit.Period = time.Second
	}
	return &RedisGCRALimiter{
		cmd:      cmd,
		limit:    limit,
		failOpen: true,
	}
}

// WithFailOpen controls behavior when Redis is unreachable:
// true allows the request, false denies with ErrRedisUnavailable.
func (l *RedisGCRALimiter) WithFailOpen(failOpen bool) *RedisGCRALimiter {
	l.failOpen = failOpen
	return l
}

// Allow checks whether a single request for the key is allowed.
// It implements RedisLimiter.
func (l *RedisGCRALimiter) Allow(ctx context.Context, key string) (bool, error) {
	return l.AllowN(ctx, key, 1)
}

// AllowN checks whether n requests for the key are allowed.
// It implements RedisLimiter.
func (l *RedisGCRALimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	res, err := l.AllowResult(ctx, key, n)
	if err != nil {
		return false, err
	}
	return res.Allowed > 0, nil
}

// AllowResult checks n requests and returns quota metadata (Remaining,
// RetryAfter, ResetAfter) alongside the decision.
func (l *RedisGCRALimiter) AllowResult(ctx context.Context, key string, n int) (*RateResult, error) {
	if n <= 0 {
		return &RateResult{Allowed: n, RetryAfter: -1}, nil
	}

	res, err := gcraScript.Run(ctx, l.cmd,
		[]string{key},
		strconv.Itoa(l.limit.Burst),
		strconv.Itoa(l.limit.Rate),
		strconv.FormatFloat(l.limit.Period.Seconds(), 'f', -1, 64),
		strconv.Itoa(n),
	).Result()
	if err != nil {
		if l.failOpen {
			return &RateResult{Allowed: n, RetryAfter: -1}, nil
		}
		return nil, fmt.Errorf("%w: %v", ErrRedisUnavailable, err)
	}

	return decodeGCRAResult(res)
}

// decodeGCRAResult parses the Lua array [allowed, remaining, retry_after, reset_after].
func decodeGCRAResult(res any) (*RateResult, error) {
	values, ok := res.([]any)
	if !ok || len(values) != 4 {
		return nil, errors.New("ratelimit: unexpected gcra script result")
	}

	// The Lua script returns retry_after = -1 (seconds) when allowed.
	// Normalize any negative value to -1 so `RetryAfter == -1` always means
	// "not limited", matching the redis_rate convention.
	retryAfter := toSecondsDuration(values[2])
	if retryAfter < 0 {
		retryAfter = -1
	}

	return &RateResult{
		Allowed:    int(toInt64(values[0])),
		Remaining:  int(toFloat64(values[1])),
		RetryAfter: retryAfter,
		ResetAfter: toSecondsDuration(values[3]),
	}, nil
}

// toInt64 converts a Lua integer result to int64.
func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

// toFloat64 converts a Lua number result to float64.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}

// toSecondsDuration parses a Lua string holding seconds (float) into a Duration.
func toSecondsDuration(v any) time.Duration {
	s, ok := v.(string)
	if !ok {
		return -1
	}
	sec, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1
	}
	return time.Duration(sec * float64(time.Second))
}
