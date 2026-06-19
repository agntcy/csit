// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

type RunResult struct {
	ScenarioName          string  `json:"scenario_name"`
	Domain                string  `json:"domain"`
	Implementation        string  `json:"implementation"`
	Agents                int     `json:"agents"`
	ThinkTimeMs           int64   `json:"think_time_ms"`
	ConsensusWallMS       int64   `json:"consensus_wall_ms"`
	ConsensusRound        int     `json:"consensus_round"`
	FindingsEmitted       int     `json:"findings_emitted"`
	FindingsReceivedTotal int     `json:"findings_received_total"`
	AvgPropagationMS      int64   `json:"avg_propagation_ms"`
	P95PropagationMS      int64   `json:"p95_propagation_ms"`
	LastAgentConvergeMS   int64   `json:"last_agent_converge_ms"`
	CoordFanoutMS         int64   `json:"coord_fanout_ms"`
	StreamRPCCount        int     `json:"stream_rpc_count"`
	UnicastRPCCount       int     `json:"unicast_rpc_count"`
	Success               bool    `json:"success"`
	Error                 string  `json:"error,omitempty"`
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
			"scenario_name", "domain", "implementation", "agents", "think_time_ms",
			"consensus_wall_ms", "consensus_round", "findings_emitted", "findings_received_total",
			"avg_propagation_ms", "p95_propagation_ms", "last_agent_converge_ms",
			"coord_fanout_ms", "stream_rpc_count", "unicast_rpc_count",
			"success", "error",
		}); err != nil {
			return err
		}
	}
	if err := w.Write([]string{
		result.ScenarioName,
		result.Domain,
		result.Implementation,
		strconv.Itoa(result.Agents),
		strconv.FormatInt(result.ThinkTimeMs, 10),
		strconv.FormatInt(result.ConsensusWallMS, 10),
		strconv.Itoa(result.ConsensusRound),
		strconv.Itoa(result.FindingsEmitted),
		strconv.Itoa(result.FindingsReceivedTotal),
		strconv.FormatInt(result.AvgPropagationMS, 10),
		strconv.FormatInt(result.P95PropagationMS, 10),
		strconv.FormatInt(result.LastAgentConvergeMS, 10),
		strconv.FormatInt(result.CoordFanoutMS, 10),
		strconv.Itoa(result.StreamRPCCount),
		strconv.Itoa(result.UnicastRPCCount),
		strconv.FormatBool(result.Success),
		result.Error,
	}); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func AggregatePropagation(durations []int64) (avg, p95 int64) {
	if len(durations) == 0 {
		return 0, 0
	}
	var sum int64
	for _, d := range durations {
		sum += d
	}
	avg = sum / int64(len(durations))
	sorted := append([]int64(nil), durations...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	idx := int(float64(len(sorted)-1) * 0.95)
	p95 = sorted[idx]
	return avg, p95
}
