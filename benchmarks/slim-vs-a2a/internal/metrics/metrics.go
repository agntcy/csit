// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type RunResult struct {
	PlanName               string  `json:"plan_name"`
	Domain                 string  `json:"domain"`
	Implementation         string  `json:"implementation"`
	Agents                 int     `json:"agents"`
	Tasks                  int     `json:"tasks"`
	TotalWallClockMS       int64   `json:"total_wall_clock_ms"`
	TasksCompleted         int     `json:"tasks_completed"`
	TasksFailed            int     `json:"tasks_failed"`
	TasksTimedOut          int     `json:"tasks_timed_out"`
	TasksCancelled         int     `json:"tasks_cancelled"`
	ObsoleteTasksCompleted int     `json:"obsolete_tasks_completed"`
	RetriesAttempted       int     `json:"retries_attempted"`
	RetriesSucceeded       int     `json:"retries_succeeded"`
	ContextPushMS          int64   `json:"context_push_ms"`
	SyncBarrierMS          int64   `json:"sync_barrier_ms"`
	CancelPropagationMS    int64   `json:"cancel_propagation_ms"`
	ExecuteRPCCount        int     `json:"execute_rpc_count"`
	MulticastRPCCount      int     `json:"multicast_rpc_count"`
	SequentialRPCCount       int     `json:"sequential_rpc_count"`
	MakespanMS             int64   `json:"makespan_ms"`
	ContextPushP95MS       int64   `json:"context_push_p95_ms"`
	ContextPushOps         int     `json:"context_push_ops"`
	CoordMissingResponses  int     `json:"coord_missing_responses"`
	CoordDeadlineMisses    int     `json:"coord_deadline_misses"`
	CoordBytesSent         int64   `json:"coord_bytes_sent"`
	CoordTimeSharePct      float64 `json:"coord_time_share_pct"`
	RoundBudgetMS          int64   `json:"round_budget_ms"`
	Success                bool    `json:"success"`
	Error                  string  `json:"error,omitempty"`
}

func WriteJSON(path string, result RunResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func AppendTSV(path string, result RunResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	writeHeader := true
	if info, err := os.Stat(path); err == nil {
		writeHeader = info.Size() == 0
	} else if !os.IsNotExist(err) {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	w.Comma = '\t'
	if writeHeader {
		if err := w.Write([]string{
			"plan_name", "domain", "implementation", "agents", "tasks",
			"total_wall_clock_ms", "tasks_completed", "tasks_failed", "tasks_timed_out",
			"tasks_cancelled", "obsolete_tasks_completed", "retries_attempted", "retries_succeeded",
			"context_push_ms", "sync_barrier_ms", "cancel_propagation_ms",
			"execute_rpc_count", "multicast_rpc_count", "sequential_rpc_count", "makespan_ms",
			"context_push_p95_ms", "context_push_ops", "coord_missing_responses", "coord_deadline_misses",
			"coord_bytes_sent", "coord_time_share_pct", "round_budget_ms",
			"success", "error",
		}); err != nil {
			return err
		}
	}
	if err := w.Write([]string{
		result.PlanName,
		result.Domain,
		result.Implementation,
		strconv.Itoa(result.Agents),
		strconv.Itoa(result.Tasks),
		strconv.FormatInt(result.TotalWallClockMS, 10),
		strconv.Itoa(result.TasksCompleted),
		strconv.Itoa(result.TasksFailed),
		strconv.Itoa(result.TasksTimedOut),
		strconv.Itoa(result.TasksCancelled),
		strconv.Itoa(result.ObsoleteTasksCompleted),
		strconv.Itoa(result.RetriesAttempted),
		strconv.Itoa(result.RetriesSucceeded),
		strconv.FormatInt(result.ContextPushMS, 10),
		strconv.FormatInt(result.SyncBarrierMS, 10),
		strconv.FormatInt(result.CancelPropagationMS, 10),
		strconv.Itoa(result.ExecuteRPCCount),
		strconv.Itoa(result.MulticastRPCCount),
		strconv.Itoa(result.SequentialRPCCount),
		strconv.FormatInt(result.MakespanMS, 10),
		strconv.FormatInt(result.ContextPushP95MS, 10),
		strconv.Itoa(result.ContextPushOps),
		strconv.Itoa(result.CoordMissingResponses),
		strconv.Itoa(result.CoordDeadlineMisses),
		strconv.FormatInt(result.CoordBytesSent, 10),
		strconv.FormatFloat(result.CoordTimeSharePct, 'f', 1, 64),
		strconv.FormatInt(result.RoundBudgetMS, 10),
		strconv.FormatBool(result.Success),
		result.Error,
	}); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func WriteSummaryMarkdown(path string, results []RunResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b []byte
	b = append(b, "# SLIM vs A2A Comparison Summary\n\n"...)
	b = append(b, "| Plan | Implementation | Wall (ms) | Context push (ms) | Cancel prop (ms) | Completed | Failed | Cancelled | Success |\n"...)
	b = append(b, "| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n"...)
	for _, r := range results {
		b = append(b, fmt.Sprintf("| %s | %s | %d | %d | %d | %d | %d | %d | %t |\n",
			r.PlanName, r.Implementation, r.TotalWallClockMS, r.ContextPushMS, r.CancelPropagationMS,
			r.TasksCompleted, r.TasksFailed, r.TasksCancelled, r.Success)...)
	}
	return os.WriteFile(path, b, 0o644)
}
