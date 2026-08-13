// Package distlock provides a Redis-backed distributed mutex.
//
// The lock is acquired atomically with SET NX PX and released via a Lua
// script that verifies the owner token, so a holder can never release
// someone else's lock (e.g. after its TTL expired). An optional watchdog
// goroutine renews the lock while it is held.
package distlock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrLockNotAcquired is returned by TryLock when the lock is held elsewhere.
var ErrLockNotAcquired = errors.New("distlock: lock not acquired")

// unlockScript atomically verifies the token before deleting the key.
var unlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`)

// renewScript extends the lock TTL only if it is still owned by this token.
var renewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
`)

// Mutex is a distributed lock on a single Redis key.
// It is safe for concurrent use, though a Mutex is typically owned by one
// goroutine at a time.
type Mutex struct {
	cmd   redis.Cmdable
	key   string
	token string
	ttl   time.Duration

	watchdogEnabled bool
	watchdog        time.Duration // renewal interval
	stopCh          chan struct{}
	watchdogOn      bool
	mu              sync.Mutex
}

// Option configures a Mutex.
type Option func(*Mutex)

// WithTTL sets the lock lease duration (default 30s).
// The TTL bounds how long a lock can be held after the holder crashes.
func WithTTL(ttl time.Duration) Option {
	return func(m *Mutex) { m.ttl = ttl }
}

// WithWatchdog enables automatic renewal while the lock is held.
// The lease is renewed every ttl/3. If the holder dies, the lease expires
// after ttl and other nodes can acquire the lock.
func WithWatchdog() Option {
	return func(m *Mutex) { m.watchdogEnabled = true }
}

// WithWatchdogInterval enables the watchdog with an explicit renewal interval.
func WithWatchdogInterval(d time.Duration) Option {
	return func(m *Mutex) {
		m.watchdogEnabled = true
		m.watchdog = d
	}
}

// NewMutex creates a distributed mutex for the given key.
//
//	mu := distlock.NewMutex(rdb, "order:123", distlock.WithTTL(10*time.Second))
//	if err := mu.Lock(ctx); err != nil { ... }
//	defer mu.Unlock(ctx)
func NewMutex(cmd redis.Cmdable, key string, opts ...Option) *Mutex {
	m := &Mutex{
		cmd:   cmd,
		key:   key,
		token: randomToken(),
		ttl:   30 * time.Second,
	}
	for _, opt := range opts {
		opt(m)
	}
	if m.watchdog == 0 {
		m.watchdog = m.ttl / 3
	}
	return m
}

// TryLock attempts to acquire the lock once. It returns ErrLockNotAcquired
// immediately if the lock is held elsewhere.
func (m *Mutex) TryLock(ctx context.Context) error {
	ok, err := m.cmd.SetNX(ctx, m.key, m.token, m.ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrLockNotAcquired
	}
	m.startWatchdog()
	return nil
}

// Lock blocks until the lock is acquired or ctx is done.
// It retries with a short backoff; the retry interval is bounded so a busy
// lock does not spin hot.
func (m *Mutex) Lock(ctx context.Context) error {
	backoff := time.Millisecond
	for {
		err := m.TryLock(ctx)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrLockNotAcquired) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 100*time.Millisecond {
			backoff *= 2
		}
	}
}

// Unlock releases the lock. It is safe to call even if the lock was lost
// (e.g. expired while the holder was slow) — the token check prevents
// releasing a lock now owned by someone else.
func (m *Mutex) Unlock(ctx context.Context) error {
	m.stopWatchdog()

	res, err := unlockScript.Run(ctx, m.cmd, []string{m.key}, m.token).Result()
	if err != nil {
		return err
	}
	// 0 means the lock was already gone or owned by another holder; that is
	// not an error for the caller.
	_ = res
	return nil
}

// startWatchdog launches the renewal goroutine if enabled and not running.
func (m *Mutex) startWatchdog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.watchdogOn || !m.watchdogEnabled || m.watchdog <= 0 {
		return
	}
	m.watchdogOn = true
	m.stopCh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(m.watchdog)
		defer ticker.Stop()
		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				if err := m.renew(context.Background()); err != nil {
					// Renewal failures (e.g. Redis down) are logged by the
					// caller if needed; the lock will expire naturally.
					continue
				}
			}
		}
	}()
}

// stopWatchdog stops the renewal goroutine.
func (m *Mutex) stopWatchdog() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.watchdogOn {
		return
	}
	m.watchdogOn = false
	close(m.stopCh)
}

// renew extends the lease if still owned by this token.
func (m *Mutex) renew(ctx context.Context) error {
	res, err := renewScript.Run(ctx, m.cmd, []string{m.key}, m.token, m.ttl.Milliseconds()).Result()
	if err != nil {
		return err
	}
	if n, ok := res.(int64); ok && n == 0 {
		return ErrLockNotAcquired // lock lost, stop renewing
	}
	return nil
}

// randomToken generates a unique 16-byte lock token.
func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is catastrophic; fall back to a time-based token.
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b)
}
