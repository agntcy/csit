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

type matrixCell struct {
	HasP2P      bool
	HasSLIM     bool
	P2PWall     int64
	SLIMWall    int64
	P2PSuccess  bool
	SLIMSuccess bool
	P2PRatio    string
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
		matrixResults, err := readTSV(*matrixTSV)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read matrix tsv: %v\n", err)
			os.Exit(1)
		}
		matrices := buildMatrixTables(matrixResults)
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
		if mr, err := readTSV(*matrixTSV); err == nil {
			matrices = buildMatrixTables(mr)
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
		c.P2PWall = p.P2P.ConsensusWallMS
		c.P2PSuccess = p.P2P.Success
	}
	if p.SLIM != nil {
		c.HasSLIM = true
		c.SLIMWall = p.SLIM.ConsensusWallMS
		c.SLIMSuccess = p.SLIM.Success
	}
	if c.HasP2P && c.HasSLIM && c.SLIMWall > 0 && c.P2PSuccess && c.P2PWall > 0 {
		ratio := float64(c.P2PWall) / float64(c.SLIMWall)
		c.P2PRatio = fmt.Sprintf("%.1f×", ratio)
	}
	return c
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
		"formatPayload": formatPayloadLabel,
		"formatWall":    formatWall,
		"formatLatencyFixed": formatLatencyFixed,
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
		"formatPayload": formatPayloadLabel,
		"formatWall":    formatWall,
		"formatLatencyFixed": formatLatencyFixed,
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
    .matrix-slice { margin: 2rem 0 2.5rem; }
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
    }
    .matrix-slice table.matrix tr:first-child th { border-top: none; }
    .matrix-legend { color: #555; font-size: 0.875rem; margin: 0 0 1rem; }`

const matrixSection = `
  {{if .Matrices}}
  <h2 id="matrices">Matrix sweep (agents × payload at fixed latency)</h2>
  <p class="matrix-legend">Each block below fixes latency for every cell in its table. Rows vary agent count;
  columns vary payload size. Cells show <strong>P2P / SLIM</strong> consensus wall (ms) and P2P÷SLIM ratio when both succeeded.</p>
  {{range .Matrices}}
  {{$tbl := .}}
  <div class="matrix-slice">
    <p class="matrix-fixed">
      Fixed latency: <strong>{{$tbl.LatencyMs}} ms</strong> — {{formatLatencyFixed $tbl.LatencyMs}}<br>
      Fixed think time: <strong>{{$tbl.ThinkTimeMs}} ms</strong> · Swept axes: agents (rows) × payload (columns)
    </p>
    <table class="matrix">
      <tr><th>Agents \ Payload</th>{{range $tbl.PayloadCols}}<th>{{formatPayload .}}</th>{{end}}</tr>
      {{range $agents := $tbl.AgentRows}}
      <tr>
        <th>{{$agents}}</th>
        {{range $payload := $tbl.PayloadCols}}
        {{with index (index $tbl.Cells $agents) $payload}}
        <td>
          <span class="walls">{{formatWall .P2PSuccess .P2PWall}} / {{formatWall .SLIMSuccess .SLIMWall}}</span>
          {{if .P2PRatio}}<br><small class="ratio">{{.P2PRatio}}</small>{{end}}
          {{if or (not .P2PSuccess) (not .SLIMSuccess)}}<br><small class="warn">{{if not .P2PSuccess}}P2P fail{{end}}{{if and (not .P2PSuccess) (not .SLIMSuccess)}} · {{end}}{{if not .SLIMSuccess}}SLIM fail{{end}}</small>{{end}}
        </td>
        {{else}}
        <td>—</td>
        {{end}}
        {{end}}
      </tr>
      {{end}}
    </table>
  </div>
  {{end}}
  {{end}}`

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
    .diagram { border: 1px solid #dde3ea; border-radius: 6px; padding: 1rem; background: #fff; overflow-x: auto; }
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
  <div class="diagram">
    <pre class="mermaid">
flowchart TB
  subgraph slim["SLIM group multicast — native group session, no relay"]
    R1["runner (passive observer: start signal + consumes pushed metrics)"]
    SA0["agent-0"]
    SA1["agent-1"]
    SA2["agent-2"]
    SA0 ---|"Publish finding → all (dataplane fan-out)"| SA1
    SA1 --- SA2
    SA0 --- SA2
    SA0 -.->|"push snapshot on convergence"| R1
    SA1 -.-> R1
    SA2 -.-> R1
  end
  subgraph p2p["P2P streaming — runner is the relay hub"]
    R2["runner = RELAY (hosts streaming server)"]
    AA0["agent-0"]
    AA1["agent-1"]
    AA2["agent-2"]
    AA0 -->|"stream findings → runner"| R2
    AA1 --> R2
    AA2 --> R2
    R2 -->|"stream relayed findings → agent"| AA0
    R2 --> AA1
    R2 --> AA2
  end
    </pre>
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
    mermaid.initialize({ startOnLoad: true, securityLevel: 'strict' });
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
  {{end}}` + matrixSection + reportMetricDefsSection + reportMermaidScript + `
</body>
</html>
`

const matrixHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Agent Consensus Convergence</title>
  <style>` + reportStyles + `
  </style>
</head>
<body>` + reportIntroSection + reportArchSection + matrixSection + `
  {{if not .Matrices}}
  <p>No matrix results found. Run <code>task compare:sweep:matrix</code> first.</p>
  {{end}}` + reportMetricDefsSection + reportMermaidScript + `
</body>
</html>
`
