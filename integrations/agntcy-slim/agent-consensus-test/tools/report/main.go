// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/metrics"
	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/scenario"
)

type planMeta struct {
	Description string
	Order       int
}

type scenarioComparison struct {
	ScenarioName string
	Description  string
	Order        int
	P2P          metrics.RunResult
	SLIM         metrics.RunResult
	HasP2P       bool
	HasSLIM      bool
}

type metricStat struct {
	Mean   float64
	StdDev float64
	CILow  float64
	CIHigh float64
	N      int
}

type matrixCell struct {
	HasStats bool

	HasP2P      bool
	HasSLIM     bool
	P2PSuccess  bool
	SLIMSuccess bool

	P2PWall  int64
	SLIMWall int64
	P2PWallStat metricStat
	SLIMWallStat metricStat
	WallRatio string

	P2PEmitted  int
	SLIMEmitted int
	P2PEmittedStat metricStat
	SLIMEmittedStat metricStat
	EmittedRatio string

	P2PReceived  int
	SLIMReceived int
	P2PReceivedStat metricStat
	SLIMReceivedStat metricStat
	ReceivedRatio string

	P2PAvgProp  int64
	SLIMAvgProp int64
	P2PAvgPropStat metricStat
	SLIMAvgPropStat metricStat
	AvgPropRatio string

	P2PP95Prop  int64
	SLIMP95Prop int64
	P2PP95PropStat metricStat
	SLIMP95PropStat metricStat
	P95PropRatio string
}

type matrixTable struct {
	LatencyMs   int64
	ThinkTimeMs int64
	AgentRows   []int
	PayloadCols []int
	Cells       map[int]map[int]matrixCell
}

func main() {
	tsvPath := flag.String("tsv", "./reports/results.tsv", "comparison results tsv")
	sweepTSV := flag.String("sweep-tsv", "./reports/sweep.tsv", "optional sweep results tsv")
	matrixTSV := flag.String("matrix-tsv", "./reports/matrix.tsv", "optional matrix results tsv")
	scenarioDir := flag.String("scenario-dir", "./plans/sweeps", "directory of scenario yaml files (for per-plan descriptions)")
	matrixDir := flag.String("matrix-dir", "./plans/matrix", "directory of matrix scenario yaml files")
	matrixOnly := flag.Bool("matrix-only", false, "render matrix dashboard only (requires matrix-tsv)")
	output := flag.String("output", "./reports/index.html", "html dashboard output")
	flag.Parse()

	if *matrixOnly {
		matrices, err := loadMatrixTables(*matrixTSV)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read matrix tsv: %v\n", err)
			os.Exit(1)
		}
		if err := writeMatrixHTML(*output, matrices); err != nil {
			fmt.Fprintf(os.Stderr, "write html: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", *output)
		return
	}

	results, err := readTSV(*tsvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read tsv: %v\n", err)
		os.Exit(1)
	}
	planMetas := loadPlanMeta(*scenarioDir)
	comparisons := groupByScenario(results, planMetas)

	var sweepResults []metrics.RunResult
	if *sweepTSV != "" {
		if sr, err := readTSV(*sweepTSV); err == nil {
			sweepResults = sr
		}
	}

	var matrices []matrixTable
	if *matrixTSV != "" {
		if m, err := loadMatrixTables(*matrixTSV); err == nil {
			matrices = m
		}
	}
	_ = matrixDir // reserved for future per-cell descriptions from yaml

	if err := writeHTML(*output, comparisons, sweepResults, matrices); err != nil {
		fmt.Fprintf(os.Stderr, "write html: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", *output)
}

func readTSV(path string) ([]metrics.RunResult, error) {
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
		if len(row) < 19 {
			continue
		}
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
			ConsensusRound:        atoi(field(row, col, "consensus_round")),
			FindingsEmitted:       atoi(field(row, col, "findings_emitted")),
			FindingsReceivedTotal: atoi(field(row, col, "findings_received_total")),
			AvgPropagationMS:      atoi64(field(row, col, "avg_propagation_ms")),
			P95PropagationMS:      atoi64(field(row, col, "p95_propagation_ms")),
			LastAgentConvergeMS:   atoi64(field(row, col, "last_agent_converge_ms")),
			CoordFanoutMS:         atoi64(field(row, col, "coord_fanout_ms")),
			StreamRPCCount:        atoi(field(row, col, "stream_rpc_count")),
			Epochs:                atoi(field(row, col, "epochs")),
			EpochsSucceeded:       atoi(field(row, col, "epochs_succeeded")),
			EpochsFailed:          atoi(field(row, col, "epochs_failed")),
			Success:               field(row, col, "success") == "true",
			Error:                 field(row, col, "error"),
		}
		out = append(out, r)
	}
	return out, nil
}

func loadMatrixTables(path string) ([]matrixTable, error) {
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
	if _, ok := columnMap(records[0])["consensus_wall_mean"]; ok {
		return buildMatrixTablesFromStats(records)
	}
	raw, err := readTSV(path)
	if err != nil {
		return nil, err
	}
	return buildMatrixTables(raw), nil
}

type statsRow struct {
	LatencyMs      int64
	Agents         int
	PayloadBytes   int
	ThinkTimeMs    int64
	Implementation string
	SuccessCount   int
	Wall           metricStat
	Emitted        metricStat
	Received       metricStat
	AvgProp        metricStat
	P95Prop        metricStat
}

func buildMatrixTablesFromStats(records [][]string) ([]matrixTable, error) {
	col := columnMap(records[0])
	byKey := map[cellKey]cellPairStats{}
	latencyThink := map[int64]int64{}

	for _, row := range records[1:] {
		sr := parseStatsRow(row, col)
		k := cellKey{
			LatencyMs:    sr.LatencyMs,
			Agents:       sr.Agents,
			PayloadBytes: sr.PayloadBytes,
			ThinkTimeMs:  sr.ThinkTimeMs,
		}
		p := byKey[k]
		is := implStats{
			Has:          true,
			Success:      sr.SuccessCount > 0,
			Wall:         sr.Wall,
			Emitted:      sr.Emitted,
			Received:     sr.Received,
			AvgProp:      sr.AvgProp,
			P95Prop:      sr.P95Prop,
		}
		switch sr.Implementation {
		case "p2p-relay-stream":
			p.P2P = is
		case "slim-group-session":
			p.SLIM = is
		}
		byKey[k] = p
		latencyThink[sr.LatencyMs] = sr.ThinkTimeMs
	}

	latencies := sortedInt64Keys(latencyThink)
	var tables []matrixTable
	for _, lat := range latencies {
		agentSet := map[int]struct{}{}
		payloadSet := map[int]struct{}{}
		for k := range byKey {
			if k.LatencyMs != lat {
				continue
			}
			agentSet[k.Agents] = struct{}{}
			payloadSet[k.PayloadBytes] = struct{}{}
		}
		tbl := matrixTable{
			LatencyMs:   lat,
			ThinkTimeMs: latencyThink[lat],
			AgentRows:   sortedIntKeys(agentSet),
			PayloadCols: sortedIntKeys(payloadSet),
			Cells:       map[int]map[int]matrixCell{},
		}
		for _, agents := range tbl.AgentRows {
			tbl.Cells[agents] = map[int]matrixCell{}
			for _, payload := range tbl.PayloadCols {
				k := cellKey{
					LatencyMs:    lat,
					Agents:       agents,
					PayloadBytes: payload,
					ThinkTimeMs:  tbl.ThinkTimeMs,
				}
				tbl.Cells[agents][payload] = mergeMatrixCellStats(byKey[k])
			}
		}
		tables = append(tables, tbl)
	}
	return tables, nil
}

type implStats struct {
	Has     bool
	Success bool
	Wall    metricStat
	Emitted metricStat
	Received metricStat
	AvgProp metricStat
	P95Prop metricStat
}

type cellPairStats struct {
	P2P  implStats
	SLIM implStats
}

func parseStatsRow(row []string, col map[string]int) statsRow {
	return statsRow{
		LatencyMs:      atoi64(field(row, col, "latency_ms")),
		Agents:         atoi(field(row, col, "agents")),
		PayloadBytes:   atoi(field(row, col, "payload_bytes")),
		ThinkTimeMs:    atoi64(field(row, col, "think_time_ms")),
		Implementation: field(row, col, "implementation"),
		SuccessCount:   atoi(field(row, col, "success_count")),
		Wall:           parseMetricStat(row, col, "consensus_wall"),
		Emitted:        parseMetricStat(row, col, "findings_emitted"),
		Received:       parseMetricStat(row, col, "findings_received"),
		AvgProp:        parseMetricStat(row, col, "avg_propagation"),
		P95Prop:        parseMetricStat(row, col, "p95_propagation"),
	}
}

func parseMetricStat(row []string, col map[string]int, prefix string) metricStat {
	return metricStat{
		Mean:   atof(field(row, col, prefix+"_mean")),
		StdDev: atof(field(row, col, prefix+"_stddev")),
		CILow:  atof(field(row, col, prefix+"_ci_low")),
		CIHigh: atof(field(row, col, prefix+"_ci_high")),
		N:      atoi(field(row, col, prefix+"_n")),
	}
}

func mergeMatrixCellStats(p cellPairStats) matrixCell {
	c := matrixCell{HasStats: true}
	if p.P2P.Has {
		c.HasP2P = true
		c.P2PSuccess = p.P2P.Success
		c.P2PWallStat = p.P2P.Wall
		c.P2PWall = int64(p.P2P.Wall.Mean)
		c.P2PEmittedStat = p.P2P.Emitted
		c.P2PEmitted = int(p.P2P.Emitted.Mean)
		c.P2PReceivedStat = p.P2P.Received
		c.P2PReceived = int(p.P2P.Received.Mean)
		c.P2PAvgPropStat = p.P2P.AvgProp
		c.P2PAvgProp = int64(p.P2P.AvgProp.Mean)
		c.P2PP95PropStat = p.P2P.P95Prop
		c.P2PP95Prop = int64(p.P2P.P95Prop.Mean)
	}
	if p.SLIM.Has {
		c.HasSLIM = true
		c.SLIMSuccess = p.SLIM.Success
		c.SLIMWallStat = p.SLIM.Wall
		c.SLIMWall = int64(p.SLIM.Wall.Mean)
		c.SLIMEmittedStat = p.SLIM.Emitted
		c.SLIMEmitted = int(p.SLIM.Emitted.Mean)
		c.SLIMReceivedStat = p.SLIM.Received
		c.SLIMReceived = int(p.SLIM.Received.Mean)
		c.SLIMAvgPropStat = p.SLIM.AvgProp
		c.SLIMAvgProp = int64(p.SLIM.AvgProp.Mean)
		c.SLIMP95PropStat = p.SLIM.P95Prop
		c.SLIMP95Prop = int64(p.SLIM.P95Prop.Mean)
	}
	c.WallRatio = ratioFloat(c.P2PWallStat.Mean, c.SLIMWallStat.Mean, c.P2PSuccess, c.SLIMSuccess)
	c.EmittedRatio = ratioFloat(c.P2PEmittedStat.Mean, c.SLIMEmittedStat.Mean, c.HasP2P, c.HasSLIM)
	c.ReceivedRatio = ratioFloat(c.P2PReceivedStat.Mean, c.SLIMReceivedStat.Mean, c.HasP2P, c.HasSLIM)
	c.AvgPropRatio = ratioFloat(c.P2PAvgPropStat.Mean, c.SLIMAvgPropStat.Mean, c.P2PSuccess, c.SLIMSuccess)
	c.P95PropRatio = ratioFloat(c.P2PP95PropStat.Mean, c.SLIMP95PropStat.Mean, c.P2PSuccess, c.SLIMSuccess)
	return c
}

func ratioFloat(p2p, slim float64, p2pOk, slimOk bool) string {
	if !p2pOk || !slimOk || p2p <= 0 || slim <= 0 {
		return ""
	}
	return fmt.Sprintf("%.1f×", p2p/slim)
}

func atof(v string) float64 {
	f, _ := strconv.ParseFloat(v, 64)
	return f
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

// loadPlanMeta scans a scenario directory and maps each scenario's
// metadata.name to its description and order so the dashboard can annotate and
// sort every comparison using values sourced from the plan yaml.
func loadPlanMeta(dir string) map[string]planMeta {
	out := map[string]planMeta{}
	if dir == "" {
		return out
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return out
	}
	for _, path := range matches {
		s, err := scenario.LoadFile(path)
		if err != nil {
			continue
		}
		out[s.Metadata.Name] = planMeta{
			Description: s.Metadata.Description,
			Order:       s.Metadata.Order,
		}
	}
	return out
}

func groupByScenario(results []metrics.RunResult, metas map[string]planMeta) []scenarioComparison {
	byScenario := map[string]*scenarioComparison{}
	for _, r := range results {
		sc, ok := byScenario[r.ScenarioName]
		if !ok {
			meta := metas[r.ScenarioName]
			sc = &scenarioComparison{
				ScenarioName: r.ScenarioName,
				Description:  meta.Description,
				Order:        meta.Order,
			}
			byScenario[r.ScenarioName] = sc
		}
		switch r.Implementation {
		case "p2p-relay-stream":
			sc.P2P = r
			sc.HasP2P = true
		case "slim-group-session":
			sc.SLIM = r
			sc.HasSLIM = true
		}
	}
	out := make([]scenarioComparison, 0, len(byScenario))
	for _, sc := range byScenario {
		out = append(out, *sc)
	}
	sort.SliceStable(out, func(i, j int) bool {
		oi, oj := out[i].Order, out[j].Order
		switch {
		case oi != 0 && oj != 0 && oi != oj:
			return oi < oj
		case oi != 0 && oj == 0:
			return true
		case oi == 0 && oj != 0:
			return false
		default:
			return out[i].ScenarioName < out[j].ScenarioName
		}
	})
	return out
}

type cellKey struct {
	LatencyMs    int64
	Agents       int
	PayloadBytes int
	ThinkTimeMs  int64
}

type cellPair struct {
	P2P  *metrics.RunResult
	SLIM *metrics.RunResult
}

func buildMatrixTables(results []metrics.RunResult) []matrixTable {
	byKey := map[cellKey]cellPair{}
	latencyThink := map[int64]int64{}

	for i := range results {
		r := results[i]
		if !strings.HasPrefix(r.ScenarioName, "matrix-lat") {
			continue
		}
		k := cellKey{
			LatencyMs:    r.LatencyMs,
			Agents:       r.Agents,
			PayloadBytes: r.PayloadBytes,
			ThinkTimeMs:  r.ThinkTimeMs,
		}
		p := byKey[k]
		cp := r
		switch r.Implementation {
		case "p2p-relay-stream":
			p.P2P = &cp
		case "slim-group-session":
			p.SLIM = &cp
		}
		byKey[k] = p
		latencyThink[r.LatencyMs] = r.ThinkTimeMs
	}

	latencies := sortedInt64Keys(latencyThink)
	var tables []matrixTable
	for _, lat := range latencies {
		agentSet := map[int]struct{}{}
		payloadSet := map[int]struct{}{}
		for k := range byKey {
			if k.LatencyMs != lat {
				continue
			}
			agentSet[k.Agents] = struct{}{}
			payloadSet[k.PayloadBytes] = struct{}{}
		}
		tbl := matrixTable{
			LatencyMs:   lat,
			ThinkTimeMs: latencyThink[lat],
			AgentRows:   sortedIntKeys(agentSet),
			PayloadCols: sortedIntKeys(payloadSet),
			Cells:       map[int]map[int]matrixCell{},
		}
		for _, agents := range tbl.AgentRows {
			tbl.Cells[agents] = map[int]matrixCell{}
			for _, payload := range tbl.PayloadCols {
				k := cellKey{
					LatencyMs:    lat,
					Agents:       agents,
					PayloadBytes: payload,
					ThinkTimeMs:  tbl.ThinkTimeMs,
				}
				p := byKey[k]
				tbl.Cells[agents][payload] = mergeMatrixCell(p)
			}
		}
		tables = append(tables, tbl)
	}
	return tables
}

func mergeMatrixCell(p cellPair) matrixCell {
	c := matrixCell{}
	if p.P2P != nil {
		c.HasP2P = true
		c.P2PSuccess = p.P2P.Success
		c.P2PWall = p.P2P.ConsensusWallMS
		c.P2PEmitted = p.P2P.FindingsEmitted
		c.P2PReceived = p.P2P.FindingsReceivedTotal
		c.P2PAvgProp = p.P2P.AvgPropagationMS
		c.P2PP95Prop = p.P2P.P95PropagationMS
	}
	if p.SLIM != nil {
		c.HasSLIM = true
		c.SLIMSuccess = p.SLIM.Success
		c.SLIMWall = p.SLIM.ConsensusWallMS
		c.SLIMEmitted = p.SLIM.FindingsEmitted
		c.SLIMReceived = p.SLIM.FindingsReceivedTotal
		c.SLIMAvgProp = p.SLIM.AvgPropagationMS
		c.SLIMP95Prop = p.SLIM.P95PropagationMS
	}
	c.WallRatio = ratioInt64(c.P2PWall, c.SLIMWall, c.P2PSuccess, c.SLIMSuccess)
	c.EmittedRatio = ratioInt(c.P2PEmitted, c.SLIMEmitted, c.HasP2P, c.HasSLIM)
	c.ReceivedRatio = ratioInt(c.P2PReceived, c.SLIMReceived, c.HasP2P, c.HasSLIM)
	c.AvgPropRatio = ratioInt64(c.P2PAvgProp, c.SLIMAvgProp, c.P2PSuccess, c.SLIMSuccess)
	c.P95PropRatio = ratioInt64(c.P2PP95Prop, c.SLIMP95Prop, c.P2PSuccess, c.SLIMSuccess)
	return c
}

func ratioInt64(p2p, slim int64, p2pOk, slimOk bool) string {
	if !p2pOk || !slimOk || p2p <= 0 || slim <= 0 {
		return ""
	}
	return fmt.Sprintf("%.1f×", float64(p2p)/float64(slim))
}

func ratioInt(p2p, slim int, hasP2P, hasSLIM bool) string {
	if !hasP2P || !hasSLIM || p2p <= 0 || slim <= 0 {
		return ""
	}
	return fmt.Sprintf("%.1f×", float64(p2p)/float64(slim))
}

func sortedIntKeys(set map[int]struct{}) []int {
	out := make([]int, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func sortedInt64Keys(m map[int64]int64) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func formatPayloadLabel(bytes int) string {
	switch {
	case bytes == 0:
		return "0 B"
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes%1024 == 0:
		return fmt.Sprintf("%d KB", bytes/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatWall(success bool, wall int64) string {
	if !success && wall == 0 {
		return "FAIL"
	}
	return strconv.FormatInt(wall, 10)
}

func formatStatInt64(p2pOk, slimOk, hasStats bool, p2pStat, slimStat metricStat, p2pFallback, slimFallback int64) string {
	if hasStats {
		return formatStatPair(p2pOk, slimOk, p2pStat, slimStat, formatWall(p2pOk, p2pFallback), formatWall(slimOk, slimFallback))
	}
	return fmt.Sprintf("%s / %s", formatWall(p2pOk, p2pFallback), formatWall(slimOk, slimFallback))
}

func formatStatInt(hasP2P, hasSLIM, hasStats bool, p2pStat, slimStat metricStat, p2pFallback, slimFallback int) string {
	if hasStats {
		return formatStatPair(hasP2P, hasSLIM, p2pStat, slimStat, formatCount(hasP2P, p2pFallback), formatCount(hasSLIM, slimFallback))
	}
	return fmt.Sprintf("%s / %s", formatCount(hasP2P, p2pFallback), formatCount(hasSLIM, slimFallback))
}

func formatStatPair(leftOk, rightOk bool, leftStat, rightStat metricStat, leftFallback, rightFallback string) string {
	return fmt.Sprintf("%s / %s", formatStatValue(leftOk, leftStat, leftFallback), formatStatValue(rightOk, rightStat, rightFallback))
}

func formatStatDispersionInt64(hasStats, p2pOk, slimOk bool, p2pStat, slimStat metricStat) string {
	if !hasStats {
		return ""
	}
	return formatStatDispersionPair(p2pOk, slimOk, p2pStat, slimStat)
}

func formatStatDispersionInt(hasStats, hasP2P, hasSLIM bool, p2pStat, slimStat metricStat) string {
	if !hasStats {
		return ""
	}
	return formatStatDispersionPair(hasP2P, hasSLIM, p2pStat, slimStat)
}

func formatStatDispersionPair(leftOk, rightOk bool, leftStat, rightStat metricStat) string {
	if leftStat.N <= 1 && rightStat.N <= 1 {
		return ""
	}
	return fmt.Sprintf("%s / %s", formatStatDispersionSide(leftOk, leftStat), formatStatDispersionSide(rightOk, rightStat))
}

func formatStatDispersionSide(ok bool, stat metricStat) string {
	if stat.N > 1 {
		return fmt.Sprintf("σ=%.0f · n=%d", stat.StdDev, stat.N)
	}
	if stat.N == 1 {
		return "n=1"
	}
	if !ok {
		return "—"
	}
	return "—"
}

func formatStatSideInt64(ok bool, hasStats bool, stat metricStat, fallback int64) string {
	if hasStats {
		return formatStatValue(ok, stat, formatWall(ok, fallback))
	}
	return formatWall(ok, fallback)
}

func formatStatSideInt(has bool, hasStats bool, stat metricStat, fallback int) string {
	if hasStats {
		return formatStatValue(has, stat, formatCount(has, fallback))
	}
	return formatCount(has, fallback)
}

func formatStatDispersionSideInt64(ok bool, hasStats bool, stat metricStat) string {
	if !hasStats || stat.N <= 1 {
		return ""
	}
	return formatStatDispersionSide(ok, stat)
}

func formatStatDispersionSideInt(has bool, hasStats bool, stat metricStat) string {
	if !hasStats || stat.N <= 1 {
		return ""
	}
	return formatStatDispersionSide(has, stat)
}

func formatStatValue(ok bool, stat metricStat, fallback string) string {
	if !ok && stat.N == 0 {
		if fallback == "0" || fallback == "—" {
			return "FAIL"
		}
	}
	if stat.N > 1 {
		return fmt.Sprintf("%.0f [%.0f, %.0f]", stat.Mean, stat.CILow, stat.CIHigh)
	}
	if stat.N == 1 {
		return fmt.Sprintf("%.0f", stat.Mean)
	}
	return fallback
}

func formatCount(has bool, v int) string {
	if !has {
		return "—"
	}
	return strconv.Itoa(v)
}

func formatLatencyFixed(latencyMs int64) string {
	if latencyMs == 0 {
		return "All agents local (0 ms one-way latency)"
	}
	return fmt.Sprintf("%d ms one-way on ~⅓ of agents (trailing workers; coordinator stays local)", latencyMs)
}

type reportData struct {
	Comparisons []scenarioComparison
	Sweep       []metrics.RunResult
	Matrices    []matrixTable
	MatrixOnly  bool
}

func writeHTML(path string, comparisons []scenarioComparison, sweep []metrics.RunResult, matrices []matrixTable) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmpl := template.Must(template.New("report").Funcs(template.FuncMap{
		"formatPayload":            formatPayloadLabel,
		"formatWall":               formatWall,
		"formatCount":              formatCount,
		"formatStatInt64":            formatStatInt64,
		"formatStatInt":              formatStatInt,
		"formatStatSideInt64":        formatStatSideInt64,
		"formatStatSideInt":          formatStatSideInt,
		"formatStatDispersionInt64":       formatStatDispersionInt64,
		"formatStatDispersionInt":         formatStatDispersionInt,
		"formatStatDispersionSideInt64":   formatStatDispersionSideInt64,
		"formatStatDispersionSideInt":     formatStatDispersionSideInt,
		"formatLatencyFixed":              formatLatencyFixed,
	}).Parse(htmlTemplate))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, reportData{Comparisons: comparisons, Sweep: sweep, Matrices: matrices})
}

func writeMatrixHTML(path string, matrices []matrixTable) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmpl := template.Must(template.New("matrix").Funcs(template.FuncMap{
		"formatPayload":            formatPayloadLabel,
		"formatWall":               formatWall,
		"formatCount":              formatCount,
		"formatStatInt64":            formatStatInt64,
		"formatStatInt":              formatStatInt,
		"formatStatSideInt64":        formatStatSideInt64,
		"formatStatSideInt":          formatStatSideInt,
		"formatStatDispersionInt64":       formatStatDispersionInt64,
		"formatStatDispersionInt":         formatStatDispersionInt,
		"formatStatDispersionSideInt64":   formatStatDispersionSideInt64,
		"formatStatDispersionSideInt":     formatStatDispersionSideInt,
		"formatLatencyFixed":              formatLatencyFixed,
	}).Parse(matrixHTMLTemplate))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, reportData{Matrices: matrices, MatrixOnly: true})
}

func atoi(v string) int {
	n, _ := strconv.Atoi(v)
	return n
}

func atoi64(v string) int64 {
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

const matrixStyles = `
    .matrix-slice { margin: 2rem 0 2.5rem; overflow-x: auto; -webkit-overflow-scrolling: touch; }
    .matrix-fixed {
      background: #eef3fa;
      border: 1px solid #c5d4e8;
      border-radius: 6px 6px 0 0;
      margin: 0;
      padding: 0.75rem 1rem;
      color: #1a2a3a;
      font-size: 0.95rem;
      line-height: 1.45;
    }
    .matrix-fixed strong { font-weight: 650; }
    .matrix-slice table.matrix {
      margin-top: 0;
      border-top: none;
      width: max-content;
      min-width: 100%;
    }
    .matrix-slice table.matrix thead th {
      background: #2d4a6f;
      color: #f8fafc;
      font-weight: 650;
      font-size: 0.875rem;
      letter-spacing: 0.02em;
      border-color: #243d5c;
    }
    .matrix-slice table.matrix thead th.matrix-corner {
      background: #1e334d;
      font-weight: 700;
    }
    .matrix-slice table.matrix thead th.matrix-corner-split {
      width: 8rem;
      min-width: 8rem;
      max-width: 8rem;
      height: 4.25rem;
      padding: 0;
      vertical-align: middle;
      position: relative;
      overflow: hidden;
    }
    .matrix-corner-split .corner-payload {
      position: absolute;
      top: 0.4rem;
      right: 0.5rem;
      left: 2.75rem;
      font-size: 0.62rem;
      font-weight: 650;
      line-height: 1.2;
      text-align: right;
    }
    .matrix-corner-split .corner-agents {
      position: absolute;
      bottom: 0.4rem;
      left: 0.5rem;
      right: 2.75rem;
      font-size: 0.62rem;
      font-weight: 650;
      line-height: 1.2;
      text-align: left;
    }
    .matrix-corner-split::after {
      content: '';
      position: absolute;
      inset: 0;
      background: linear-gradient(
        to top right,
        transparent calc(50% - 0.5px),
        rgba(248, 250, 252, 0.45) calc(50% - 0.5px),
        rgba(248, 250, 252, 0.45) calc(50% + 0.5px),
        transparent calc(50% + 0.5px)
      );
      pointer-events: none;
    }
    .matrix-slice table.matrix tbody th {
      background: #e8eef5;
      color: #1a2a3a;
      font-weight: 650;
      border-color: #c5d4e8;
      white-space: nowrap;
    }
    .matrix-slice table.matrix tbody td {
      background: #fff;
      padding: 0.4rem 0.45rem;
      vertical-align: top;
      min-width: 22rem;
    }
    .matrix-slice table.matrix tbody tr:nth-child(even) td {
      background: #fafbfc;
    }
    .matrix-slice table.matrix tr:first-child th { border-top: none; }
    .matrix-legend { color: #555; font-size: 0.875rem; margin: 0 0 1rem; }
    .matrix-toolbar {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      margin: 1rem 0 0.5rem;
      padding: 0.65rem 0.9rem;
      background: #f5f7fa;
      border: 1px solid #dde3ea;
      border-radius: 6px;
    }
    .matrix-toolbar label { font-size: 0.9rem; color: #333; }
    .matrix-toolbar select {
      font: inherit;
      font-size: 0.9rem;
      padding: 0.35rem 0.6rem;
      border: 1px solid #c5ccd6;
      border-radius: 4px;
      background: #fff;
      min-width: 14rem;
    }
    .matrix-toolbar .matrix-toggle {
      display: inline-flex;
      align-items: center;
      gap: 0.35rem;
      font-size: 0.9rem;
      color: #333;
      margin-left: auto;
      cursor: pointer;
      user-select: none;
    }
    .matrix-toolbar .matrix-toggle input {
      margin: 0;
      cursor: pointer;
    }
    .stat-dispersion {
      display: none;
      color: #666;
      font-size: 0.8rem;
      line-height: 1.35;
    }
    body.show-dispersion .stat-dispersion:not(:empty) {
      display: block;
    }
    .matrix-cell-split {
      display: flex;
      flex-direction: column;
      gap: 0.35rem;
    }
    .matrix-subcells {
      display: grid;
      grid-template-columns: minmax(10.5rem, 1fr) minmax(10.5rem, 1fr);
      grid-template-rows: auto auto;
      gap: 0.2rem 0.35rem;
      position: relative;
    }
    body.show-dispersion .matrix-subcells:has(.stat-dispersion:not(:empty)) {
      grid-template-rows: auto auto auto;
    }
    .matrix-subcell-bg {
      grid-row: 1 / 3;
      border-radius: 4px;
      z-index: 0;
    }
    body.show-dispersion .matrix-subcells:has(.stat-dispersion:not(:empty)) .matrix-subcell-bg {
      grid-row: 1 / 4;
    }
    .matrix-subcell-bg.matrix-subcell-p2p {
      grid-column: 1;
      background: #f3f6fb;
      border: 1px solid #d4deec;
    }
    .matrix-subcell-bg.matrix-subcell-slim {
      grid-column: 2;
      background: #eef8f4;
      border: 1px solid #c8e6d6;
    }
    .matrix-col-p2p { grid-column: 1; }
    .matrix-col-slim { grid-column: 2; }
    .subcell-label {
      grid-row: 1;
      position: relative;
      z-index: 1;
      font-size: 0.65rem;
      font-weight: 650;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      color: #5a6a7a;
      text-align: center;
      padding: 0.35rem 0.4rem 0;
    }
    .subcell-value {
      grid-row: 2;
      position: relative;
      z-index: 1;
      font-size: 0.8rem;
      line-height: 1.35;
      font-variant-numeric: tabular-nums;
      white-space: nowrap;
      text-align: center;
      padding: 0.1rem 0.4rem 0.35rem;
    }
    .matrix-subcells .stat-dispersion {
      grid-row: 3;
      position: relative;
      z-index: 1;
      font-size: 0.72rem;
      line-height: 1.35;
      text-align: center;
      padding: 0 0.4rem 0.35rem;
      white-space: nowrap;
    }
    .matrix-cell-ratio {
      text-align: center;
      border-top: 1px solid #e8ecf0;
      padding-top: 0.25rem;
    }
    .matrix-cell-warn {
      text-align: center;
    }
    .metric-view[hidden] { display: none; }`

const matrixCellWarn = `{{if or (not .P2PSuccess) (not .SLIMSuccess)}}<div class="matrix-cell-warn"><small class="warn">{{if not .P2PSuccess}}P2P fail{{end}}{{if and (not .P2PSuccess) (not .SLIMSuccess)}} · {{end}}{{if not .SLIMSuccess}}SLIM fail{{end}}</small></div>{{end}}`

const matrixCellSplitInt64 = `
            <div class="matrix-cell-split">
              <div class="matrix-subcells">
                <div class="matrix-subcell-bg matrix-subcell-p2p" aria-hidden="true"></div>
                <div class="matrix-subcell-bg matrix-subcell-slim" aria-hidden="true"></div>
                <span class="subcell-label matrix-col-p2p">P2P</span>
                <span class="subcell-label matrix-col-slim">SLIM</span>
                <span class="subcell-value matrix-col-p2p">{{formatStatSideInt64 $p2pOk $hasStats $p2pStat $p2pFallback}}</span>
                <span class="subcell-value matrix-col-slim">{{formatStatSideInt64 $slimOk $hasStats $slimStat $slimFallback}}</span>
                <span class="stat-dispersion matrix-col-p2p">{{formatStatDispersionSideInt64 $p2pOk $hasStats $p2pStat}}</span>
                <span class="stat-dispersion matrix-col-slim">{{formatStatDispersionSideInt64 $slimOk $hasStats $slimStat}}</span>
              </div>
              {{if $ratio}}<div class="matrix-cell-ratio"><small class="ratio">{{$ratio}}</small></div>{{end}}
              ` + matrixCellWarn + `
            </div>`

const matrixCellSplitInt = `
            <div class="matrix-cell-split">
              <div class="matrix-subcells">
                <div class="matrix-subcell-bg matrix-subcell-p2p" aria-hidden="true"></div>
                <div class="matrix-subcell-bg matrix-subcell-slim" aria-hidden="true"></div>
                <span class="subcell-label matrix-col-p2p">P2P</span>
                <span class="subcell-label matrix-col-slim">SLIM</span>
                <span class="subcell-value matrix-col-p2p">{{formatStatSideInt $p2pOk $hasStats $p2pStat $p2pFallback}}</span>
                <span class="subcell-value matrix-col-slim">{{formatStatSideInt $slimOk $hasStats $slimStat $slimFallback}}</span>
                <span class="stat-dispersion matrix-col-p2p">{{formatStatDispersionSideInt $p2pOk $hasStats $p2pStat}}</span>
                <span class="stat-dispersion matrix-col-slim">{{formatStatDispersionSideInt $slimOk $hasStats $slimStat}}</span>
              </div>
              {{if $ratio}}<div class="matrix-cell-ratio"><small class="ratio">{{$ratio}}</small></div>{{end}}
              ` + matrixCellWarn + `
            </div>`

const matrixSection = `
  {{if .Matrices}}
  <h2 id="matrices">Matrix sweep (agents × payload at fixed latency)</h2>
  <div class="matrix-toolbar">
    <label for="matrix-metric-select"><strong>Metric</strong></label>
    <select id="matrix-metric-select" aria-label="Matrix metric">
      <option value="consensus_wall_ms" selected>Consensus wall (ms)</option>
      <option value="findings_emitted">Findings emitted</option>
      <option value="findings_received_total">Findings received (total)</option>
      <option value="avg_propagation_ms">Avg propagation (ms)</option>
      <option value="p95_propagation_ms">P95 propagation (ms)</option>
    </select>
    <label class="matrix-toggle" for="matrix-dispersion-toggle">
      <input type="checkbox" id="matrix-dispersion-toggle" aria-label="Show standard deviation and sample size">
      Show σ
    </label>
  </div>
  <p class="matrix-legend" id="matrix-legend">Each block fixes latency for every cell. Rows vary agent count; columns vary payload size. Each cell splits into <strong>P2P</strong> and <strong>SLIM</strong> columns with mean and 95% CI when aggregated over multiple runs; toggle <strong>Show σ</strong> for standard deviation and sample size.</p>
  {{range .Matrices}}
  {{$tbl := .}}
  <div class="matrix-slice">
    <p class="matrix-fixed">
      Fixed latency: <strong>{{$tbl.LatencyMs}} ms</strong> — {{formatLatencyFixed $tbl.LatencyMs}}<br>
      Fixed think time: <strong>{{$tbl.ThinkTimeMs}} ms</strong> · Swept axes: agents (rows) × payload (columns)
    </p>
    <table class="matrix">
      <thead>
        <tr><th class="matrix-corner matrix-corner-split" scope="col"><span class="corner-payload">Payload size</span><span class="corner-agents">Number of Agents</span></th>{{range $tbl.PayloadCols}}<th>{{formatPayload .}}</th>{{end}}</tr>
      </thead>
      <tbody>
      {{range $agents := $tbl.AgentRows}}
      <tr>
        <th scope="row">{{$agents}}</th>
        {{range $payload := $tbl.PayloadCols}}
        {{with index (index $tbl.Cells $agents) $payload}}
        {{$c := .}}
        <td>
          <span class="metric-view" data-metric="consensus_wall_ms">
            {{$hasStats := $c.HasStats}}{{$p2pOk := $c.P2PSuccess}}{{$slimOk := $c.SLIMSuccess}}
            {{$p2pStat := $c.P2PWallStat}}{{$slimStat := $c.SLIMWallStat}}
            {{$p2pFallback := $c.P2PWall}}{{$slimFallback := $c.SLIMWall}}{{$ratio := $c.WallRatio}}
            ` + matrixCellSplitInt64 + `
          </span>
          <span class="metric-view" data-metric="findings_emitted" hidden>
            {{$hasStats := $c.HasStats}}{{$p2pOk := $c.HasP2P}}{{$slimOk := $c.HasSLIM}}
            {{$p2pStat := $c.P2PEmittedStat}}{{$slimStat := $c.SLIMEmittedStat}}
            {{$p2pFallback := $c.P2PEmitted}}{{$slimFallback := $c.SLIMEmitted}}{{$ratio := $c.EmittedRatio}}
            ` + matrixCellSplitInt + `
          </span>
          <span class="metric-view" data-metric="findings_received_total" hidden>
            {{$hasStats := $c.HasStats}}{{$p2pOk := $c.HasP2P}}{{$slimOk := $c.HasSLIM}}
            {{$p2pStat := $c.P2PReceivedStat}}{{$slimStat := $c.SLIMReceivedStat}}
            {{$p2pFallback := $c.P2PReceived}}{{$slimFallback := $c.SLIMReceived}}{{$ratio := $c.ReceivedRatio}}
            ` + matrixCellSplitInt + `
          </span>
          <span class="metric-view" data-metric="avg_propagation_ms" hidden>
            {{$hasStats := $c.HasStats}}{{$p2pOk := $c.P2PSuccess}}{{$slimOk := $c.SLIMSuccess}}
            {{$p2pStat := $c.P2PAvgPropStat}}{{$slimStat := $c.SLIMAvgPropStat}}
            {{$p2pFallback := $c.P2PAvgProp}}{{$slimFallback := $c.SLIMAvgProp}}{{$ratio := $c.AvgPropRatio}}
            ` + matrixCellSplitInt64 + `
          </span>
          <span class="metric-view" data-metric="p95_propagation_ms" hidden>
            {{$hasStats := $c.HasStats}}{{$p2pOk := $c.P2PSuccess}}{{$slimOk := $c.SLIMSuccess}}
            {{$p2pStat := $c.P2PP95PropStat}}{{$slimStat := $c.SLIMP95PropStat}}
            {{$p2pFallback := $c.P2PP95Prop}}{{$slimFallback := $c.SLIMP95Prop}}{{$ratio := $c.P95PropRatio}}
            ` + matrixCellSplitInt64 + `
          </span>
        </td>
        {{else}}
        <td>—</td>
        {{end}}
        {{end}}
      </tr>
      {{end}}
      </tbody>
    </table>
  </div>
  {{end}}
  {{end}}`

const matrixSwitcherScript = `
  <script>
    (function () {
      var legends = {
        consensus_wall_ms: 'Each block fixes latency for every cell. Cells split into P2P and SLIM columns for consensus wall (ms), with P2P÷SLIM ratio below when both succeeded.',
        findings_emitted: 'Total findings emitted by all agents during the run. P2P and SLIM columns show counts per side; ratio when both reported data.',
        findings_received_total: 'Total findings applied by all agents. P2P and SLIM columns show counts per side; ratio when both reported data.',
        avg_propagation_ms: 'Average per-finding delivery latency (emit → apply). P2P and SLIM columns show avg propagation (ms); ratio when both succeeded.',
        p95_propagation_ms: '95th percentile per-finding delivery latency. P2P and SLIM columns show P95 propagation (ms); ratio when both succeeded.'
      };
      var select = document.getElementById('matrix-metric-select');
      var legend = document.getElementById('matrix-legend');
      var dispToggle = document.getElementById('matrix-dispersion-toggle');
      if (!select || !legend) return;
      function applyMetric(metric) {
        document.querySelectorAll('.metric-view').forEach(function (el) {
          el.hidden = el.getAttribute('data-metric') !== metric;
        });
        legend.textContent = legends[metric] || '';
      }
      select.addEventListener('change', function () { applyMetric(select.value); });
      if (dispToggle) {
        dispToggle.addEventListener('change', function () {
          document.body.classList.toggle('show-dispersion', dispToggle.checked);
        });
      }
    })();
  </script>`

const reportStyles = `
    body { font-family: system-ui, sans-serif; margin: 2rem; max-width: 1100px; }
    table { border-collapse: collapse; width: 100%; margin-bottom: 2rem; }
    th, td { border: 1px solid #ccc; padding: 0.5rem 0.75rem; text-align: right; }
    th:first-child, td:first-child { text-align: left; }
    table.matrix td { min-width: 6rem; vertical-align: top; }
    .walls { font-weight: 600; }
    .ratio { color: #555; }
    .warn { color: #a33; }
    h2 { margin-top: 2rem; }
    h3 { margin-top: 1.5rem; }` + matrixStyles + `
    .intro { background: #f5f7fa; border: 1px solid #dde3ea; border-radius: 6px; padding: 1rem 1.25rem; }
    .intro code { background: #eceff3; padding: 0.05rem 0.3rem; border-radius: 3px; }
    dl.defs dt { font-weight: 600; margin-top: 0.75rem; }
    dl.defs dd { margin: 0.2rem 0 0 1.25rem; color: #333; }
    .arch-diagrams { display: flex; flex-direction: column; gap: 1.25rem; margin: 1rem 0 1.5rem; }
    .diagram {
      border: 1px solid #dde3ea;
      border-radius: 6px;
      padding: 1rem 1.25rem 1.25rem;
      background: #fff;
      overflow-x: auto;
    }
    .diagram h3 {
      margin: 0 0 0.75rem;
      font-size: 0.95rem;
      font-weight: 650;
      color: #1a2a3a;
      line-height: 1.35;
    }
    .diagram h3 small { display: block; font-weight: 450; color: #555; margin-top: 0.2rem; }
    .diagram pre.mermaid { margin: 0; background: transparent; }
    .diagram svg { display: block; width: 100%; max-width: 520px; height: auto; margin: 0 auto; }
    .diagram .nodeLabel, .diagram .edgeLabel { font-size: 13px !important; }
    p.plandesc { background: #fbfaf3; border-left: 3px solid #d8b83f; margin: 0.25rem 0 0.75rem; padding: 0.5rem 0.9rem; color: #444; }`

const reportIntroSection = `
  <h1>Agent Consensus Convergence</h1>
  <div class="intro">
    <p>This test runs the same distributed <strong>hypothesis-convergence</strong> workload on two
    transports and measures how fast N agents reach identical consensus. <strong>SLIM group multicast</strong>
    (<code>slim-group-session</code>) broadcasts each finding once over a native group session and the
    dataplane fans it out — there is <em>no relay</em>. <strong>P2P streaming</strong> (<code>p2p-relay-stream</code>)
    has no group multicast, so the runner is an explicit <em>relay hub</em>: every finding makes two streaming
    hops (<code>agent → runner → N−1 agents</code>). The goal is to show that many-to-many group multicast
    beats point-to-point streaming, not to compare messaging frameworks. The P2P streaming leg is implemented
    with the <a href="https://github.com/a2aproject/a2a-go">A2A SDK</a> (server-streaming through the runner);
    it stands in for the point-to-point streaming pattern rather than being the subject of comparison.</p>
    <p><strong>How the test works.</strong> Each agent runs a small consensus engine. In every round an agent
    <em>thinks</em> and, when its opinion changes, <em>emits a finding</em> — its current value and confidence —
    to the group. Peers <em>apply</em> incoming findings: a value gains confidence as more distinct agents
    support it, and agents drift toward the shared majority <em>target value</em>. An agent reaches
    <em>local consensus</em> once it holds the target value with confidence above a fixed threshold, and the
    epoch reaches <em>global consensus</em> the moment <em>every</em> agent agrees. Each run repeats this attempt
    over several <em>epochs</em> to expose reliability under load. The only thing that differs between the two
    runs is <em>how findings travel</em> between agents — shown in the architecture below.</p>
    <p><em>Note: absolute timings depend on the CPU and load of the machine that produced this report, so the
    raw numbers vary run to run. Compare the two transports within the same run rather than across machines.</em></p>
  </div>`

const reportArchSection = `
  <h2 id="arch">Architecture</h2>
  <p>Both runs use the same agents and consensus logic; only the message path differs. <strong>SLIM group multicast</strong>
    agents publish each finding once to a native group session and the dataplane fans it out to all peers (the
    runner only observes). <strong>P2P streaming</strong> streams every finding to the runner acting as a
    <em>relay hub</em>, which re-streams it to every other agent — two hops per finding, all through one point.</p>
  <div class="arch-diagrams">
    <div class="diagram">
      <h3>SLIM group multicast<small>Native group session — one publish, dataplane fan-out, no relay</small></h3>
      <pre class="mermaid">
flowchart TD
  R["Runner<br/>start signal + metrics"]
  G["Group session<br/>dataplane fan-out"]
  A0["Agent 0"]
  A1["Agent 1"]
  A2["Agent 2"]
  A0 -->|"publish finding"| G
  A1 -->|"publish finding"| G
  A2 -->|"publish finding"| G
  G -->|"deliver to peers"| A0
  G --> A1
  G --> A2
  A0 -.->|"snapshot on converge"| R
  A1 -.-> R
  A2 -.-> R
      </pre>
    </div>
    <div class="diagram">
      <h3>P2P streaming<small>Runner is the relay hub — two hops per finding through one point</small></h3>
      <pre class="mermaid">
flowchart TD
  A0["Agent 0"]
  A1["Agent 1"]
  A2["Agent 2"]
  H["Runner<br/>RELAY hub"]
  A0 -->|"stream finding"| H
  A1 --> H
  A2 --> H
  H -->|"relay to each peer"| A0
  H --> A1
  H --> A2
      </pre>
    </div>
  </div>`

const reportMetricDefsSection = `
  <h2 id="defs">Metric definitions</h2>
  <dl class="defs">
    <dt>Consensus wall (ms)</dt>
    <dd>Headline metric: wall-clock time from the runner's single <em>start</em> broadcast until every agent
      has reached identical local consensus. Excludes process/library startup and teardown.</dd>
    <dt>Last agent converge (ms)</dt>
    <dd>Time until the slowest agent converged, taken from each agent's own convergence timestamp relative to
      start. Highlights stragglers behind the headline number.</dd>
    <dt>Avg / P95 propagation (ms)</dt>
    <dd>Per-finding delivery latency, from the moment a finding is emitted to when a peer applies it
      (average and 95th percentile). SLIM is one multicast hop; P2P streaming adds the extra relay hop, so it grows
      with agent count, message rate, and payload size.</dd>
    <dt>Stream RPC count</dt>
    <dd>Number of finding-carrying messages on the data path. For SLIM this equals findings emitted (one
      native multicast each). For P2P streaming it is the relay deliveries, ≈ <code>findings × (N−1)</code>, because the
      hub re-sends every finding to all other agents.</dd>
    <dt>Epochs (ok / failed)</dt>
    <dd>Each run repeats the same consensus attempt over several <em>epochs</em>. An epoch succeeds if every
      agent reaches global consensus within <code>spec.maxEpochTimeMs</code> (each attempt runs up to
      <code>maxRounds</code>); otherwise it is counted as failed. This surfaces reliability differences: under
      load P2P streaming may miss the per-epoch budget while SLIM still converges. Overall <em>Success</em> is true only
      when no epoch failed.</dd>
    <dt>Coord fanout (ms)</dt>
    <dd>Cumulative time the P2P streaming relay hub spent fanning findings out to peers. <code>0</code> for SLIM because
      there is no relay — the dataplane does the fan-out.</dd>
    <dt>Payload B</dt>
    <dd>Fixed-size padding (<code>spec.payloadBytes</code>) added to every finding to stress transport
      bandwidth. Semantically inert; it does not change the consensus math or round count.</dd>
  </dl>`

const reportMermaidScript = `
  <script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
    mermaid.initialize({
      startOnLoad: true,
      securityLevel: 'strict',
      theme: 'base',
      flowchart: { htmlLabels: true, curve: 'basis', padding: 16, nodeSpacing: 28, rankSpacing: 40 },
      themeVariables: {
        fontFamily: 'system-ui, sans-serif',
        fontSize: '14px',
        primaryColor: '#eef3fa',
        primaryTextColor: '#1a2a3a',
        primaryBorderColor: '#2d4a6f',
        lineColor: '#5a6a7e',
        secondaryColor: '#f5f7fa',
        tertiaryColor: '#fff',
        clusterBkg: '#f5f7fa',
        clusterBorder: '#c5d4e8',
        edgeLabelBackground: '#fff'
      }
    });
  </script>`

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Agent Consensus Convergence</title>
  <style>` + reportStyles + `
  </style>
</head>
<body>` + reportIntroSection + reportArchSection + `
  {{range .Comparisons}}
  <h2>{{.ScenarioName}}</h2>
  {{if .Description}}<p class="plandesc">{{.Description}}</p>{{end}}
  <table>
    <tr><th>Metric</th><th>P2P</th><th>SLIM</th></tr>
    <tr><td>Consensus wall (ms)</td><td>{{if .HasP2P}}{{.P2P.ConsensusWallMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.ConsensusWallMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Last agent converge (ms)</td><td>{{if .HasP2P}}{{.P2P.LastAgentConvergeMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.LastAgentConvergeMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Avg propagation (ms)</td><td>{{if .HasP2P}}{{.P2P.AvgPropagationMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.AvgPropagationMS}}{{else}}—{{end}}</td></tr>
    <tr><td>P95 propagation (ms)</td><td>{{if .HasP2P}}{{.P2P.P95PropagationMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.P95PropagationMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Stream RPC count</td><td>{{if .HasP2P}}{{.P2P.StreamRPCCount}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.StreamRPCCount}}{{else}}—{{end}}</td></tr>
    <tr><td>Epochs (ok / failed)</td><td>{{if .HasP2P}}{{.P2P.EpochsSucceeded}} / {{.P2P.EpochsFailed}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.EpochsSucceeded}} / {{.SLIM.EpochsFailed}}{{else}}—{{end}}</td></tr>
    <tr><td>Success</td><td>{{if .HasP2P}}{{.P2P.Success}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.Success}}{{else}}—{{end}}</td></tr>
  </table>
  {{end}}
  {{if .Sweep}}
  <h2>Sweep results</h2>
  <table>
    <tr><th>Scenario</th><th>Impl</th><th>Agents</th><th>Think ms</th><th>Payload B</th><th>Latency ms</th><th>Consensus wall</th><th>Avg propagation</th><th>P95 propagation</th><th>Stream RPCs</th><th>Epochs ok/failed</th></tr>
    {{range .Sweep}}
    <tr>
      <td>{{.ScenarioName}}</td><td>{{.Implementation}}</td><td>{{.Agents}}</td><td>{{.ThinkTimeMs}}</td><td>{{.PayloadBytes}}</td><td>{{.LatencyMs}}</td>
      <td>{{.ConsensusWallMS}}</td><td>{{.AvgPropagationMS}}</td><td>{{.P95PropagationMS}}</td><td>{{.StreamRPCCount}}</td><td>{{.EpochsSucceeded}}/{{.EpochsFailed}}</td>
    </tr>
    {{end}}
  </table>
  {{end}}` + matrixSection + reportMetricDefsSection + matrixSwitcherScript + reportMermaidScript + `
</body>
</html>
`

const matrixHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Agent Consensus Convergence</title>
  <style>` + reportStyles + `
    body.matrix-report { max-width: none; }
  </style>
</head>
<body class="matrix-report">` + reportIntroSection + reportArchSection + matrixSection + `
  {{if not .Matrices}}
  <p>No matrix results found. Run <code>task compare:sweep:matrix</code> first.</p>
  {{end}}` + reportMetricDefsSection + matrixSwitcherScript + reportMermaidScript + `
</body>
</html>
`
