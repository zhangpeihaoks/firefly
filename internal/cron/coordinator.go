package cron

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ShardCoordinator provides the cluster topology used by ModeShard jobs.
// Members are identified by stable per-instance strings.
type ShardCoordinator interface {
	// Register announces this instance to the cluster and starts heartbeating
	// so its membership does not expire.
	Register(ctx context.Context) error
	// Members returns the sorted list of live instance identifiers.
	Members(ctx context.Context) ([]string, error)
	// Self returns this instance's identifier.
	Self() string
	// Close stops the heartbeat and removes this instance from the cluster.
	Close()
}

// RedisCoordinator tracks cluster members in a Redis sorted set.
//
// Each instance registers with ZADD (score = now) and heartbeats by updating
// its score + refreshing the key TTL. Membership expires if an instance dies,
// since its score ages out. This works across containers/VMs without relying
// on machine fingerprints.
type RedisCoordinator struct {
	cmd    redis.Cmdable
	key    string
	self   string
	ttl    time.Duration
	renew  time.Duration
	window time.Duration // membership validity window

	mu      sync.Mutex
	renewOn bool
	stopCh  chan struct{}
}

// RedisCoordinatorOption configures a RedisCoordinator.
type RedisCoordinatorOption func(*RedisCoordinator)

// WithCoordinatorTTL sets the membership validity window (default 30s).
func WithCoordinatorTTL(d time.Duration) RedisCoordinatorOption {
	return func(c *RedisCoordinator) { c.ttl = d }
}

// WithCoordinatorHeartbeat sets the heartbeat interval (default ttl/3).
func WithCoordinatorHeartbeat(d time.Duration) RedisCoordinatorOption {
	return func(c *RedisCoordinator) { c.renew = d }
}

// WithSelf overrides the instance identifier (default: random token).
func WithSelf(self string) RedisCoordinatorOption {
	return func(c *RedisCoordinator) { c.self = self }
}

// NewRedisCoordinator creates a cluster membership coordinator backed by a
// Redis sorted set.
func NewRedisCoordinator(cmd redis.Cmdable, key string, opts ...RedisCoordinatorOption) *RedisCoordinator {
	c := &RedisCoordinator{
		cmd:    cmd,
		key:    key,
		self:   fmt.Sprintf("instance-%d", time.Now().UnixNano()),
		ttl:    30 * time.Second,
		window: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.renew == 0 {
		c.renew = c.ttl / 3
	}
	return c
}

// Register adds this instance and starts the heartbeat.
func (c *RedisCoordinator) Register(ctx context.Context) error {
	if err := c.beat(ctx); err != nil {
		return err
	}
	c.startHeartbeat()
	return nil
}

// Members returns live instances: those whose last heartbeat falls within the
// validity window, sorted lexically.
func (c *RedisCoordinator) Members(ctx context.Context) ([]string, error) {
	min := fmt.Sprintf("(%d", time.Now().Add(-c.window).UnixNano())
	members, err := c.cmd.ZRangeByScore(ctx, c.key, &redis.ZRangeBy{
		Min: min,
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("cron: coordinator list members: %w", err)
	}
	return members, nil
}

// Self returns this instance's identifier.
func (c *RedisCoordinator) Self() string {
	return c.self
}

// Close stops the heartbeat and removes this instance from the cluster.
func (c *RedisCoordinator) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.renewOn {
		return
	}
	c.renewOn = false
	close(c.stopCh)
	// Best-effort removal; failures are logged by the caller if needed.
	_ = c.cmd.ZRem(context.Background(), c.key, c.self).Err()
}

// beat updates this instance's score and refreshes the key TTL.
func (c *RedisCoordinator) beat(ctx context.Context) error {
	score := float64(time.Now().UnixNano())
	pipe := c.cmd.Pipeline()
	pipe.ZAdd(ctx, c.key, redis.Z{Score: score, Member: c.self})
	pipe.Expire(ctx, c.key, c.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("cron: coordinator heartbeat: %w", err)
	}
	return nil
}

// startHeartbeat launches periodic beats.
func (c *RedisCoordinator) startHeartbeat() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.renewOn {
		return
	}
	c.renewOn = true
	c.stopCh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(c.renew)
		defer ticker.Stop()
		for {
			select {
			case <-c.stopCh:
				return
			case <-ticker.C:
				if err := c.beat(context.Background()); err != nil {
					// Heartbeat failure: membership may expire and the
					// instance will skip shard rounds until it recovers.
					continue
				}
			}
		}
	}()
}
