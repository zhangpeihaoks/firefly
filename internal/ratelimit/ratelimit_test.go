package ratelimit

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newTestClient starts an in-memory Redis (miniredis) and returns a go-redis
// client pointing at it, plus the miniredis handle for time control.
func newTestClient(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { client.Close() })
	return client, m
}

func TestRedisTokenBucketLimiter(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	limiter := NewRedisTokenBucketLimiter(rdb, RedisTokenBucketConfig{
		Rate: 1, Burst: 3, // 1 token/s, burst 3
	})

	// Burst capacity is fully available immediately.
	for i := 0; i < 3; i++ {
		ok, err := limiter.Allow(ctx, "user:1")
		if err != nil || !ok {
			t.Fatalf("request %d: expected allowed (nil err), got ok=%v err=%v", i, ok, err)
		}
	}
	// Fourth request exceeds burst → denied.
	ok, err := limiter.Allow(ctx, "user:1")
	if err != nil || ok {
		t.Fatalf("expected denied after burst, got ok=%v err=%v", ok, err)
	}

	// Keys are isolated.
	ok, _ = limiter.Allow(ctx, "user:2")
	if !ok {
		t.Error("expected different key to be allowed")
	}

	// Refill: after ~1s the bucket regains a token.
	time.Sleep(1100 * time.Millisecond)
	ok, err = limiter.Allow(ctx, "user:1")
	if err != nil || !ok {
		t.Fatalf("expected allowed after refill, got ok=%v err=%v", ok, err)
	}
}

func TestRedisFixedWindowLimiter(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	limiter := NewRedisFixedWindowLimiter(rdb, RedisFixedWindowConfig{
		Limit: 3, Window: time.Minute,
	})

	for i := 0; i < 3; i++ {
		ok, err := limiter.Allow(ctx, "ip:1.2.3.4")
		if err != nil || !ok {
			t.Fatalf("request %d: expected allowed, got ok=%v err=%v", i, ok, err)
		}
	}
	ok, err := limiter.Allow(ctx, "ip:1.2.3.4")
	if err != nil || ok {
		t.Fatalf("expected denied over limit, got ok=%v err=%v", ok, err)
	}

	// Different key unaffected.
	ok, _ = limiter.Allow(ctx, "ip:5.6.7.8")
	if !ok {
		t.Error("expected different key to be allowed")
	}

	// New window → counter resets.
	windowStart := time.Now().Unix() / int64(time.Minute/time.Second)
	rdb.Del(ctx, "ip:1.2.3.4:"+strconv.FormatInt(windowStart, 10))
	ok, err = limiter.Allow(ctx, "ip:1.2.3.4")
	if err != nil || !ok {
		t.Fatalf("expected allowed after window reset, got ok=%v err=%v", ok, err)
	}
}

func TestRedisSlidingWindowLimiter(t *testing.T) {
	rdb, m := newTestClient(t)
	ctx := context.Background()

	limiter := NewRedisSlidingWindowLimiter(rdb, RedisSlidingWindowConfig{
		Limit: 2, Window: time.Minute,
	})

	for i := 0; i < 2; i++ {
		ok, err := limiter.Allow(ctx, "api:key-1")
		if err != nil || !ok {
			t.Fatalf("request %d: expected allowed, got ok=%v err=%v", i, ok, err)
		}
	}
	ok, err := limiter.Allow(ctx, "api:key-1")
	if err != nil || ok {
		t.Fatalf("expected denied at limit, got ok=%v err=%v", ok, err)
	}

	// Advance time past the window: members expire and the key is allowed again.
	m.FastForward(61 * time.Second)
	ok, err = limiter.Allow(ctx, "api:key-1")
	if err != nil || !ok {
		t.Fatalf("expected allowed after window elapsed, got ok=%v err=%v", ok, err)
	}
}

func TestRedisLimiters_FailClosed(t *testing.T) {
	// No Redis backend; closed client so every call errors.
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	client.Close()
	ctx := context.Background()

	tokenBucket := NewRedisTokenBucketLimiter(client, RedisTokenBucketConfig{Rate: 1, Burst: 10})
	if ok, err := tokenBucket.Allow(ctx, "k"); ok || err == nil {
		t.Errorf("fail-closed token bucket: expected error, got ok=%v err=%v", ok, err)
	}

	fixed := NewRedisFixedWindowLimiter(client, RedisFixedWindowConfig{Limit: 10, Window: time.Second})
	if ok, err := fixed.Allow(ctx, "k"); ok || err == nil {
		t.Errorf("fail-closed fixed window: expected error, got ok=%v err=%v", ok, err)
	}

	sliding := NewRedisSlidingWindowLimiter(client, RedisSlidingWindowConfig{Limit: 10, Window: time.Second})
	if ok, err := sliding.Allow(ctx, "k"); ok || err == nil {
		t.Errorf("fail-closed sliding window: expected error, got ok=%v err=%v", ok, err)
	}
}
