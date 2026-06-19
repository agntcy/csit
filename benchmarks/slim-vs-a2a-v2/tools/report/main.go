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
	"strconv"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/metrics"
)

type scenarioComparison struct {
	ScenarioName string
	A2A          metrics.RunResult
	SLIM         metrics.RunResult
	HasA2A       bool
	HasSLIM      bool
}

func main() {
	tsvPath := flag.String("tsv", "./reports/results.tsv", "comparison results tsv")
	sweepTSV := flag.String("sweep-tsv", "./reports/sweep.tsv", "optional sweep results tsv")
	output := flag.String("output", "./reports/index.html", "html dashboard output")
	flag.Parse()

	results, err := readTSV(*tsvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read tsv: %v\n", err)
		os.Exit(1)
	}
	comparisons := groupByScenario(results)

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
		if len(row) < 16 {
			continue
		}
		r := metrics.RunResult{
			ScenarioName:   row[0],
			Domain:         row[1],
			Implementation: row[2],
			Error:          row[len(row)-1],
		}
		r.Agents = atoi(row[3])
		r.ThinkTimeMs = atoi64(row[4])
		r.ConsensusWallMS = atoi64(row[5])
		r.ConsensusRound = atoi(row[6])
		r.FindingsEmitted = atoi(row[7])
		r.FindingsReceivedTotal = atoi(row[8])
		r.AvgPropagationMS = atoi64(row[9])
		r.P95PropagationMS = atoi64(row[10])
		r.LastAgentConvergeMS = atoi64(row[11])
		r.CoordFanoutMS = atoi64(row[12])
		r.StreamRPCCount = atoi(row[13])
		r.UnicastRPCCount = atoi(row[14])
		r.Success = row[15] == "true"
		if len(row) > 16 {
			r.Error = row[16]
		}
		out = append(out, r)
	}
	return out, nil
}

func groupByScenario(results []metrics.RunResult) []scenarioComparison {
	byScenario := map[string]*scenarioComparison{}
	for _, r := range results {
		sc, ok := byScenario[r.ScenarioName]
		if !ok {
			sc = &scenarioComparison{ScenarioName: r.ScenarioName}
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

func deltaPct(a2a, slim int64) string {
	if a2a == 0 {
		return "n/a"
	}
	pct := (float64(a2a-slim) / float64(a2a)) * 100
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
  <title>SLIM vs A2A v2 Consensus</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; }
    table { border-collapse: collapse; width: 100%; margin-bottom: 2rem; }
    th, td { border: 1px solid #ccc; padding: 0.5rem 0.75rem; text-align: right; }
    th:first-child, td:first-child { text-align: left; }
    h2 { margin-top: 2rem; }
  </style>
</head>
<body>
  <h1>SLIM vs A2A v2 — Consensus Streaming</h1>
  <p>Delta columns show (A2A − SLIM) / A2A. Positive values mean SLIM reached consensus faster.</p>
  {{range .Comparisons}}
  <h2>{{.ScenarioName}}</h2>
  <table>
    <tr><th>Metric</th><th>A2A</th><th>SLIM</th><th>Delta</th></tr>
    <tr><td>Consensus wall (ms)</td><td>{{if .HasA2A}}{{.A2A.ConsensusWallMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.ConsensusWallMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.ConsensusWallMS .SLIM.ConsensusWallMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Last agent converge (ms)</td><td>{{if .HasA2A}}{{.A2A.LastAgentConvergeMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.LastAgentConvergeMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.LastAgentConvergeMS .SLIM.LastAgentConvergeMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Avg propagation (ms)</td><td>{{if .HasA2A}}{{.A2A.AvgPropagationMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.AvgPropagationMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.AvgPropagationMS .SLIM.AvgPropagationMS}}{{else}}—{{end}}</td></tr>
    <tr><td>P95 propagation (ms)</td><td>{{if .HasA2A}}{{.A2A.P95PropagationMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.P95PropagationMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.P95PropagationMS .SLIM.P95PropagationMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Stream RPC count</td><td>{{if .HasA2A}}{{.A2A.StreamRPCCount}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.StreamRPCCount}}{{else}}—{{end}}</td><td>—</td></tr>
    <tr><td>Unicast RPC count</td><td>{{if .HasA2A}}{{.A2A.UnicastRPCCount}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.UnicastRPCCount}}{{else}}—{{end}}</td><td>—</td></tr>
    <tr><td>Success</td><td>{{if .HasA2A}}{{.A2A.Success}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.Success}}{{else}}—{{end}}</td><td>—</td></tr>
  </table>
  {{end}}
  {{if .Sweep}}
  <h2>Sweep results</h2>
  <table>
    <tr><th>Scenario</th><th>Impl</th><th>Agents</th><th>Think ms</th><th>Consensus wall</th><th>P95 propagation</th><th>Stream RPCs</th></tr>
    {{range .Sweep}}
    <tr>
      <td>{{.ScenarioName}}</td><td>{{.Implementation}}</td><td>{{.Agents}}</td><td>{{.ThinkTimeMs}}</td>
      <td>{{.ConsensusWallMS}}</td><td>{{.P95PropagationMS}}</td><td>{{.StreamRPCCount}}</td>
    </tr>
    {{end}}
  </table>
  {{end}}
</body>
</html>
`
