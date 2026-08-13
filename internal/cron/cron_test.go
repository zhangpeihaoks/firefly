package cron

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// testJob is a Job backed by a function.
type testJob struct {
	name string
	fn   func(ctx context.Context) error
}

func (j *testJob) Name() string { return j.name }
func (j *testJob) Run(ctx context.Context) error {
	if j.fn != nil {
		return j.fn(ctx)
	}
	return nil
}

func newTestClient(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { client.Close() })
	return client, m
}

// TestScheduler_Basic verifies a job fires on its schedule.
func TestScheduler_Basic(t *testing.T) {
	sch := New()
	var count atomic.Int32
	if err := sch.AddJob("*/1 * * * * *", &testJob{
		name: "every-second",
		fn:   func(context.Context) error { count.Add(1); return nil },
	}); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	if err := sch.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	time.Sleep(1500 * time.Millisecond)
	sch.Stop(context.Background())

	if count.Load() < 1 {
		t.Errorf("expected job to run at least once, got %d", count.Load())
	}
}

// TestScheduler_SpecValidation rejects bad specs and accepts 5-field specs.
func TestScheduler_SpecValidation(t *testing.T) {
	sch := New()
	job := &testJob{name: "job"}

	if err := sch.AddJob("not a spec", job); err == nil {
		t.Error("expected error for invalid spec")
	}
	// 5-field spec is normalized (seconds=0).
	if err := sch.AddJob("0 3 * * *", job); err != nil {
		t.Errorf("expected 5-field spec to be accepted, got %v", err)
	}
}

// TestScheduler_ModeSingleton verifies only one instance executes per round.
func TestScheduler_ModeSingleton(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	sch1 := New(WithRedisLock(rdb))
	sch2 := New(WithRedisLock(rdb))

	var c1, c2 atomic.Int32
	spec := "*/1 * * * * *"
	sch1.AddJob(spec, &testJob{name: "singleton", fn: func(context.Context) error { c1.Add(1); return nil }}, WithMode(ModeSingleton))
	sch2.AddJob(spec, &testJob{name: "singleton", fn: func(context.Context) error { c2.Add(1); return nil }}, WithMode(ModeSingleton))

	if err := sch1.Start(ctx); err != nil {
		t.Fatalf("Start 1 failed: %v", err)
	}
	if err := sch2.Start(ctx); err != nil {
		t.Fatalf("Start 2 failed: %v", err)
	}
	time.Sleep(2200 * time.Millisecond)
	sch1.Stop(ctx)
	sch2.Stop(ctx)

	sum := c1.Load() + c2.Load()
	// ~2-3 trigger rounds; each round runs on exactly one instance.
	if sum < 1 {
		t.Fatal("expected at least one execution")
	}
	if sum > 4 {
		t.Errorf("singleton executed %d times, expected <= rounds (~3): one per round", sum)
	}
}

// TestScheduler_ModeAll verifies every instance executes.
func TestScheduler_ModeAll(t *testing.T) {
	sch1 := New()
	sch2 := New()

	var c1, c2 atomic.Int32
	spec := "*/1 * * * * *"
	sch1.AddJob(spec, &testJob{name: "all", fn: func(context.Context) error { c1.Add(1); return nil }}, WithMode(ModeAll))
	sch2.AddJob(spec, &testJob{name: "all", fn: func(context.Context) error { c2.Add(1); return nil }}, WithMode(ModeAll))

	sch1.Start(context.Background())
	sch2.Start(context.Background())
	time.Sleep(2200 * time.Millisecond)
	sch1.Stop(context.Background())
	sch2.Stop(context.Background())

	if c1.Load() < 1 || c2.Load() < 1 {
		t.Errorf("expected both instances to execute, got c1=%d c2=%d", c1.Load(), c2.Load())
	}
}

// TestScheduler_ModeShard verifies shard contexts are computed from cluster
// membership.
func TestScheduler_ModeShard(t *testing.T) {
	rdb, _ := newTestClient(t)
	ctx := context.Background()

	coordA := NewRedisCoordinator(rdb, "svc:members", WithSelf("A"))
	coordB := NewRedisCoordinator(rdb, "svc:members", WithSelf("B"))

	schA := New(WithShardCoordinator(coordA))
	schB := New(WithShardCoordinator(coordB))

	var mu sync.Mutex
	var shardsA, shardsB []ShardContext
	spec := "*/1 * * * * *"
	schA.AddJob(spec, &testJob{
		name: "shard",
		fn: func(ctx context.Context) error {
			sc, ok := ShardFromContext(ctx)
			if ok {
				mu.Lock()
				shardsA = append(shardsA, sc)
				mu.Unlock()
			}
			return nil
		},
	}, WithMode(ModeShard))
	schB.AddJob(spec, &testJob{
		name: "shard",
		fn: func(ctx context.Context) error {
			sc, ok := ShardFromContext(ctx)
			if ok {
				mu.Lock()
				shardsB = append(shardsB, sc)
				mu.Unlock()
			}
			return nil
		},
	}, WithMode(ModeShard))

	if err := schA.Start(ctx); err != nil {
		t.Fatalf("Start A failed: %v", err)
	}
	if err := schB.Start(ctx); err != nil {
		t.Fatalf("Start B failed: %v", err)
	}
	time.Sleep(2200 * time.Millisecond)
	schA.Stop(ctx)
	schB.Stop(ctx)

	mu.Lock()
	defer mu.Unlock()
	if len(shardsA) == 0 || len(shardsB) == 0 {
		t.Fatalf("expected both instances to run sharded jobs, got A=%d B=%d", len(shardsA), len(shardsB))
	}
	// Lexically sorted members: A is index 0, B is index 1, total 2.
	if got := shardsA[0]; got.Index != 0 || got.Total != 2 {
		t.Errorf("instance A shard = %+v, want {0 2}", got)
	}
	if got := shardsB[0]; got.Index != 1 || got.Total != 2 {
		t.Errorf("instance B shard = %+v, want {1 2}", got)
	}
}

// TestRedisCoordinator_MembershipExpiry verifies dead instances drop out.
func TestRedisCoordinator_MembershipExpiry(t *testing.T) {
	rdb, m := newTestClient(t)
	ctx := context.Background()

	// Long heartbeat interval: the instance stops heartbeating immediately.
	coord := NewRedisCoordinator(rdb, "svc:members",
		WithSelf("C"),
		WithCoordinatorTTL(2*time.Second),
		WithCoordinatorHeartbeat(time.Hour),
	)
	if err := coord.Register(ctx); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	members, err := coord.Members(ctx)
	if err != nil || len(members) != 1 {
		t.Fatalf("expected 1 member, got %v err=%v", members, err)
	}

	// Advance the clock past the membership window: the member expires.
	m.FastForward(3 * time.Second)
	members, err = coord.Members(ctx)
	if err != nil {
		t.Fatalf("Members failed: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("expected membership to expire, got %v", members)
	}
}
