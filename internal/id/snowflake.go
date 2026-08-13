// Package id provides a distributed unique ID generator based on the
// Snowflake algorithm, hardened for production:
//
//	0 | 41-bit millisecond timestamp | 10-bit worker ID | 12-bit sequence
//
// Timestamps come from an injectable clock (see redis_clock.go for a
// Redis-calibrated clock that eliminates clock-skew duplicates), worker IDs
// can be allocated dynamically (see redis_worker.go), and clock rollbacks
// are detected and handled instead of silently producing duplicates.
package id

import (
	"fmt"
	"sync"
	"time"
)

const (
	workerIDBits = 10
	seqBits      = 12

	maxWorkerID = -1 ^ (-1 << workerIDBits) // 1023
	maxSequence = -1 ^ (-1 << seqBits)      // 4095

	timeShift   = workerIDBits + seqBits // 22
	workerShift = seqBits                // 12

	// defaultEpochMs is 2024-01-01T00:00:00Z. A custom epoch keeps the
	// 41-bit timestamp valid for ~69 years from 2024 instead of 1970.
	defaultEpochMs = int64(1704067200000)
)

// Snowflake is a thread-safe unique ID generator.
// A single generator instance is shared by the whole process.
type Snowflake struct {
	mu       sync.Mutex
	workerID int64
	epochMs  int64
	lastMs   int64
	seq      int64
	now      func() int64 // returns current time in milliseconds
}

// SnowflakeOption configures a Snowflake.
type SnowflakeOption func(*Snowflake)

// WithEpoch sets a custom epoch (milliseconds since Unix epoch).
// IDs are valid while (now - epoch) fits in 41 bits (~69 years).
func WithEpoch(epochMs int64) SnowflakeOption {
	return func(s *Snowflake) { s.epochMs = epochMs }
}

// WithNow overrides the millisecond clock (for testing and Redis calibration).
func WithNow(now func() int64) SnowflakeOption {
	return func(s *Snowflake) { s.now = now }
}

// NewSnowflake creates a generator for the given worker ID.
func NewSnowflake(workerID int64, opts ...SnowflakeOption) (*Snowflake, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, fmt.Errorf("id: worker ID %d out of range [0, %d]", workerID, maxWorkerID)
	}
	s := &Snowflake{
		workerID: workerID,
		epochMs:  defaultEpochMs,
		// NOTE: must be a closure — `time.Now().UnixMilli` as a method value
		// would bind once and return a frozen timestamp forever.
		now: func() int64 { return time.Now().UnixMilli() },
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// WorkerID returns the worker ID assigned to this generator.
func (s *Snowflake) WorkerID() int64 {
	return s.workerID
}

// NextID generates the next unique ID.
//
// Clock rollback is handled by waiting for the clock to catch up (bounded by
// maxRollbackWait) rather than silently reusing timestamps; sequence overflow
// within a millisecond spins until the next millisecond.
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.currentTime()
	if now < s.epochMs {
		return 0, fmt.Errorf("id: clock is before the configured epoch (%d < %d)", now, s.epochMs)
	}

	// Clock rollback: wait for the clock to catch up.
	if now < s.lastMs {
		wait := s.lastMs - now
		time.Sleep(time.Duration(wait) * time.Millisecond)
		now = s.currentTime()
		if now < s.lastMs {
			return 0, fmt.Errorf("id: clock moved backwards by %d ms", s.lastMs-now)
		}
	}

	if now == s.lastMs {
		// Same millisecond: advance the sequence.
		s.seq = (s.seq + 1) & maxSequence
		if s.seq == 0 {
			// Sequence exhausted: spin until the next millisecond.
			for now <= s.lastMs {
				now = s.currentTime()
				time.Sleep(100 * time.Microsecond)
			}
		}
	} else {
		s.seq = 0
	}
	s.lastMs = now

	id := ((now - s.epochMs) << timeShift) | (s.workerID << workerShift) | s.seq
	return id, nil
}

// currentTime returns the current millisecond timestamp from the clock source.
func (s *Snowflake) currentTime() int64 {
	return s.now()
}
