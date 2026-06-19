// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/protocol"
)

type Engine struct {
	SlimName string
	mu       sync.Mutex
	cancels  map[string]context.CancelFunc
}

func New(slimName string) *Engine {
	return &Engine{SlimName: slimName, cancels: map[string]context.CancelFunc{}}
}

func (e *Engine) Handle(ctx context.Context, req protocol.Request) protocol.Response {
	if len(req.TargetSlimNames) > 0 && !req.Targets(e.SlimName) {
		return protocol.Response{OK: true, SlimName: e.SlimName}
	}

	switch req.Op {
	case protocol.OpExecute:
		resp := e.execute(ctx, req)
		resp.SlimName = e.SlimName
		return resp
	case protocol.OpCancel:
		resp := e.cancel(req)
		resp.SlimName = e.SlimName
		return resp
	case protocol.OpContext, protocol.OpSync:
		return protocol.Response{OK: true, SlimName: e.SlimName}
	default:
		return protocol.Response{OK: false, SlimName: e.SlimName, Error: "unknown op"}
	}
}

func (e *Engine) execute(parent context.Context, req protocol.Request) protocol.Response {
	if req.InjectFailure {
		return protocol.Response{OK: false, TaskID: req.TaskID, Error: "injected failure"}
	}

	ctx, cancel := context.WithCancel(parent)
	e.mu.Lock()
	e.cancels[req.TaskID] = cancel
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.cancels, req.TaskID)
		e.mu.Unlock()
		cancel()
	}()

	deadline := time.Duration(req.MaxCompletionTimeSec * float64(time.Second))
	if deadline <= 0 {
		deadline = time.Duration(req.CompletionTimeSec*float64(time.Second)) + time.Second
	}
	timer := time.NewTimer(deadline)
	defer timer.Stop()

	sleep := time.Duration(req.CompletionTimeSec * float64(time.Second))
	start := time.Now()
	select {
	case <-time.After(sleep):
	case <-ctx.Done():
		return protocol.Response{OK: false, TaskID: req.TaskID, Error: "cancelled"}
	case <-timer.C:
		return protocol.Response{OK: false, TaskID: req.TaskID, Error: "timeout"}
	}

	return protocol.Response{
		OK:         true,
		TaskID:     req.TaskID,
		Output:     req.Output,
		ElapsedSec: time.Since(start).Seconds(),
	}
}

func (e *Engine) cancel(req protocol.Request) protocol.Response {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, id := range req.TaskIDs {
		if cancel, ok := e.cancels[id]; ok {
			cancel()
		}
	}
	return protocol.Response{OK: true}
}

func (e *Engine) ObsoleteFromPayload(payload string) bool {
	return strings.Contains(strings.ToLower(payload), "cancel")
}
