package id

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { client.Close() })
	return client, m
}

func TestRedisClock_AlignsToRedisTime(t *testing.T) {
	rdb, m := newTestClient(t)
	ctx := context.Background()

	// Pin the Redis server clock to a known instant.
	redisTime := time.Unix(5000, 123456000) // 5000.123456s
	m.SetTime(redisTime)

	clock, err := NewRedisClock(ctx, rdb)
	if err != nil {
		t.Fatalf("NewRedisClock failed: %v", err)
	}

	// Now() must be close to the Redis time base, regardless of local clock.
	now := clock.Now()
	wantMs := redisTime.UnixMilli()
	if now < wantMs-2000 || now > wantMs+2000 {
		t.Errorf("Now() = %d, want ~%d (Redis time base)", now, wantMs)
	}
}

func TestRedisClock_Resync(t *testing.T) {
	rdb, m := newTestClient(t)
	ctx := context.Background()

	m.SetTime(time.Unix(5000, 0))
	clock, err := NewRedisClock(ctx, rdb)
	if err != nil {
		t.Fatalf("NewRedisClock failed: %v", err)
	}

	// The Redis clock moves forward; a resync picks up the new base.
	m.SetTime(time.Unix(6000, 0))
	if err := clock.Sync(ctx); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	now := clock.Now()
	if now < 6000000-2000 || now > 6000000+2000 {
		t.Errorf("Now() = %d, want ~6000000 after resync", now)
	}
}

func TestRedisWorkerAllocator_UniqueSlots(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	alloc := NewRedisWorkerAllocator(rdb, "test-svc")

	s1, err := alloc.Allocate(ctx)
	if err != nil {
		t.Fatalf("first Allocate failed: %v", err)
	}
	s2, err := alloc.Allocate(ctx)
	if err != nil {
		t.Fatalf("second Allocate failed: %v", err)
	}
	if s1 == s2 {
		t.Fatalf("expected distinct worker slots, got %d twice", s1)
	}
}

func TestRedisWorkerAllocator_ReleaseReusesSlot(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	alloc := NewRedisWorkerAllocator(rdb, "test-svc")

	s1, err := alloc.Allocate(ctx)
	if err != nil {
		t.Fatalf("first Allocate failed: %v", err)
	}
	if err := alloc.Release(ctx, s1); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// The freed slot becomes available again (allocator scans from 0).
	s2, err := alloc.Allocate(ctx)
	if err != nil {
		t.Fatalf("Allocate after release failed: %v", err)
	}
	if s2 != s1 {
		t.Errorf("expected slot %d to be reused, got %d", s1, s2)
	}
}

func TestRedisWorkerAllocator_LeaseExpiry(t *testing.T) {
	rdb, m := newTestClient(t)
	ctx := context.Background()

	// Short lease, no renewal in the test window (long renewal interval).
	alloc := NewRedisWorkerAllocator(rdb, "test-svc",
		WithWorkerTTL(2*time.Second),
		WithWorkerRenewInterval(time.Hour),
	)

	s1, err := alloc.Allocate(ctx)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	// Advance the Redis clock past the lease: the slot expires.
	m.FastForward(3 * time.Second)

	// A fresh allocator sees the slot as free.
	alloc2 := NewRedisWorkerAllocator(rdb, "test-svc",
		WithWorkerTTL(2*time.Second),
		WithWorkerRenewInterval(time.Hour),
	)
	s2, err := alloc2.Allocate(ctx)
	if err != nil {
		t.Fatalf("Allocate after lease expiry failed: %v", err)
	}
	if s2 != s1 {
		t.Errorf("expected expired slot %d to be re-allocated, got %d", s1, s2)
	}
}

func TestRedisWorkerAllocator_Exhausted(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	alloc := NewRedisWorkerAllocator(rdb, "test-svc")

	// Occupy all slots.
	for slot := int64(0); slot <= maxWorkerID; slot++ {
		if _, err := alloc.Allocate(ctx); err != nil {
			t.Fatalf("Allocate slot %d failed: %v", slot, err)
		}
	}

	if _, err := alloc.Allocate(ctx); err == nil {
		t.Fatal("expected error when all worker slots are in use")
	}
}
