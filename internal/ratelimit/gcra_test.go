package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newGCRAHarness starts miniredis and returns a GCRA limiter plus the
// miniredis handle for time control (GCRA reads Redis server time via TIME).
func newGCRAHarness(t *testing.T, limit GCRALimit) (*RedisGCRALimiter, *miniredis.Miniredis) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { client.Close() })
	return NewRedisGCRALimiter(client, limit), m
}

func TestRedisGCRALimiter_Basic(t *testing.T) {
	limiter, _ := newGCRAHarness(t, GCRALimit{Rate: 2, Burst: 2, Period: time.Second})
	ctx := context.Background()

	// Burst capacity allows the first two requests.
	for i := 0; i < 2; i++ {
		ok, err := limiter.Allow(ctx, "user:1")
		if err != nil || !ok {
			t.Fatalf("request %d: expected allowed, got ok=%v err=%v", i, ok, err)
		}
	}
	// Third request denied.
	ok, err := limiter.Allow(ctx, "user:1")
	if err != nil || ok {
		t.Fatalf("expected denied over rate, got ok=%v err=%v", ok, err)
	}

	// Keys are isolated.
	ok, _ = limiter.Allow(ctx, "user:2")
	if !ok {
		t.Error("expected different key to be allowed")
	}
}

func TestRedisGCRALimiter_Refill(t *testing.T) {
	limiter, m := newGCRAHarness(t, GCRALimit{Rate: 1, Burst: 1, Period: time.Second})
	ctx := context.Background()

	if ok, err := limiter.Allow(ctx, "api:key"); err != nil || !ok {
		t.Fatalf("expected first request allowed, got ok=%v err=%v", ok, err)
	}
	if ok, _ := limiter.Allow(ctx, "api:key"); ok {
		t.Fatal("expected second request denied (rate 1/s)")
	}

	// Advance Redis server time past one emission interval.
	m.FastForward(1100 * time.Millisecond)

	ok, err := limiter.Allow(ctx, "api:key")
	if err != nil || !ok {
		t.Fatalf("expected allowed after time passes, got ok=%v err=%v", ok, err)
	}
}

func TestRedisGCRALimiter_Result(t *testing.T) {
	limiter, _ := newGCRAHarness(t, GCRALimit{Rate: 2, Burst: 2, Period: time.Second})
	ctx := context.Background()

	// First request: allowed, RetryAfter == -1, some quota remaining.
	res, err := limiter.AllowResult(ctx, "ip:9.9.9.9", 1)
	if err != nil {
		t.Fatalf("AllowResult failed: %v", err)
	}
	if res.Allowed != 1 {
		t.Errorf("expected Allowed=1, got %d", res.Allowed)
	}
	if res.RetryAfter != -1 {
		t.Errorf("expected RetryAfter=-1 when allowed, got %v", res.RetryAfter)
	}
	if res.Remaining <= 0 {
		t.Errorf("expected Remaining > 0 after first request, got %d", res.Remaining)
	}

	// Exhaust the quota: one more allowed, then denied with positive RetryAfter.
	if _, err := limiter.Allow(ctx, "ip:9.9.9.9"); err != nil {
		t.Fatalf("second request should be allowed: %v", err)
	}
	res, err = limiter.AllowResult(ctx, "ip:9.9.9.9", 1)
	if err != nil {
		t.Fatalf("AllowResult failed: %v", err)
	}
	if res.Allowed != 0 {
		t.Errorf("expected Allowed=0 when denied, got %d", res.Allowed)
	}
	if res.RetryAfter <= 0 {
		t.Errorf("expected RetryAfter > 0 when denied, got %v", res.RetryAfter)
	}
	if res.ResetAfter <= 0 {
		t.Errorf("expected ResetAfter > 0 when denied, got %v", res.ResetAfter)
	}
}

func TestRedisGCRALimiter_FailClosed(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	client.Close()
	ctx := context.Background()

	limiter := NewRedisGCRALimiter(client, GCRALimit{Rate: 1, Burst: 10}).WithFailOpen(false)
	if ok, err := limiter.Allow(ctx, "k"); ok || err == nil {
		t.Errorf("fail-closed gcra: expected error, got ok=%v err=%v", ok, err)
	}
}

func TestDecodeGCRAResult(t *testing.T) {
	tests := []struct {
		name          string
		res           any
		wantAllowed   int
		wantRemaining int
		wantRetryAfter time.Duration
		wantErr       bool
	}{
		{
			name:          "allowed",
			res:           []any{int64(1), int64(3), "-1", "1.5"},
			wantAllowed:   1,
			wantRemaining: 3,
			wantRetryAfter: -1,
		},
		{
			name:          "denied with fractional retry",
			res:           []any{int64(0), int64(0), "0.5", "2.25"},
			wantAllowed:   0,
			wantRemaining: 0,
			wantRetryAfter: 500 * time.Millisecond,
		},
		{
			name:    "malformed",
			res:     []any{int64(1)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeGCRAResult(tt.res)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Allowed != tt.wantAllowed {
				t.Errorf("Allowed = %d, want %d", got.Allowed, tt.wantAllowed)
			}
			if got.Remaining != tt.wantRemaining {
				t.Errorf("Remaining = %d, want %d", got.Remaining, tt.wantRemaining)
			}
			if got.RetryAfter != tt.wantRetryAfter {
				t.Errorf("RetryAfter = %v, want %v", got.RetryAfter, tt.wantRetryAfter)
			}
		})
	}
}
