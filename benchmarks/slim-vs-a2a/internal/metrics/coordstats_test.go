// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics_test

import (
	"testing"
	"time"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
)

func TestCoordStatsP95AndDeadline(t *testing.T) {
	cs := metrics.NewCoordStats(10)
	cs.RecordCoordOp(5*time.Millisecond, 3, 3, 100, 1)
	cs.RecordCoordOp(15*time.Millisecond, 3, 3, 100, 1)
	cs.RecordCoordOp(20*time.Millisecond, 3, 2, 100, 1)

	if cs.P95ContextPushMS() != 15 {
		t.Fatalf("p95=%d want 15", cs.P95ContextPushMS())
	}
	_, missing, misses, bytes, _ := cs.Snapshot()
	if missing != 1 {
		t.Fatalf("missing=%d want 1", missing)
	}
	if misses != 2 {
		t.Fatalf("deadline misses=%d want 2", misses)
	}
	if bytes != 300 {
		t.Fatalf("bytes=%d want 300", bytes)
	}

	var r metrics.RunResult
	r.TotalWallClockMS = 100
	cs.ApplyToRunResult(&r, 30)
	if r.CoordTimeSharePct != 30 {
		t.Fatalf("share=%f want 30", r.CoordTimeSharePct)
	}
}
