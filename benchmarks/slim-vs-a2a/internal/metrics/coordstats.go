// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"sort"
	"sync"
	"time"
)

// CoordStats tracks coordination RPC quality metrics across a run.
type CoordStats struct {
	mu sync.Mutex

	RoundBudgetMS      int64
	ContextPushOps     int
	pushDurations      []int64
	MissingResponses   int
	DeadlineMisses     int
	BytesSent          int64
}

func NewCoordStats(roundBudgetMS int64) *CoordStats {
	return &CoordStats{RoundBudgetMS: roundBudgetMS}
}

// RecordCoordOp records one coordination operation (context, sync, cancel, notify).
func (c *CoordStats) RecordCoordOp(duration time.Duration, targets int, responded int, payloadBytes int, messagesSent int) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ContextPushOps++
	c.pushDurations = append(c.pushDurations, duration.Milliseconds())
	if targets > responded {
		c.MissingResponses += targets - responded
	}
	if c.RoundBudgetMS > 0 && duration.Milliseconds() > c.RoundBudgetMS {
		c.DeadlineMisses++
	}
	if messagesSent <= 0 {
		messagesSent = 1
	}
	c.BytesSent += int64(payloadBytes * messagesSent)
}

// P95ContextPushMS returns the p95 of recorded coordination op durations.
func (c *CoordStats) P95ContextPushMS() int64 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.p95Locked()
}

func (c *CoordStats) p95Locked() int64 {
	if len(c.pushDurations) == 0 {
		return 0
	}
	sorted := append([]int64(nil), c.pushDurations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)-1) * 0.95)
	return sorted[idx]
}

// Snapshot returns a copy of aggregate counters.
func (c *CoordStats) Snapshot() (ops int, missing, misses int, bytes int64, p95 int64) {
	if c == nil {
		return 0, 0, 0, 0, 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ContextPushOps, c.MissingResponses, c.DeadlineMisses, c.BytesSent, c.p95Locked()
}

// ApplyToRunResult copies coordination stats into a run result.
func (c *CoordStats) ApplyToRunResult(r *RunResult, coordTimeMS int64) {
	if c == nil || r == nil {
		return
	}
	ops, missing, misses, bytes, p95 := c.Snapshot()
	r.ContextPushOps = ops
	r.ContextPushP95MS = p95
	r.CoordMissingResponses = missing
	r.CoordDeadlineMisses = misses
	r.CoordBytesSent = bytes
	r.RoundBudgetMS = c.RoundBudgetMS
	if r.TotalWallClockMS > 0 && coordTimeMS > 0 {
		r.CoordTimeSharePct = float64(coordTimeMS) * 100 / float64(r.TotalWallClockMS)
	}
}
