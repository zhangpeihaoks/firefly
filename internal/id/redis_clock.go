package id

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClock provides millisecond timestamps aligned to the Redis server
// clock (TIME command).
//
// It calibrates the offset between local and Redis time at construction and
// re-calibrates periodically, then serves Now() from the local clock plus the
// cached offset — zero per-call Redis overhead, while keeping all instances
// of a deployment on the same time base so generated IDs cannot collide from
// local clock skew.
type RedisClock struct {
	cmd       redis.Cmdable
	mu        sync.Mutex
	offsetMs  int64 // redisMs - localMs
	lastSync  time.Time
	syncEvery time.Duration
}

// ClockOption configures a RedisClock.
type ClockOption func(*RedisClock)

// WithSyncEvery sets the re-calibration interval (default 30s).
func WithSyncEvery(d time.Duration) ClockOption {
	return func(c *RedisClock) { c.syncEvery = d }
}

// NewRedisClock calibrates against the Redis server time and returns a clock.
// It fails if Redis is unreachable at startup (the caller may fall back to a
// plain clock).
func NewRedisClock(ctx context.Context, cmd redis.Cmdable, opts ...ClockOption) (*RedisClock, error) {
	c := &RedisClock{
		cmd:       cmd,
		syncEvery: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	if err := c.Sync(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// Now returns the current millisecond timestamp on the Redis time base.
func (c *RedisClock) Now() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Periodic re-calibration compensates for drift. A failed sync keeps the
	// previous offset (best effort); the caller can resync explicitly.
	if time.Since(c.lastSync) > c.syncEvery {
		_ = c.syncLocked(context.Background())
	}
	return time.Now().UnixMilli() + c.offsetMs
}

// Sync re-calibrates the offset immediately.
func (c *RedisClock) Sync(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.syncLocked(ctx)
}

// syncLocked fetches Redis server time and recomputes the local offset.
func (c *RedisClock) syncLocked(ctx context.Context) error {
	redisTime, err := c.cmd.Time(ctx).Result()
	if err != nil {
		return err
	}
	c.offsetMs = redisTime.UnixMilli() - time.Now().UnixMilli()
	c.lastSync = time.Now()
	return nil
}
