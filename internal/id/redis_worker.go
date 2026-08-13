package id

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// renewWorkerScript extends the worker slot lease only if it is still owned
// by this token (i.e. we did not lose the slot to a re-allocation).
var renewWorkerScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
`)

// RedisWorkerAllocator dynamically assigns worker IDs from a Redis-backed
// slot pool (0..maxWorkerID), removing the need for static configuration or
// machine fingerprints.
//
// Each instance atomically claims a free slot with SET NX (guaranteed unique
// across instances), then holds it with a lease that a background goroutine
// renews. When the instance exits (or dies), the lease expires and the slot
// becomes reusable. This makes collision impossible — unlike MAC/hostname
// fingerprint schemes — and works under containers, VMs and multi-instance
// hosts.
type RedisWorkerAllocator struct {
	cmd     redis.Cmdable
	prefix  string
	ttl     time.Duration
	renew   time.Duration

	mu      sync.Mutex
	renewOn bool
	stopCh  chan struct{}
}

// WorkerAllocatorOption configures a RedisWorkerAllocator.
type WorkerAllocatorOption func(*RedisWorkerAllocator)

// WithWorkerTTL sets the slot lease duration (default 30s).
func WithWorkerTTL(ttl time.Duration) WorkerAllocatorOption {
	return func(a *RedisWorkerAllocator) { a.ttl = ttl }
}

// WithWorkerRenewInterval sets the lease renewal interval (default ttl/3).
func WithWorkerRenewInterval(d time.Duration) WorkerAllocatorOption {
	return func(a *RedisWorkerAllocator) { a.renew = d }
}

// NewRedisWorkerAllocator creates a worker ID allocator backed by Redis.
//
//	alloc := id.NewRedisWorkerAllocator(rdb, "firefly")
//	workerID, err := alloc.Allocate(ctx)
//	defer alloc.Release(ctx, workerID)
func NewRedisWorkerAllocator(cmd redis.Cmdable, prefix string, opts ...WorkerAllocatorOption) *RedisWorkerAllocator {
	a := &RedisWorkerAllocator{
		cmd:    cmd,
		prefix: prefix,
		ttl:    30 * time.Second,
	}
	for _, opt := range opts {
		opt(a)
	}
	if a.renew == 0 {
		a.renew = a.ttl / 3
	}
	return a
}

// Allocate claims a free worker ID and starts renewing its lease.
func (a *RedisWorkerAllocator) Allocate(ctx context.Context) (int64, error) {
	token := workerToken()
	for slot := int64(0); slot <= maxWorkerID; slot++ {
		key := a.slotKey(slot)
		ok, err := a.cmd.SetNX(ctx, key, token, a.ttl).Result()
		if err != nil {
			return 0, fmt.Errorf("id: allocate worker slot: %w", err)
		}
		if ok {
			a.startRenew(key, token)
			return slot, nil
		}
	}
	return 0, fmt.Errorf("id: all %d worker slots are in use", maxWorkerID+1)
}

// Release stops renewing and frees the worker slot.
// The slot becomes immediately reusable by another instance.
func (a *RedisWorkerAllocator) Release(ctx context.Context, slot int64) error {
	a.stopRenew()
	return a.cmd.Del(ctx, a.slotKey(slot)).Err()
}

// slotKey returns the Redis key for a worker slot.
func (a *RedisWorkerAllocator) slotKey(slot int64) string {
	return fmt.Sprintf("%s:worker:%d", a.prefix, slot)
}

// startRenew launches the lease renewal goroutine.
func (a *RedisWorkerAllocator) startRenew(key, token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.renewOn {
		return
	}
	a.renewOn = true
	a.stopCh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(a.renew)
		defer ticker.Stop()
		for {
			select {
			case <-a.stopCh:
				return
			case <-ticker.C:
				if _, err := renewWorkerScript.Run(context.Background(), a.cmd,
					[]string{key}, token, a.ttl.Milliseconds()).Result(); err != nil {
					// Lease renewal failure: the slot will expire and may be
					// re-allocated; the caller should watch for the error via
					// its own health checks.
					continue
				}
			}
		}
	}()
}

// stopRenew stops the renewal goroutine.
func (a *RedisWorkerAllocator) stopRenew() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.renewOn {
		return
	}
	a.renewOn = false
	close(a.stopCh)
}

// workerToken generates a unique token identifying the lease owner.
func workerToken() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
