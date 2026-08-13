// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/metrics"
)

func runMatrixCell(cfg matrixConfig) {
	ensureScenario(cfg)
	resetCellTSV(cfg)

	for repeat := 1; repeat <= cfg.Repeats; repeat++ {
		runP2P(cfg, repeat)
		logMatrixResult(cfg, repeat, "p2p-relay-stream")

		runSLIM(cfg, repeat)
		logMatrixResult(cfg, repeat, "slim-group-session")
	}
}

func logMatrixResult(cfg matrixConfig, repeat int, impl string) {
	row, ok := latestRunResult(cfg.CellTSV, repeat, impl)
	if !ok {
		_ = writeProgressLine(
			"CONSENSUS_MATRIX_RESULT latency_ms=%d agents=%d payload_bytes=%d repeat=%d implementation=%s success=false consensus_wall_ms=0 error=missing_row",
			cfg.LatencyMs, cfg.Agents, cfg.PayloadBytes, repeat, impl,
		)
		return
	}
	errMsg := row.Error
	if errMsg == "" && !row.Success {
		errMsg = "failed"
	}
	_ = writeProgressLine(
		"CONSENSUS_MATRIX_RESULT latency_ms=%d agents=%d payload_bytes=%d repeat=%d implementation=%s success=%t consensus_wall_ms=%d findings_emitted=%d findings_received_total=%d error=%s",
		cfg.LatencyMs,
		cfg.Agents,
		cfg.PayloadBytes,
		repeat,
		impl,
		row.Success,
		row.ConsensusWallMS,
		row.FindingsEmitted,
		row.FindingsReceivedTotal,
		errMsg,
	)
}

func latestRunResult(path string, runID int, impl string) (metrics.RunResult, bool) {
	file, err := os.Open(path)
	if err != nil {
		return metrics.RunResult{}, false
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	records, err := reader.ReadAll()
	if err != nil || len(records) <= 1 {
		return metrics.RunResult{}, false
	}
	col := map[string]int{}
	for i, h := range records[0] {
		col[h] = i
	}

	var found metrics.RunResult
	ok := false
	for _, row := range records[1:] {
		if field(row, col, "run_id") != fmt.Sprintf("%d", runID) {
			continue
		}
		if field(row, col, "implementation") != impl {
			continue
		}
		found = metrics.RunResult{
			RunID:                 runID,
			ScenarioName:          field(row, col, "scenario_name"),
			Implementation:        impl,
			ConsensusWallMS:       parseInt64(field(row, col, "consensus_wall_ms")),
			FindingsEmitted:       parseInt(field(row, col, "findings_emitted")),
			FindingsReceivedTotal: parseInt(field(row, col, "findings_received_total")),
			Success:               field(row, col, "success") == "true",
			Error:                 field(row, col, "error"),
		}
		ok = true
	}
	return found, ok
}

func field(row []string, col map[string]int, name string) string {
	i, ok := col[name]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}

func parseInt(v string) int {
	var n int
	fmt.Sscanf(v, "%d", &n)
	return n
}

func parseInt64(v string) int64 {
	var n int64
	fmt.Sscanf(v, "%d", &n)
	return n
}
