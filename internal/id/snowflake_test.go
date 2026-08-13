package id

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSnowflake_Basic(t *testing.T) {
	sf, err := NewSnowflake(1)
	if err != nil {
		t.Fatalf("NewSnowflake failed: %v", err)
	}

	id, err := sf.NextID()
	if err != nil {
		t.Fatalf("NextID failed: %v", err)
	}
	if id < 0 {
		t.Errorf("expected non-negative ID, got %d", id)
	}

	// Decode: timestamp high bits, worker bits, sequence.
	worker := (id >> workerShift) & maxWorkerID
	if worker != 1 {
		t.Errorf("expected worker 1, got %d", worker)
	}
}

func TestSnowflake_WorkerIDValidation(t *testing.T) {
	if _, err := NewSnowflake(-1); err == nil {
		t.Error("expected error for negative worker ID")
	}
	if _, err := NewSnowflake(maxWorkerID + 1); err == nil {
		t.Error("expected error for worker ID above max")
	}
	if _, err := NewSnowflake(maxWorkerID); err != nil {
		t.Errorf("expected max worker ID to be valid: %v", err)
	}
}

func TestSnowflake_Monotonic(t *testing.T) {
	sf, _ := NewSnowflake(7)
	const n = 10000

	prev, _ := sf.NextID()
	for i := 1; i < n; i++ {
		cur, err := sf.NextID()
		if err != nil {
			t.Fatalf("NextID failed: %v", err)
		}
		if cur <= prev {
			t.Fatalf("IDs not monotonic at %d: %d <= %d", i, cur, prev)
		}
		prev = cur
	}
}

func TestSnowflake_UniquenessConcurrent(t *testing.T) {
	sf, _ := NewSnowflake(3)

	const goroutines = 20
	const perGoroutine = 500

	ids := make(chan int64, goroutines*perGoroutine)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				id, err := sf.NextID()
				if err != nil {
					t.Errorf("NextID failed: %v", err)
					return
				}
				ids <- id
			}
		}()
	}
	wg.Wait()
	close(ids)

	seen := make(map[int64]struct{}, goroutines*perGoroutine)
	for id := range ids {
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate ID generated: %d", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != goroutines*perGoroutine {
		t.Errorf("expected %d unique IDs, got %d", goroutines*perGoroutine, len(seen))
	}
}

func TestSnowflake_ClockRollback(t *testing.T) {
	current := defaultEpochMs + int64(1000)
	sf, _ := NewSnowflake(1, WithNow(func() int64 { return current }))

	if _, err := sf.NextID(); err != nil {
		t.Fatalf("first ID failed: %v", err)
	}

	// Simulate a 100ms clock rollback.
	current = defaultEpochMs + int64(900)
	start := time.Now()
	_, err := sf.NextID()
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error when clock stays behind after rollback wait")
	}
	if elapsed < 90*time.Millisecond {
		t.Errorf("expected rollback wait of ~100ms, got %v", elapsed)
	}
}

func TestSnowflake_ClockRecovers(t *testing.T) {
	current := defaultEpochMs + int64(1000)
	sf, _ := NewSnowflake(1, WithNow(func() int64 { return current }))

	if _, err := sf.NextID(); err != nil {
		t.Fatalf("first ID failed: %v", err)
	}

	// Rollback then recover: the wait should pass and generation continues.
	current = defaultEpochMs + int64(980)
	go func() {
		time.Sleep(10 * time.Millisecond) // recover before the 20ms rollback wait ends
		current = defaultEpochMs + int64(1005) // catch up past lastMs=epoch+1000
	}()
	if _, err := sf.NextID(); err != nil {
		t.Fatalf("expected ID after clock recovery, got %v", err)
	}
}

func TestSnowflake_SequenceOverflow(t *testing.T) {
	current := defaultEpochMs + int64(1000)
	sf, _ := NewSnowflake(1, WithNow(func() int64 { return current }))

	// Fill the entire sequence space within the same millisecond.
	for i := 0; i < maxSequence; i++ {
		if _, err := sf.NextID(); err != nil {
			t.Fatalf("NextID %d failed: %v", i, err)
		}
	}

	// Next call overflows the sequence and spins until the next millisecond.
	go func() {
		time.Sleep(20 * time.Millisecond)
		current = defaultEpochMs + int64(1001)
	}()
	id, err := sf.NextID()
	if err != nil {
		t.Fatalf("NextID after overflow failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected valid ID after overflow, got %d", id)
	}
}

func TestSnowflake_CustomEpoch(t *testing.T) {
	epoch := time.Now().Add(-24 * time.Hour).UnixMilli()
	sf, _ := NewSnowflake(1, WithEpoch(epoch))

	for i := 0; i < 100; i++ {
		if _, err := sf.NextID(); err != nil {
			t.Fatalf("NextID failed: %v", err)
		}
	}
}

func ExampleSnowflake_NextID() {
	sf, err := NewSnowflake(1)
	if err != nil {
		panic(err)
	}
	id, err := sf.NextID()
	if err != nil {
		panic(err)
	}
	fmt.Println(id > 0)
	// Output: true
}
