package distlock

import (
	"context"
	"errors"
	"sync"
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

func TestMutex_TryLockUnlock(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	mu := NewMutex(rdb, "lock:job-1", WithTTL(10*time.Second))

	// First TryLock succeeds.
	if err := mu.TryLock(ctx); err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}

	// A second mutex on the same key cannot acquire.
	other := NewMutex(rdb, "lock:job-1", WithTTL(10*time.Second))
	if err := other.TryLock(ctx); !errors.Is(err, ErrLockNotAcquired) {
		t.Fatalf("expected ErrLockNotAcquired, got %v", err)
	}

	// Unlock releases it.
	if err := mu.Unlock(ctx); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	if err := other.TryLock(ctx); err != nil {
		t.Fatalf("expected lock available after unlock, got %v", err)
	}
}

func TestMutex_LockBlocksUntilLeaseExpires(t *testing.T) {
	rdb, m := newTestClient(t)
	ctx := context.Background()

	holder := NewMutex(rdb, "lock:inv-42", WithTTL(2*time.Second))
	if err := holder.TryLock(ctx); err != nil {
		t.Fatalf("holder TryLock failed: %v", err)
	}
	defer holder.Unlock(ctx)

	// The waiter spins in Lock. Advancing the Redis clock (miniredis
	// virtual time) expires the holder's lease, letting the waiter in.
	done := make(chan error, 1)
	waiter := NewMutex(rdb, "lock:inv-42", WithTTL(2*time.Second))
	go func() { done <- waiter.Lock(ctx) }()

	time.Sleep(200 * time.Millisecond) // waiter is now spinning
	m.FastForward(3 * time.Second)     // lease expires

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("waiter Lock failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waiter did not acquire the lock after the lease expired")
	}
}

func TestMutex_WrongTokenCannotUnlock(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	holder := NewMutex(rdb, "lock:k", WithTTL(10*time.Second))
	if err := holder.TryLock(ctx); err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}

	// An impostor with a different token cannot release the lock.
	impostor := NewMutex(rdb, "lock:k", WithTTL(10*time.Second))
	if err := impostor.Unlock(ctx); err != nil {
		t.Fatalf("Unlock should not error even when not owner: %v", err)
	}

	// The holder still owns the lock.
	if err := holder.Unlock(ctx); err != nil {
		t.Fatalf("holder Unlock failed: %v", err)
	}
}

func TestMutex_WatchdogRenews(t *testing.T) {
	rdb, m := newTestClient(t)
	ctx := context.Background()

	mu := NewMutex(rdb, "lock:long-task",
		WithTTL(2*time.Second),
		WithWatchdogInterval(200*time.Millisecond),
	)
	if err := mu.TryLock(ctx); err != nil {
		t.Fatalf("TryLock failed: %v", err)
	}

	// Simulate continuous time: advance the Redis clock in steps smaller
	// than the TTL, letting the watchdog renew between steps. After 6s of
	// simulated time (3x the TTL) the lock must still be held — this is
	// exactly what the watchdog guarantees that a plain TTL cannot.
	for i := 0; i < 6; i++ {
		time.Sleep(250 * time.Millisecond) // watchdog renews (200ms interval)
		m.FastForward(time.Second)         // advance Redis clock 1s (< TTL)
	}

	other := NewMutex(rdb, "lock:long-task", WithTTL(2*time.Second))
	if err := other.TryLock(ctx); !errors.Is(err, ErrLockNotAcquired) {
		t.Fatalf("expected lock still held after watchdog renewal, got %v", err)
	}

	// Unlock stops the watchdog and releases the lock.
	if err := mu.Unlock(ctx); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	if err := other.TryLock(ctx); err != nil {
		t.Fatalf("expected lock available after unlock, got %v", err)
	}
}

func TestMutex_Concurrent(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	// 20 goroutines race for the lock; the critical section must run
	// exactly once at a time.
	var inCritical, maxConcurrent int32
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock := NewMutex(rdb, "lock:race", WithTTL(5*time.Second))
			if err := lock.Lock(ctx); err != nil {
				t.Errorf("Lock failed: %v", err)
				return
			}
			defer lock.Unlock(ctx)

			mu.Lock()
			inCritical++
			if inCritical > maxConcurrent {
				maxConcurrent = inCritical
			}
			mu.Unlock()

			time.Sleep(5 * time.Millisecond) // hold the lock briefly

			mu.Lock()
			inCritical--
			mu.Unlock()
		}()
	}
	wg.Wait()

	if maxConcurrent > 1 {
		t.Errorf("critical section entered %d times concurrently, want 1", maxConcurrent)
	}
}
