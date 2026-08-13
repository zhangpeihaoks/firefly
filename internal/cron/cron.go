// Package cron provides scheduled task execution for the Firefly framework.
//
// It wraps robfig/cron and adds distributed execution semantics for clustered
// deployments:
//
//   - ModeSingleton: the task runs on exactly one instance (Redis lock).
//   - ModeAll:       the task runs on every instance.
//   - ModeShard:     the task runs on every instance, each handling a shard
//     of the workload via a ShardContext (index/total).
//
// A Scheduler implements app.Lifecycle, so it starts with the application and
// stops gracefully on shutdown. The Job interface is host-agnostic: the same
// job can run inside the process scheduler or be handed to a distributed
// platform (e.g. XXL-JOB) without rewriting.
package cron

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/zhangpeihaoks/firefly/internal/distlock"
	"github.com/redis/go-redis/v9"
)

// defaultLockTTL bounds how long a singleton job's lock is held after the
// executing instance dies.
const defaultLockTTL = 30 * time.Second

// cronParser accepts 6-field specs (with seconds), which the scheduler uses.
var cronParser = cron.NewParser(
	cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// Job is a unit of scheduled work. Implementations must be safe for
// concurrent execution (a single scheduler instance runs jobs in parallel by
// default).
type Job interface {
	// Name returns the job name, used for logging and lock keys.
	Name() string
	// Run executes the job. In ModeShard, use ShardFromContext to obtain the
	// shard assignment.
	Run(ctx context.Context) error
}

// Mode controls how a task behaves across a cluster.
type Mode int

const (
	// ModeSingleton runs the task on exactly one instance using a Redis lock.
	// Instances that fail to acquire the lock skip this round.
	ModeSingleton Mode = iota
	// ModeAll runs the task on every instance.
	ModeAll
	// ModeShard runs the task on every instance, each receiving a
	// ShardContext so it can process a slice of the workload.
	ModeShard
)

func (m Mode) String() string {
	switch m {
	case ModeSingleton:
		return "singleton"
	case ModeAll:
		return "all"
	case ModeShard:
		return "shard"
	default:
		return fmt.Sprintf("unknown(%d)", int(m))
	}
}

// ShardContext describes this instance's share of a sharded workload.
type ShardContext struct {
	// Index is this instance's 0-based position among all members.
	Index int
	// Total is the number of cluster members participating in the shard.
	Total int
}

type shardKey struct{}

// WithShard returns a context carrying the shard assignment.
func WithShard(ctx context.Context, sc ShardContext) context.Context {
	return context.WithValue(ctx, shardKey{}, sc)
}

// ShardFromContext extracts the shard assignment; ok is false in non-shard
// modes.
func ShardFromContext(ctx context.Context) (sc ShardContext, ok bool) {
	sc, ok = ctx.Value(shardKey{}).(ShardContext)
	return sc, ok
}

// entry is a registered task with its execution mode.
type entry struct {
	spec string
	job  Job
	mode Mode
}

// Scheduler runs registered jobs on a cron schedule.
// It implements app.Lifecycle.
type Scheduler struct {
	mu      sync.Mutex
	cron    *cron.Cron
	entries []*entry
	logger  *slog.Logger

	lockCmd     redis.Cmdable // enables ModeSingleton
	coordinator ShardCoordinator // enables ModeShard
}

// Option configures a Scheduler.
type Option func(*Scheduler)

// WithLogger sets the scheduler logger.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Scheduler) { s.logger = logger }
}

// WithRedisLock enables ModeSingleton by providing the Redis client used for
// distributed locking.
func WithRedisLock(cmd redis.Cmdable) Option {
	return func(s *Scheduler) { s.lockCmd = cmd }
}

// WithShardCoordinator enables ModeShard by providing the cluster topology
// source.
func WithShardCoordinator(c ShardCoordinator) Option {
	return func(s *Scheduler) { s.coordinator = c }
}

// New creates a scheduler. Jobs are registered with AddJob before the
// scheduler is started via app.RegisterLifecycle or Start.
func New(opts ...Option) *Scheduler {
	s := &Scheduler{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// JobOption configures a registered job.
type JobOption func(*entry)

// WithMode sets the job's execution mode (default ModeSingleton when a lock
// backend is configured, ModeAll otherwise).
func WithMode(m Mode) JobOption {
	return func(e *entry) { e.mode = m }
}

// AddJob registers a job with a cron spec. Both 6-field (with seconds,
// e.g. "*/5 * * * * *") and 5-field ("0 3 * * *") specs are accepted; a
// 5-field spec is treated as seconds=0.
func (s *Scheduler) AddJob(spec string, job Job, opts ...JobOption) error {
	if job == nil {
		return fmt.Errorf("cron: nil job")
	}
	spec = normalizeSpec(spec)
	if _, err := cronParser.Parse(spec); err != nil {
		return fmt.Errorf("cron: invalid schedule %q: %v", spec, err)
	}

	e := &entry{spec: spec, job: job}
	if s.lockCmd == nil && s.coordinator == nil {
		e.mode = ModeAll
	} else {
		e.mode = ModeSingleton
	}
	for _, opt := range opts {
		opt(e)
	}

	s.mu.Lock()
	s.entries = append(s.entries, e)
	s.mu.Unlock()
	return nil
}

// Start implements app.Lifecycle: registers the shard coordinator and starts
// the cron engine.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		return nil // already started
	}

	if s.coordinator != nil {
		if err := s.coordinator.Register(ctx); err != nil {
			return fmt.Errorf("cron: register shard coordinator: %w", err)
		}
	}

	c := cron.New(cron.WithParser(cronParser), cron.WithChain(cron.Recover(cron.DefaultLogger)))
	for _, e := range s.entries {
		entry := e
		if _, err := c.AddFunc(entry.spec, func() { s.run(entry) }); err != nil {
			return fmt.Errorf("cron: add job %s: %w", entry.job.Name(), err)
		}
	}
	s.cron = c
	c.Start()

	for _, e := range s.entries {
		s.logger.Info("cron job scheduled",
			"job", e.job.Name(),
			"spec", e.spec,
			"mode", e.mode.String(),
		)
	}
	return nil
}

// Stop implements app.Lifecycle: stops the cron engine and releases the
// shard coordinator.
func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		s.cron.Stop()
		s.cron = nil
	}
	if s.coordinator != nil {
		s.coordinator.Close()
	}
	return nil
}

// run executes a job according to its mode.
func (s *Scheduler) run(e *entry) {
	ctx := context.Background()
	s.logger.Debug("cron job triggered", "job", e.job.Name(), "mode", e.mode.String())

	switch e.mode {
	case ModeSingleton:
		if s.lockCmd == nil {
			s.logger.Error("cron: singleton job without Redis lock backend", "job", e.job.Name())
			return
		}
		mu := distlock.NewMutex(s.lockCmd, "cron:lock:"+e.job.Name(), distlock.WithTTL(defaultLockTTL))
		if err := mu.TryLock(ctx); err != nil {
			s.logger.Debug("cron: skip singleton job (lock held elsewhere)", "job", e.job.Name())
			return
		}
		defer mu.Unlock(ctx)
		s.execute(ctx, e)

	case ModeAll:
		s.execute(ctx, e)

	case ModeShard:
		if s.coordinator == nil {
			s.logger.Error("cron: shard job without coordinator", "job", e.job.Name())
			return
		}
		shard, ok := s.shardAssignment(ctx)
		if !ok {
			s.logger.Warn("cron: instance not in shard membership, skipping", "job", e.job.Name())
			return
		}
		s.execute(WithShard(ctx, shard), e)
	}
}

// execute runs the job and logs failures.
func (s *Scheduler) execute(ctx context.Context, e *entry) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("cron job panicked", "job", e.job.Name(), "panic", r)
		}
	}()
	if err := e.job.Run(ctx); err != nil {
		s.logger.Error("cron job failed", "job", e.job.Name(), "error", err)
	}
}

// shardAssignment computes this instance's shard from the coordinator's
// membership list.
func (s *Scheduler) shardAssignment(ctx context.Context) (ShardContext, bool) {
	members, err := s.coordinator.Members(ctx)
	if err != nil {
		s.logger.Error("cron: failed to list shard members", "error", err)
		return ShardContext{}, false
	}
	sort.Strings(members)
	self := s.coordinator.Self()
	for i, m := range members {
		if m == self {
			return ShardContext{Index: i, Total: len(members)}, true
		}
	}
	return ShardContext{}, false
}

// Name implements a stable identity for logging and lock keys.

// normalizeSpec converts a 5-field spec to 6-field (prepending seconds=0)
// so the scheduler's second-precision parser accepts both forms.
func normalizeSpec(spec string) string {
	if len(strings.Fields(spec)) == 5 {
		return "0 " + spec
	}
	return spec
}
