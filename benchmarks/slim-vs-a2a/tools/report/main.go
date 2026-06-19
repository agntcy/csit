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

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
)

type planComparison struct {
	PlanName string
	A2A      metrics.RunResult
	SLIM     metrics.RunResult
	HasA2A   bool
	HasSLIM  bool
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
	comparisons := groupByPlan(results)

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
		if len(row) < 22 {
			continue
		}
		r := metrics.RunResult{
			PlanName:       row[0],
			Domain:         row[1],
			Implementation: row[2],
			Error:          row[len(row)-1],
		}
		r.Agents = atoi(row[3])
		r.Tasks = atoi(row[4])
		r.TotalWallClockMS = atoi64(row[5])
		r.TasksCompleted = atoi(row[6])
		r.TasksFailed = atoi(row[7])
		r.TasksTimedOut = atoi(row[8])
		r.TasksCancelled = atoi(row[9])
		r.ObsoleteTasksCompleted = atoi(row[10])
		r.RetriesAttempted = atoi(row[11])
		r.RetriesSucceeded = atoi(row[12])
		r.ContextPushMS = atoi64(row[13])
		r.SyncBarrierMS = atoi64(row[14])
		r.CancelPropagationMS = atoi64(row[15])
		r.ExecuteRPCCount = atoi(row[16])
		r.MulticastRPCCount = atoi(row[17])
		r.SequentialRPCCount = atoi(row[18])
		r.MakespanMS = atoi64(row[19])
		if len(row) >= 29 {
			r.ContextPushP95MS = atoi64(row[20])
			r.ContextPushOps = atoi(row[21])
			r.CoordMissingResponses = atoi(row[22])
			r.CoordDeadlineMisses = atoi(row[23])
			r.CoordBytesSent = atoi64(row[24])
			r.CoordTimeSharePct, _ = strconv.ParseFloat(row[25], 64)
			r.RoundBudgetMS = atoi64(row[26])
			r.Success = row[27] == "true"
			r.Error = row[28]
		} else {
			r.Success = row[20] == "true"
			if len(row) > 21 {
				r.Error = row[21]
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func groupByPlan(results []metrics.RunResult) []planComparison {
	byPlan := map[string]*planComparison{}
	for _, r := range results {
		pc, ok := byPlan[r.PlanName]
		if !ok {
			pc = &planComparison{PlanName: r.PlanName}
			byPlan[r.PlanName] = pc
		}
		switch r.Implementation {
		case "a2a-grpc":
			pc.A2A = r
			pc.HasA2A = true
		case "slim-multicast", "slim-unicast":
			if !pc.HasSLIM || r.Implementation == "slim-multicast" {
				pc.SLIM = r
				pc.HasSLIM = true
			}
		}
	}
	out := make([]planComparison, 0, len(byPlan))
	for _, pc := range byPlan {
		out = append(out, *pc)
	}
	return out
}

type reportData struct {
	Comparisons []planComparison
	Sweep       []metrics.RunResult
}

func writeHTML(path string, comparisons []planComparison, sweep []metrics.RunResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmpl := template.Must(template.New("report").Funcs(template.FuncMap{
		"deltaPct":    deltaPct,
		"deltaPctInt": deltaPctInt,
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

func deltaPctInt(a2a, slim int) string {
	return deltaPct(int64(a2a), int64(slim))
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
  <title>SLIM vs A2A Comparison</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; }
    table { border-collapse: collapse; width: 100%; margin-bottom: 2rem; }
    th, td { border: 1px solid #ccc; padding: 0.5rem 0.75rem; text-align: right; }
    th:first-child, td:first-child { text-align: left; }
    h2 { margin-top: 2rem; }
  </style>
</head>
<body>
  <h1>SLIM vs A2A DAG Comparison</h1>
  <p>Delta columns show (A2A − SLIM) / A2A. Positive values mean SLIM was faster or used fewer RPCs.</p>
  {{range .Comparisons}}
  <h2>{{.PlanName}}</h2>
  <table>
    <tr>
      <th>Metric</th>
      <th>A2A</th>
      <th>SLIM</th>
      <th>Delta</th>
    </tr>
    <tr><td>Wall clock (ms)</td><td>{{if .HasA2A}}{{.A2A.TotalWallClockMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.TotalWallClockMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.TotalWallClockMS .SLIM.TotalWallClockMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Context push (ms)</td><td>{{if .HasA2A}}{{.A2A.ContextPushMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.ContextPushMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.ContextPushMS .SLIM.ContextPushMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Context push p95 (ms)</td><td>{{if .HasA2A}}{{.A2A.ContextPushP95MS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.ContextPushP95MS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.ContextPushP95MS .SLIM.ContextPushP95MS}}{{else}}—{{end}}</td></tr>
    <tr><td>Missing responses</td><td>{{if .HasA2A}}{{.A2A.CoordMissingResponses}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.CoordMissingResponses}}{{else}}—{{end}}</td><td>—</td></tr>
    <tr><td>Deadline misses</td><td>{{if .HasA2A}}{{.A2A.CoordDeadlineMisses}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.CoordDeadlineMisses}}{{else}}—{{end}}</td><td>—</td></tr>
    <tr><td>Coord bytes sent</td><td>{{if .HasA2A}}{{.A2A.CoordBytesSent}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.CoordBytesSent}}{{else}}—{{end}}</td><td>—</td></tr>
    <tr><td>Cancel propagation (ms)</td><td>{{if .HasA2A}}{{.A2A.CancelPropagationMS}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.CancelPropagationMS}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPct .A2A.CancelPropagationMS .SLIM.CancelPropagationMS}}{{else}}—{{end}}</td></tr>
    <tr><td>Sequential RPC count</td><td>{{if .HasA2A}}{{.A2A.SequentialRPCCount}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.SequentialRPCCount}}{{else}}—{{end}}</td><td>{{if and .HasA2A .HasSLIM}}{{deltaPctInt .A2A.SequentialRPCCount .SLIM.SequentialRPCCount}}{{else}}—{{end}}</td></tr>
    <tr><td>Multicast RPC count</td><td>{{if .HasA2A}}{{.A2A.MulticastRPCCount}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.MulticastRPCCount}}{{else}}—{{end}}</td><td>—</td></tr>
    <tr><td>Success</td><td>{{if .HasA2A}}{{.A2A.Success}}{{else}}—{{end}}</td><td>{{if .HasSLIM}}{{.SLIM.Success}}{{else}}—{{end}}</td><td>—</td></tr>
  </table>
  {{end}}
  {{if .Sweep}}
  <h2>Sweep results</h2>
  <table>
    <tr>
      <th>Plan</th><th>Impl</th><th>Agents</th><th>Budget ms</th><th>Ctx p95</th><th>Missing</th><th>Deadline misses</th><th>Bytes</th>
    </tr>
    {{range .Sweep}}
    <tr>
      <td>{{.PlanName}}</td><td>{{.Implementation}}</td><td>{{.Agents}}</td><td>{{.RoundBudgetMS}}</td>
      <td>{{.ContextPushP95MS}}</td><td>{{.CoordMissingResponses}}</td><td>{{.CoordDeadlineMisses}}</td><td>{{.CoordBytesSent}}</td>
    </tr>
    {{end}}
  </table>
  {{end}}
</body>
</html>
`
