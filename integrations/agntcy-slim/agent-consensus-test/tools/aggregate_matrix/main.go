// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// aggregate_matrix reads raw per-run matrix TSV rows and writes aggregated statistics.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/metrics"
	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/stats"
)

type groupKey struct {
	LatencyMs    int64
	Agents       int
	PayloadBytes int
	ThinkTimeMs  int64
	ScenarioName string
	Domain       string
	Implementation string
}

type metricStats struct {
	Mean     float64
	Variance float64
	StdDev   float64
	CILow    float64
	CIHigh   float64
	N        int
}

type aggregatedRow struct {
	groupKey
	RunCount     int
	SuccessCount int
	Wall         metricStats
	Emitted      metricStats
	Received     metricStats
	AvgProp      metricStats
	P95Prop      metricStats
}

func main() {
	inPath := flag.String("in", "./reports/matrix-raw.tsv", "raw matrix TSV with run_id rows")
	outPath := flag.String("out", "./reports/matrix-stats.tsv", "aggregated stats TSV output")
	flag.Parse()

	rows, err := readRawTSV(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read raw tsv: %v\n", err)
		os.Exit(1)
	}
	grouped := groupRows(rows)
	aggregated := aggregateGroups(grouped)
	if err := writeStatsTSV(*outPath, aggregated); err != nil {
		fmt.Fprintf(os.Stderr, "write stats tsv: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d aggregated rows)\n", *outPath, len(aggregated))
}

func readRawTSV(path string) ([]metrics.RunResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) <= 1 {
		return nil, fmt.Errorf("no data rows in %s", path)
	}
	col := columnMap(records[0])
	var out []metrics.RunResult
	for _, row := range records[1:] {
		r := metrics.RunResult{
			RunID:                 atoi(field(row, col, "run_id")),
			ScenarioName:          field(row, col, "scenario_name"),
			Domain:                field(row, col, "domain"),
			Implementation:        field(row, col, "implementation"),
			Agents:                atoi(field(row, col, "agents")),
			ThinkTimeMs:           atoi64(field(row, col, "think_time_ms")),
			PayloadBytes:          atoi(field(row, col, "payload_bytes")),
			LatencyMs:             atoi64(field(row, col, "latency_ms")),
			ConsensusWallMS:       atoi64(field(row, col, "consensus_wall_ms")),
			FindingsEmitted:       atoi(field(row, col, "findings_emitted")),
			FindingsReceivedTotal: atoi(field(row, col, "findings_received_total")),
			AvgPropagationMS:      atoi64(field(row, col, "avg_propagation_ms")),
			P95PropagationMS:      atoi64(field(row, col, "p95_propagation_ms")),
			Success:               field(row, col, "success") == "true",
		}
		out = append(out, r)
	}
	return out, nil
}

func groupRows(rows []metrics.RunResult) map[groupKey][]metrics.RunResult {
	grouped := map[groupKey][]metrics.RunResult{}
	for _, r := range rows {
		k := groupKey{
			LatencyMs:      r.LatencyMs,
			Agents:         r.Agents,
			PayloadBytes:   r.PayloadBytes,
			ThinkTimeMs:    r.ThinkTimeMs,
			ScenarioName:   r.ScenarioName,
			Domain:         r.Domain,
			Implementation: r.Implementation,
		}
		grouped[k] = append(grouped[k], r)
	}
	return grouped
}

func aggregateGroups(grouped map[groupKey][]metrics.RunResult) []aggregatedRow {
	keys := make([]groupKey, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].LatencyMs != keys[j].LatencyMs {
			return keys[i].LatencyMs < keys[j].LatencyMs
		}
		if keys[i].Agents != keys[j].Agents {
			return keys[i].Agents < keys[j].Agents
		}
		if keys[i].PayloadBytes != keys[j].PayloadBytes {
			return keys[i].PayloadBytes < keys[j].PayloadBytes
		}
		return keys[i].Implementation < keys[j].Implementation
	})

	out := make([]aggregatedRow, 0, len(keys))
	for _, k := range keys {
		rows := grouped[k]
		success := 0
		for _, r := range rows {
			if r.Success {
				success++
			}
		}
		out = append(out, aggregatedRow{
			groupKey:     k,
			RunCount:     len(rows),
			SuccessCount: success,
			Wall:         metricStatsFrom(rows, func(r metrics.RunResult) (float64, bool) {
				return float64(r.ConsensusWallMS), r.Success
			}),
			Emitted: metricStatsFrom(rows, func(r metrics.RunResult) (float64, bool) {
				return float64(r.FindingsEmitted), r.Success
			}),
			Received: metricStatsFrom(rows, func(r metrics.RunResult) (float64, bool) {
				return float64(r.FindingsReceivedTotal), r.Success
			}),
			AvgProp: metricStatsFrom(rows, func(r metrics.RunResult) (float64, bool) {
				return float64(r.AvgPropagationMS), r.Success && r.AvgPropagationMS > 0
			}),
			P95Prop: metricStatsFrom(rows, func(r metrics.RunResult) (float64, bool) {
				return float64(r.P95PropagationMS), r.Success && r.P95PropagationMS > 0
			}),
		})
	}
	return out
}

func metricStatsFrom(rows []metrics.RunResult, pick func(metrics.RunResult) (float64, bool)) metricStats {
	values := make([]float64, 0, len(rows))
	for _, r := range rows {
		v, ok := pick(r)
		if ok {
			values = append(values, v)
		}
	}
	s := stats.ComputeSampleStats(values)
	return metricStats{
		Mean:     s.Mean,
		Variance: s.Variance,
		StdDev:   s.StdDev,
		CILow:    s.CILow,
		CIHigh:   s.CIHigh,
		N:        s.Count,
	}
}

func writeStatsTSV(path string, rows []aggregatedRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	w.Comma = '\t'
	header := []string{
		"scenario_name", "domain", "implementation", "agents", "think_time_ms", "payload_bytes", "latency_ms",
		"run_count", "success_count",
		"consensus_wall_mean", "consensus_wall_variance", "consensus_wall_stddev", "consensus_wall_ci_low", "consensus_wall_ci_high", "consensus_wall_n",
		"findings_emitted_mean", "findings_emitted_variance", "findings_emitted_stddev", "findings_emitted_ci_low", "findings_emitted_ci_high", "findings_emitted_n",
		"findings_received_mean", "findings_received_variance", "findings_received_stddev", "findings_received_ci_low", "findings_received_ci_high", "findings_received_n",
		"avg_propagation_mean", "avg_propagation_variance", "avg_propagation_stddev", "avg_propagation_ci_low", "avg_propagation_ci_high", "avg_propagation_n",
		"p95_propagation_mean", "p95_propagation_variance", "p95_propagation_stddev", "p95_propagation_ci_low", "p95_propagation_ci_high", "p95_propagation_n",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, r := range rows {
		record := []string{
			r.ScenarioName, r.Domain, r.Implementation,
			strconv.Itoa(r.Agents), strconv.FormatInt(r.ThinkTimeMs, 10),
			strconv.Itoa(r.PayloadBytes), strconv.FormatInt(r.LatencyMs, 10),
			strconv.Itoa(r.RunCount), strconv.Itoa(r.SuccessCount),
		}
		record = append(record, formatMetricRow(r.Wall)...)
		record = append(record, formatMetricRow(r.Emitted)...)
		record = append(record, formatMetricRow(r.Received)...)
		record = append(record, formatMetricRow(r.AvgProp)...)
		record = append(record, formatMetricRow(r.P95Prop)...)
		if err := w.Write(record); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func formatMetricRow(m metricStats) []string {
	return []string{
		formatFloat(m.Mean),
		formatFloat(m.Variance),
		formatFloat(m.StdDev),
		formatFloat(m.CILow),
		formatFloat(m.CIHigh),
		strconv.Itoa(m.N),
	}
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func columnMap(header []string) map[string]int {
	out := make(map[string]int, len(header))
	for i, h := range header {
		out[h] = i
	}
	return out
}

func field(row []string, col map[string]int, name string) string {
	i, ok := col[name]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}

func atoi(v string) int {
	n, _ := strconv.Atoi(v)
	return n
}

func atoi64(v string) int64 {
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}
