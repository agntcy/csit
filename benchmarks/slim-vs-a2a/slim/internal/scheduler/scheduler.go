// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/plan"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/client"
)

type taskState int

const (
	statePending taskState = iota
	stateRunning
	stateCompleted
	stateFailed
	stateTimedOut
	stateCancelled
)

type Scheduler struct {
	plan   *plan.ExecutionPlan
	client *client.Client
}

func New(p *plan.ExecutionPlan, cli *client.Client) *Scheduler {
	return &Scheduler{plan: p, client: cli}
}

func (s *Scheduler) Run(ctx context.Context) metrics.RunResult {
	start := time.Now()
	benchlog.SetRunStart(start)
	result := metrics.RunResult{
		PlanName:       s.plan.Metadata.Name,
		Domain:         s.plan.Metadata.Domain,
		Implementation: s.client.Implementation(),
		Agents:         len(s.plan.Agents),
		Tasks:          len(s.plan.Tasks),
	}

	states := map[string]taskState{}
	contextFired := map[string]bool{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	var contextPushMS int64
	var syncBarrierMS int64
	var cancelPropagationMS int64
	var obsoleteCompleted int
	abort := false

	for _, t := range s.plan.Tasks {
		states[t.ID] = statePending
	}

	cancelTasks := func(taskIDs []string) {
		if len(taskIDs) == 0 {
			return
		}
		agentSet := map[string]struct{}{}
		for _, id := range taskIDs {
			task, ok := s.plan.TaskByID(id)
			if !ok {
				continue
			}
			agentSet[task.Agent] = struct{}{}
		}
		agentIDs := make([]string, 0, len(agentSet))
		for id := range agentSet {
			agentIDs = append(agentIDs, id)
		}
		startCancel := time.Now()
		err := s.client.CancelTasks(ctx, agentIDs, taskIDs)
		duration := time.Since(startCancel)
		cancelPropagationMS += duration.Milliseconds()
		kv := []string{
			fmt.Sprintf("agents=%d", len(agentIDs)),
			fmt.Sprintf("tasks=%d", len(taskIDs)),
		}
		if err != nil {
			kv = append(kv, fmt.Sprintf("err=%v", err))
		}
		benchlog.Coord(benchlog.ImplSLIM, "cancel", err == nil, duration, kv...)
	}

	cancelDownstream := func(root string) {
		downstream := plan.DownstreamTasks(s.plan.Tasks, root)
		var ids []string
		mu.Lock()
		for id := range downstream {
			switch states[id] {
			case statePending, stateRunning:
				states[id] = stateCancelled
				result.TasksCancelled++
				ids = append(ids, id)
			}
		}
		mu.Unlock()
		cancelTasks(ids)
	}

	handleFailure := func(taskID string, timedOut bool) {
		mu.Lock()
		if timedOut {
			states[taskID] = stateTimedOut
			result.TasksTimedOut++
		} else {
			states[taskID] = stateFailed
			result.TasksFailed++
		}
		abort = true
		mu.Unlock()
		agents := affectedAgents([]string{taskID}, s.plan)
		if err := s.client.NotifyFailure(ctx, agents, taskID); err != nil {
			benchlog.Coord(benchlog.ImplSLIM, "notify_failure", false, 0,
				fmt.Sprintf("task=%s", taskID),
				fmt.Sprintf("agents=%d", len(agents)),
				fmt.Sprintf("err=%v", err),
			)
		}
		cancelTasks([]string{taskID})
		cancelDownstream(taskID)
	}

	applyContext := func(taskID string) {
		mu.Lock()
		if contextFired[taskID] {
			mu.Unlock()
			return
		}
		contextFired[taskID] = true
		mu.Unlock()

		for _, cu := range s.plan.ContextUpdatesAfter(taskID) {
			d, err := s.client.PushContext(ctx, cu.TargetAgents, cu.Payload)
			contextPushMS += d.Milliseconds()
			kv := []string{
				fmt.Sprintf("after_task=%s", taskID),
				fmt.Sprintf("agents=%d", len(cu.TargetAgents)),
				fmt.Sprintf("payload=%q", benchlog.TruncatePayload(cu.Payload)),
			}
			if err != nil {
				kv = append(kv, fmt.Sprintf("err=%v", err))
			}
			benchlog.Coord(benchlog.ImplSLIM, "context_push", err == nil, d, kv...)

			if strings.Contains(cu.Payload, "sync=") || strings.Contains(cu.Payload, "phase=") {
				syncD, syncErr := s.client.SyncPhase(ctx, cu.TargetAgents, cu.Payload)
				if syncErr == nil {
					syncBarrierMS += syncD.Milliseconds()
				}
				syncKV := []string{
					fmt.Sprintf("after_task=%s", taskID),
					fmt.Sprintf("agents=%d", len(cu.TargetAgents)),
				}
				if syncErr != nil {
					syncKV = append(syncKV, fmt.Sprintf("err=%v", syncErr))
				}
				benchlog.Coord(benchlog.ImplSLIM, "sync_phase", syncErr == nil, syncD, syncKV...)
			}
			if strings.Contains(strings.ToLower(cu.Payload), "cancel") {
				var toCancel []string
				mu.Lock()
				for _, t := range s.plan.Tasks {
					if states[t.ID] != stateRunning {
						continue
					}
					if shouldCancelByContext(t, cu.Payload) {
						states[t.ID] = stateCancelled
						result.TasksCancelled++
						toCancel = append(toCancel, t.ID)
						obsoleteCompleted++
					}
				}
				mu.Unlock()
				cancelTasks(toCancel)
			}
		}
	}

	launch := func(task plan.Task) {
		wg.Add(1)
		go func(task plan.Task) {
			defer wg.Done()
			taskStart := time.Now()
			benchlog.Task(benchlog.ImplSLIM, "started", task.ID, task.Agent, 0)
			resp, err := s.client.ExecuteTask(ctx, task.Agent, task)
			taskDuration := time.Since(taskStart)
			mu.Lock()
			if abort && states[task.ID] == stateRunning {
				states[task.ID] = stateCancelled
				result.TasksCancelled++
				mu.Unlock()
				benchlog.Task(benchlog.ImplSLIM, "cancelled", task.ID, task.Agent, taskDuration)
				return
			}
			mu.Unlock()

			if err != nil {
				benchlog.Task(benchlog.ImplSLIM, "failed", task.ID, task.Agent, taskDuration,
					fmt.Sprintf("err=%v", err))
				handleFailure(task.ID, false)
				return
			}
			if !resp.OK {
				benchlog.Task(benchlog.ImplSLIM, "failed", task.ID, task.Agent, taskDuration,
					fmt.Sprintf("resp_error=%s", resp.Error))
				handleFailure(task.ID, resp.Error == "timeout")
				return
			}

			mu.Lock()
			states[task.ID] = stateCompleted
			result.TasksCompleted++
			mu.Unlock()
			benchlog.Task(benchlog.ImplSLIM, "completed", task.ID, task.Agent, taskDuration)
			applyContext(task.ID)
		}(task)
	}

	for {
		mu.Lock()
		if abort {
			mu.Unlock()
			break
		}
		ready := readyTasks(s.plan.Tasks, states)
		for _, task := range ready {
			states[task.ID] = stateRunning
			launch(task)
		}
		pending := countStates(states, statePending)
		running := countStates(states, stateRunning)
		mu.Unlock()

		if len(ready) == 0 && pending == 0 && running == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	wg.Wait()
	stats := s.client.Stats()
	coordTimeMS := contextPushMS + syncBarrierMS + cancelPropagationMS
	result.TotalWallClockMS = time.Since(start).Milliseconds()
	result.ContextPushMS = contextPushMS
	result.SyncBarrierMS = syncBarrierMS
	result.CancelPropagationMS = cancelPropagationMS
	result.ObsoleteTasksCompleted = obsoleteCompleted
	result.ExecuteRPCCount = stats.ExecuteRPCCount
	result.SequentialRPCCount = stats.SequentialRPCCount
	result.MulticastRPCCount = stats.MulticastRPCCount
	result.MakespanMS = result.TotalWallClockMS
	result.Success = result.TasksFailed == 0 && result.TasksTimedOut == 0
	if s.client.CoordStats() != nil {
		s.client.CoordStats().ApplyToRunResult(&result, coordTimeMS)
	}
	if !result.Success && result.Error == "" {
		result.Error = fmt.Sprintf("failed=%d timed_out=%d cancelled=%d", result.TasksFailed, result.TasksTimedOut, result.TasksCancelled)
	}
	benchlog.Coord(benchlog.ImplSLIM, "run_finished", result.Success, time.Since(start),
		fmt.Sprintf("tasks_completed=%d", result.TasksCompleted),
		fmt.Sprintf("tasks_failed=%d", result.TasksFailed),
		fmt.Sprintf("tasks_cancelled=%d", result.TasksCancelled),
		fmt.Sprintf("context_push_ms=%d", contextPushMS),
		fmt.Sprintf("execute_rpcs=%d", stats.ExecuteRPCCount),
		fmt.Sprintf("multicast_rpcs=%d", stats.MulticastRPCCount),
	)
	return result
}

func readyTasks(tasks []plan.Task, states map[string]taskState) []plan.Task {
	var ready []plan.Task
	for _, task := range tasks {
		if states[task.ID] != statePending {
			continue
		}
		ok := true
		for _, dep := range task.DependsOn {
			if states[dep] != stateCompleted {
				ok = false
				break
			}
		}
		if ok {
			ready = append(ready, task)
		}
	}
	return ready
}

func countStates(states map[string]taskState, target taskState) int {
	n := 0
	for _, st := range states {
		if st == target {
			n++
		}
	}
	return n
}

func affectedAgents(taskIDs []string, p *plan.ExecutionPlan) []string {
	set := map[string]struct{}{}
	for _, id := range taskIDs {
		task, ok := p.TaskByID(id)
		if !ok {
			continue
		}
		set[task.Agent] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

func shouldCancelByContext(task plan.Task, payload string) bool {
	payload = strings.ToLower(payload)
	if strings.Contains(payload, "pharmacy") && strings.Contains(task.ID, "pharmacy") {
		return true
	}
	if strings.Contains(payload, "detour") && strings.Contains(task.ID, "detour") {
		return true
	}
	if strings.Contains(payload, "node-drain") && strings.Contains(task.ID, "node-drain") {
		return true
	}
	if strings.Contains(payload, "mesh-rollback") && strings.Contains(task.ID, "rollback") {
		return true
	}
	return false
}
