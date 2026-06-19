// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package benchlog

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	ImplSLIM = "slim-multicast"
	ImplA2A  = "a2a-grpc"
)

var (
	mu       sync.RWMutex
	runStart time.Time
	enabled  = true
	out      io.Writer = os.Stderr
)

// SetRunStart records the scheduler run start for elapsed_ms on each line.
func SetRunStart(t time.Time) {
	mu.Lock()
	runStart = t
	mu.Unlock()
}

// SetEnabled toggles benchmark logging (default: on).
func SetEnabled(on bool) {
	mu.Lock()
	enabled = on
	mu.Unlock()
}

func sinceStart() int64 {
	mu.RLock()
	start := runStart
	mu.RUnlock()
	if start.IsZero() {
		return 0
	}
	return time.Since(start).Milliseconds()
}

func write(kind string, impl string, kv ...string) {
	mu.RLock()
	on := enabled
	mu.RUnlock()
	if !on {
		return
	}

	ts := time.Now().UTC().Format(time.RFC3339Nano)
	elapsed := sinceStart()
	parts := []string{
		fmt.Sprintf("bench ts=%s elapsed_ms=%d impl=%s kind=%s", ts, elapsed, impl, kind),
	}
	parts = append(parts, kv...)
	log.New(out, "", 0).Println(strings.Join(parts, " "))
}

// RPC logs a client RPC with duration and outcome.
func RPC(impl, op, mode string, duration time.Duration, ok bool, kv ...string) {
	status := "ok"
	if !ok {
		status = "error"
	}
	fields := []string{
		fmt.Sprintf("op=%s", op),
		fmt.Sprintf("mode=%s", mode),
		fmt.Sprintf("duration_ms=%d", duration.Milliseconds()),
		fmt.Sprintf("status=%s", status),
	}
	fields = append(fields, kv...)
	write("rpc", impl, fields...)
}

// Task logs scheduler task lifecycle events.
func Task(impl, event, taskID, agent string, duration time.Duration, kv ...string) {
	fields := []string{
		fmt.Sprintf("event=%s", event),
		fmt.Sprintf("task=%s", taskID),
		fmt.Sprintf("agent=%s", agent),
	}
	if duration > 0 {
		fields = append(fields, fmt.Sprintf("duration_ms=%d", duration.Milliseconds()))
	}
	fields = append(fields, kv...)
	write("task", impl, fields...)
}

// TruncatePayload shortens long payload strings for log lines.
func TruncatePayload(payload string) string {
	if len(payload) <= 80 {
		return payload
	}
	return payload[:77] + "..."
}

// Coord logs coordination ops (context push, cancel, sync) from the scheduler.
func Coord(impl, event string, ok bool, duration time.Duration, kv ...string) {
	status := "ok"
	if !ok {
		status = "error"
	}
	fields := []string{
		fmt.Sprintf("event=%s", event),
		fmt.Sprintf("status=%s", status),
	}
	if duration > 0 {
		fields = append(fields, fmt.Sprintf("duration_ms=%d", duration.Milliseconds()))
	}
	fields = append(fields, kv...)
	write("coord", impl, fields...)
}
