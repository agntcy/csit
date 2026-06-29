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
	ImplSLIM = "slim-group-session"
	ImplA2A  = "a2a-relay-stream"
)

var (
	mu       sync.RWMutex
	runStart time.Time
	enabled  = true
	out      io.Writer = os.Stderr
)

func SetRunStart(t time.Time) {
	mu.Lock()
	runStart = t
	mu.Unlock()
}

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

func write(kind, impl string, kv ...string) {
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

func Finding(impl, event string, agentIndex int, findingID int64, kv ...string) {
	fields := []string{
		fmt.Sprintf("event=%s", event),
		fmt.Sprintf("agent=%d", agentIndex),
		fmt.Sprintf("finding_id=%d", findingID),
	}
	fields = append(fields, kv...)
	write("finding", impl, fields...)
}
