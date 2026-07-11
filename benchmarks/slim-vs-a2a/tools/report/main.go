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

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/scenario"
)

type planMeta struct {
	Description string
	Order       int
}

type scenarioComparison struct {
	ScenarioName string
	Description  string
	Order        int
	A2A          metrics.RunResult
	SLIM         metrics.RunResult
	HasA2A       bool
	HasSLIM      bool
}

func main() {
	tsvPath := flag.String("tsv", "./reports/results.tsv", "comparison results tsv")
	sweepTSV := flag.String("sweep-tsv", "./reports/sweep.tsv", "optional sweep results tsv")
	scenarioDir := flag.String("scenario-dir", "./plans/sweeps", "directory of scenario yaml files (for per-plan descriptions)")
	output := flag.String("output", "./reports/index.html", "html dashboard output")
	flag.Parse()

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

	if err := writeHTML(*output, comparisons, sweepResults); err != nil {
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

	var out []metrics.RunResult
	for _, row := range records[1:] {
		if len(row) < 19 {
			continue
		}
		r := metrics.RunResult{
			ScenarioName:   row[0],
			Domain:         row[1],
			Implementation: row[2],
		}
		r.Agents = atoi(row[3])
		r.ThinkTimeMs = atoi64(row[4])
		r.PayloadBytes = atoi(row[5])
		r.ConsensusWallMS = atoi64(row[6])
		r.ConsensusRound = atoi(row[7])
		r.FindingsEmitted = atoi(row[8])
		r.FindingsReceivedTotal = atoi(row[9])
		r.AvgPropagationMS = atoi64(row[10])
		r.P95PropagationMS = atoi64(row[11])
		r.LastAgentConvergeMS = atoi64(row[12])
		r.CoordFanoutMS = atoi64(row[13])
		r.StreamRPCCount = atoi(row[14])
		r.Epochs = atoi(row[15])
		r.EpochsSucceeded = atoi(row[16])
		r.EpochsFailed = atoi(row[17])
		r.Success = row[18] == "true"
		if len(row) > 19 {
			r.Error = row[19]
		}
		out = append(out, r)
	}
	return out, nil
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
		case "a2a-relay-stream":
			sc.A2A = r
			sc.HasA2A = true
		case "slim-group-session":
			sc.SLIM = r
			sc.HasSLIM = true
		}
	}
	out := make([]scenarioComparison, 0, len(byScenario))
	for _, sc := range byScenario {
		out = append(out, *sc)
	}
	// Sort by explicit plan order first (0 == unordered sinks to the bottom),
	// then alphabetically by name for a stable, predictable layout.
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

type reportData struct {
	Comparisons []scenarioComparison
	Sweep       []metrics.RunResult
}

func writeHTML(path string, comparisons []scenarioComparison, sweep []metrics.RunResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmpl := template.Must(template.New("report").Funcs(template.FuncMap{
		"deltaPct": deltaPct,
	}).Parse(htmlTemplate))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, reportData{Comparisons: comparisons, Sweep: sweep})
}

// deltaPct expresses the A2A cost as an improvement relative to SLIM:
// (A2A − SLIM) / SLIM × 100. Unlike a reduction relative to A2A (capped at
// 100%), this is unbounded, so a 42× speedup reads as ~4100% rather than ~98%.
// Positive values mean SLIM was faster/cheaper.
func deltaPct(a2a, slim int64) string {
	if slim == 0 {
		return "n/a"
	}
	pct := (float64(a2a-slim) / float64(slim)) * 100
	return fmt.Sprintf("%.1f%%", pct)
}

func atoi(v string) int {
	n, _ := strconv.Atoi(v)
	return n
}

func atoi64(v string) int64 {
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>SLIM vs A2A — Consensus Streaming</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; max-width: 1100px; }
    table { border-collapse: collapse; width: 100%; margin-bottom: 2rem; }
    th, td { border: 1px solid #ccc; padding: 0.5rem 0.75rem; text-align: right; }
    th:first-child, td:first-child { text-align: left; }
    h2 { margin-top: 2rem; }
    .intro { background: #f5f7fa; border: 1px solid #dde3ea; border-radius: 6px; padding: 1rem 1.25rem; }
    .intro code { background: #eceff3; padding: 0.05rem 0.3rem; border-radius: 3px; }
    dl.defs dt { font-weight: 600; margin-top: 0.75rem; }
    dl.defs dd { margin: 0.2rem 0 0 1.25rem; color: #333; }
    .diagram { border: 1px solid #dde3ea; border-radius: 6px; padding: 1rem; background: #fff; overflow-x: auto; }
    p.plandesc { background: #fbfaf3; border-left: 3px solid #d8b83f; margin: 0.25rem 0 0.75rem; padding: 0.5rem 0.9rem; color: #444; }
  </style>
</head>
<body>
  <h1>SLIM vs A2A — Consensus Streaming</h1>
  <div class="intro">
    <p>This benchmark runs the same distributed <strong>hypothesis-convergence</strong> workload on two
    transports and measures how fast N agents reach identical consensus. <strong>SLIM</strong>
    (<code>slim-group-session</code>) broadcasts each finding once over a native group session and the
    dataplane fans it out — there is <em>no relay</em>. <strong>A2A</strong> (<code>a2a-relay-stream</code>)
    has no peer multicast, so the runner is an explicit <em>relay hub</em>: every finding makes two streaming
    hops (<code>agent → runner → N−1 agents</code>).</p>
    <p>The <strong>Improvement vs SLIM</strong> column is <code>(A2A − SLIM) / SLIM × 100</code> — how much
    more time/work A2A costs relative to SLIM. It is unbounded: e.g. <code>+4000%</code> means A2A took ~41×
    as long as SLIM. Positive values favor SLIM. See <a href="#defs">metric definitions</a> and the
    <a href="#arch">architecture diagram</a> below.</p>
  </div>
  {{range .Comparisons}}
  <h2>{{.ScenarioName}}</h2>
  {{if .Description}}<p class="plandesc">{{.Description}}</p>{{end}}
  <table>
    <tr><th>Metric</th><th>A2A</th><th>SLIM</th><th>Improvement vs SLIM</th></tr>
    <tr><td>Consensus wall (ms)</td><td>{{if .HasA2A}}{{.A2A.ConsensusWallMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.ConsensusWallMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.ConsensusWallMS .SLIM.ConsensusWallMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Last agent converge (ms)</td><td>{{if .HasA2A}}{{.A2A.LastAgentConvergeMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.LastAgentConvergeMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.LastAgentConvergeMS .SLIM.LastAgentConvergeMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Avg propagation (ms)</td><td>{{if .HasA2A}}{{.A2A.AvgPropagationMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.AvgPropagationMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.AvgPropagationMS .SLIM.AvgPropagationMS}}{{else}}—{{end}}</td></tr>
    <tr><td>P95 propagation (ms)</td><td>{{if .HasA2A}}{{.A2A.P95PropagationMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.P95PropagationMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.P95PropagationMS .SLIM.P95PropagationMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Stream RPC count</td><td>{{if .HasA2A}}{{.A2A.StreamRPCCount}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.StreamRPCCount}}{{else}}—{{end}}</td><td>—</td></tr>
    <tr><td>Epochs (ok / failed)</td><td>{{if .HasA2A}}{{.A2A.EpochsSucceeded}} / {{.A2A.EpochsFailed}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.EpochsSucceeded}} / {{.SLIM.EpochsFailed}}{{else}}—{{end}}</td><td>—</td></tr>
    <tr><td>Success</td><td>{{if .HasA2A}}{{.A2A.Success}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.Success}}{{else}}—{{end}}</td><td>—</td></tr>
  </table>
  {{end}}
  {{if .Sweep}}
  <h2>Sweep results</h2>
  <table>
    <tr><th>Scenario</th><th>Impl</th><th>Agents</th><th>Think ms</th><th>Payload B</th><th>Consensus wall</th><th>Avg propagation</th><th>P95 propagation</th><th>Stream RPCs</th><th>Epochs ok/failed</th></tr>
    {{range .Sweep}}
    <tr>
      <td>{{.ScenarioName}}</td><td>{{.Implementation}}</td><td>{{.Agents}}</td><td>{{.ThinkTimeMs}}</td><td>{{.PayloadBytes}}</td>
      <td>{{.ConsensusWallMS}}</td><td>{{.AvgPropagationMS}}</td><td>{{.P95PropagationMS}}</td><td>{{.StreamRPCCount}}</td><td>{{.EpochsSucceeded}}/{{.EpochsFailed}}</td>
    </tr>
    {{end}}
  </table>
  {{end}}

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
      (average and 95th percentile). SLIM is one multicast hop; A2A adds the extra relay hop, so it grows
      with agent count, message rate, and payload size.</dd>
    <dt>Stream RPC count</dt>
    <dd>Number of finding-carrying messages on the data path. For SLIM this equals findings emitted (one
      native multicast each). For A2A it is the relay deliveries, ≈ <code>findings × (N−1)</code>, because the
      hub re-sends every finding to all other agents.</dd>
    <dt>Epochs (ok / failed)</dt>
    <dd>Each run repeats the same consensus attempt over several <em>epochs</em>. An epoch succeeds if every
      agent reaches global consensus within <code>spec.maxEpochTimeMs</code> (each attempt runs up to
      <code>maxRounds</code>); otherwise it is counted as failed. This surfaces reliability differences: under
      load A2A may miss the per-epoch budget while SLIM still converges. Overall <em>Success</em> is true only
      when no epoch failed.</dd>
    <dt>Coord fanout (ms)</dt>
    <dd>Cumulative time the A2A relay hub spent fanning findings out to peers. <code>0</code> for SLIM because
      there is no relay — the dataplane does the fan-out.</dd>
    <dt>Payload B</dt>
    <dd>Fixed-size padding (<code>spec.payloadBytes</code>) added to every finding to stress transport
      bandwidth. Semantically inert; it does not change the consensus math or round count.</dd>
    <dt>Improvement vs SLIM</dt>
    <dd><code>(A2A − SLIM) / SLIM × 100</code>. Unbounded measure of how much more time/work A2A costs than
      SLIM; e.g. <code>+4100%</code> ≈ 42× slower. Positive favors SLIM. (Note: a2a-go's streaming task store
      also appends each relayed finding to task history and deep-copies per update, which amplifies A2A cost
      super-linearly at large payloads — a property of the A2A streaming/task model.)</dd>
  </dl>

  <h2 id="arch">Architecture</h2>
  <p>SLIM agents form a peer mesh over a native group session and the runner only observes; A2A routes every
    finding through the runner acting as a relay hub.</p>
  <div class="diagram">
    <pre class="mermaid">
flowchart TB
  subgraph slim["SLIM native group session — no relay"]
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
  subgraph a2a["A2A — runner is the relay hub"]
    R2["runner = RELAY (hosts A2A server)"]
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
  </div>

  <script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
    mermaid.initialize({ startOnLoad: true, securityLevel: 'strict' });
  </script>
</body>
</html>
`
