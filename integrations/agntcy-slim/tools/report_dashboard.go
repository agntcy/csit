// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type suiteReportFile struct {
	SuiteDescription string       `json:"SuiteDescription"`
	SuiteSucceeded   bool         `json:"SuiteSucceeded"`
	PreRunStats      preRunStats  `json:"PreRunStats"`
	StartTime        time.Time    `json:"StartTime"`
	EndTime          time.Time    `json:"EndTime"`
	RunTime          int64        `json:"RunTime"`
	SuiteConfig      suiteConfig  `json:"SuiteConfig"`
	SpecReports      []specReport `json:"SpecReports"`
}

type preRunStats struct {
	TotalSpecs       int `json:"TotalSpecs"`
	SpecsThatWillRun int `json:"SpecsThatWillRun"`
}

type suiteConfig struct {
	LabelFilter string `json:"LabelFilter"`
	RandomSeed  int64  `json:"RandomSeed"`
}

type specReport struct {
	ContainerHierarchyTexts  []string   `json:"ContainerHierarchyTexts"`
	ContainerHierarchyLabels [][]string `json:"ContainerHierarchyLabels"`
	LeafNodeLabels           []string   `json:"LeafNodeLabels"`
	LeafNodeText             string     `json:"LeafNodeText"`
	State                    string     `json:"State"`
	RunTime                  int64      `json:"RunTime"`
	Failure                  failure    `json:"Failure"`
	AdditionalFailures       []failure  `json:"AdditionalFailures"`
}

type failure struct {
	Message  string          `json:"Message"`
	Location failureLocation `json:"Location"`
}

type failureLocation struct {
	FileName   string `json:"FileName"`
	LineNumber int    `json:"LineNumber"`
}

type dashboardView struct {
	GeneratedAt string
	ReportDir   string
	ReportTitle string
	HasReports  bool
	Summary     summaryView
	Reports     []reportView
}

type summaryView struct {
	Files          int
	TotalSpecs     int
	SelectedSpecs  int
	ExecutedSpecs  int
	Passed         int
	Failed         int
	Skipped        int
	Pending        int
	PassingReports int
	FailingReports int
	LatestRun      string
}

type reportView struct {
	Name        string
	Title       string
	Suite       string
	LabelFilter string
	Scope       string
	Status      string
	StatusClass string
	StartedAt   string
	FinishedAt  string
	UpdatedAt   string
	Duration    string
	Total       int
	Selected    int
	Executed    int
	Passed      int
	Failed      int
	Skipped     int
	Pending     int
	RawJSON     string
	RawXML      string
	FailedSpecs []specView
	Specs       []specView
	SearchText  string
	SortTime    time.Time
}

type specView struct {
	Name            string
	State           string
	StateClass      string
	Duration        string
	Labels          string
	FailureMessage  string
	FailureDetail   string
	FailureLocation string
}

func main() {
	reportsDir := flag.String("reports-dir", "./reports", "directory containing Ginkgo JSON reports (report-slim-*.json)")
	outputPath := flag.String("output", "./reports/index.html", "path to the generated HTML dashboard")
	reportTitle := flag.String("title", "Slim integration", "dashboard title")
	flag.Parse()

	view, err := buildDashboard(*reportTitle, *reportsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build dashboard: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}

	htmlOutput, err := renderDashboard(view)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render dashboard: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputPath, htmlOutput, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write dashboard: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s\n", *outputPath)
}

func buildDashboard(reportsTitle, reportsDir string) (dashboardView, error) {
	jsonFiles, err := filepath.Glob(filepath.Join(reportsDir, "*.json"))
	if err != nil {
		return dashboardView{}, fmt.Errorf("find report files: %w", err)
	}
	sort.Strings(jsonFiles)

	view := dashboardView{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05 MST"),
		ReportDir:   reportsDir,
		ReportTitle: reportsTitle,
	}

	reports := make([]reportView, 0, len(jsonFiles))
	latestRun := time.Time{}
	for _, jsonFile := range jsonFiles {
		suites, err := readSuiteReports(jsonFile)
		if err != nil {
			return dashboardView{}, fmt.Errorf("read %s: %w", jsonFile, err)
		}

		stat, err := os.Stat(jsonFile)
		if err != nil {
			return dashboardView{}, fmt.Errorf("stat %s: %w", jsonFile, err)
		}

		for index, suite := range suites {
			report := buildReportView(jsonFile, stat.ModTime(), suite, index)
			reports = append(reports, report)
			accumulateSummary(&view.Summary, report)
			if report.SortTime.After(latestRun) {
				latestRun = report.SortTime
			}
		}
	}

	sort.Slice(reports, func(i, j int) bool {
		if reports[i].SortTime.Equal(reports[j].SortTime) {
			return reports[i].Name < reports[j].Name
		}
		return reports[i].SortTime.After(reports[j].SortTime)
	})

	view.Reports = reports
	view.HasReports = len(reports) > 0
	if !latestRun.IsZero() {
		view.Summary.LatestRun = latestRun.Format("2006-01-02 15:04:05 MST")
	}

	return view, nil
}

func readSuiteReports(path string) ([]suiteReportFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var reports []suiteReportFile
	if err := json.Unmarshal(content, &reports); err == nil && len(reports) > 0 {
		return reports, nil
	}

	var single suiteReportFile
	if err := json.Unmarshal(content, &single); err != nil {
		return nil, fmt.Errorf("parse ginkgo json: %w", err)
	}
	return []suiteReportFile{single}, nil
}

func buildReportView(jsonFile string, updatedAt time.Time, suite suiteReportFile, index int) reportView {
	name := strings.TrimSuffix(filepath.Base(jsonFile), filepath.Ext(jsonFile))
	if index > 0 {
		name = fmt.Sprintf("%s-%d", name, index+1)
	}

	report := reportView{
		Name:        name,
		Title:       titleFromReport(name, suite.SuiteConfig.LabelFilter),
		Suite:       suite.SuiteDescription,
		LabelFilter: suite.SuiteConfig.LabelFilter,
		Scope:       formatScope(suite.PreRunStats.TotalSpecs, suite.PreRunStats.SpecsThatWillRun),
		StartedAt:   formatTimestamp(suite.StartTime),
		FinishedAt:  formatTimestamp(suite.EndTime),
		UpdatedAt:   formatTimestamp(updatedAt),
		Duration:    formatDuration(time.Duration(suite.RunTime)),
		Total:       suite.PreRunStats.TotalSpecs,
		Selected:    suite.PreRunStats.SpecsThatWillRun,
		RawJSON:     filepath.Base(jsonFile),
		RawXML:      findSiblingXML(jsonFile),
		SortTime:    pickSortTime(suite.EndTime, updatedAt),
	}

	labelsForSearch := []string{report.Title, report.Suite, report.LabelFilter, report.Name}
	for _, spec := range suite.SpecReports {
		state := normalizeState(spec.State)
		specView := buildSpecView(spec)

		switch state {
		case "passed":
			report.Passed++
			report.Executed++
			report.Specs = append(report.Specs, specView)
		case "skipped":
			report.Skipped++
		case "pending":
			report.Pending++
			report.Specs = append(report.Specs, specView)
		default:
			report.Failed++
			report.Executed++
			report.Specs = append(report.Specs, specView)
			report.FailedSpecs = append(report.FailedSpecs, specView)
		}

		labelsForSearch = append(labelsForSearch, specView.Name, specView.Labels)
	}

	if report.Failed > 0 || !suite.SuiteSucceeded {
		report.Status = "Failing"
		report.StatusClass = "fail"
	} else {
		report.Status = "Passing"
		report.StatusClass = "pass"
	}

	report.SearchText = strings.ToLower(strings.Join(labelsForSearch, " "))
	return report
}

func buildSpecView(spec specReport) specView {
	primaryFailure := spec.Failure
	if strings.TrimSpace(primaryFailure.Message) == "" && len(spec.AdditionalFailures) > 0 {
		primaryFailure = spec.AdditionalFailures[0]
	}

	fullMessage := strings.TrimSpace(primaryFailure.Message)
	return specView{
		Name:            specDisplayName(spec),
		State:           displayState(spec.State),
		StateClass:      stateClass(spec.State),
		Duration:        formatDuration(time.Duration(spec.RunTime)),
		Labels:          strings.Join(collectLabels(spec), ", "),
		FailureMessage:  summarizeFailure(fullMessage),
		FailureDetail:   fullMessage,
		FailureLocation: formatLocation(primaryFailure.Location),
	}
}

func accumulateSummary(summary *summaryView, report reportView) {
	summary.Files++
	summary.TotalSpecs += report.Total
	summary.SelectedSpecs += report.Selected
	summary.ExecutedSpecs += report.Executed
	summary.Passed += report.Passed
	summary.Failed += report.Failed
	summary.Skipped += report.Skipped
	summary.Pending += report.Pending
	if report.StatusClass == "fail" {
		summary.FailingReports++
	} else {
		summary.PassingReports++
	}
}

func renderDashboard(view dashboardView) ([]byte, error) {
	tmpl, err := template.New("dashboard").Parse(dashboardTemplate)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, view); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func titleFromReport(name, labelFilter string) string {
	trimmed := strings.TrimPrefix(name, "report-slim-")
	trimmed = strings.TrimPrefix(trimmed, "report-")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		if labelFilter != "" {
			return labelFilter
		}
		return "Slim test run"
	}
	return prettySlimSlug(trimmed)
}

func prettySlimSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		switch part {
		case "mcp":
			parts[i] = "MCP"
		case "slim":
			parts[i] = "Slim"
		default:
			if part != "" {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
	}
	return strings.Join(parts, " ")
}

func findSiblingXML(jsonFile string) string {
	xmlFile := strings.TrimSuffix(jsonFile, filepath.Ext(jsonFile)) + ".xml"
	if _, err := os.Stat(xmlFile); err == nil {
		return filepath.Base(xmlFile)
	}
	return ""
}

func pickSortTime(primary, fallback time.Time) time.Time {
	if !primary.IsZero() {
		return primary
	}
	return fallback
}

func specDisplayName(spec specReport) string {
	parts := append([]string{}, spec.ContainerHierarchyTexts...)
	if spec.LeafNodeText != "" {
		parts = append(parts, spec.LeafNodeText)
	}
	return strings.Join(parts, " / ")
}

func collectLabels(spec specReport) []string {
	labels := make([]string, 0, len(spec.LeafNodeLabels)+4)
	seen := map[string]bool{}
	for _, group := range spec.ContainerHierarchyLabels {
		for _, label := range group {
			if label == "" || seen[label] {
				continue
			}
			seen[label] = true
			labels = append(labels, label)
		}
	}
	for _, label := range spec.LeafNodeLabels {
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		labels = append(labels, label)
	}
	return labels
}

func summarizeFailure(msg string) string {
	if msg == "" {
		return ""
	}
	lines := strings.Split(msg, "\n")
	var summary string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "at ") {
			continue
		}
		summary = line
		break
	}
	if summary == "" {
		summary = strings.TrimSpace(lines[0])
	}
	if len(summary) > 400 {
		summary = summary[:400] + "…"
	}
	return summary
}

func normalizeState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

func displayState(state string) string {
	switch normalizeState(state) {
	case "passed":
		return "Passed"
	case "failed":
		return "Failed"
	case "skipped":
		return "Skipped"
	case "pending":
		return "Pending"
	default:
		if state == "" {
			return "Unknown"
		}
		return state
	}
}

func stateClass(state string) string {
	switch normalizeState(state) {
	case "passed":
		return "pass"
	case "failed":
		return "fail"
	case "skipped":
		return "skip"
	case "pending":
		return "pending"
	default:
		return "unknown"
	}
}

func formatScope(total, selected int) string {
	if total == selected {
		return fmt.Sprintf("%d specs", total)
	}
	return fmt.Sprintf("%d selected / %d total", selected, total)
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.Format("2006-01-02 15:04:05 MST")
}

func formatDuration(value time.Duration) string {
	if value <= 0 {
		return "—"
	}
	if value < time.Second {
		return fmt.Sprintf("%dms", value.Milliseconds())
	}
	return value.Round(time.Millisecond).String()
}

func formatLocation(location failureLocation) string {
	if location.FileName == "" {
		return ""
	}
	if location.LineNumber > 0 {
		return fmt.Sprintf("%s:%d", location.FileName, location.LineNumber)
	}
	return location.FileName
}

const dashboardTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.ReportTitle}}</title>
  <style>
    :root {
      --bg: #f4f7fb;
      --panel: #ffffff;
      --text: #1e293b;
      --muted: #64748b;
      --accent: #2563eb;
      --pass: #15803d;
      --fail: #b91c1c;
      --skip: #a16207;
      --border: #e2e8f0;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: ui-sans-serif, system-ui, sans-serif;
      color: var(--text);
      background: linear-gradient(180deg, #eef4ff 0%, var(--bg) 100%);
      padding: 32px 20px 48px;
    }
    main { max-width: 1100px; margin: 0 auto; }
    h1 { margin: 0 0 8px; font-size: 2rem; }
    .meta { color: var(--muted); margin-bottom: 24px; }
    .cards {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      gap: 12px;
      margin-bottom: 28px;
    }
    .card {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 16px;
    }
    .card strong { display: block; font-size: 1.5rem; }
    .card span { color: var(--muted); font-size: 0.85rem; }
    .toolbar {
      display: flex;
      gap: 16px;
      flex-wrap: wrap;
      margin-bottom: 16px;
    }
    .toolbar input, .toolbar select {
      padding: 8px 10px;
      border: 1px solid var(--border);
      border-radius: 8px;
      min-width: 220px;
    }
    details.report {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 12px;
      margin-bottom: 12px;
      overflow: hidden;
    }
    details.report.fail { border-color: #fecaca; }
    summary {
      cursor: pointer;
      list-style: none;
      padding: 16px 18px;
      display: grid;
      grid-template-columns: 2fr 1fr 1fr 1fr;
      gap: 12px;
      align-items: center;
    }
    summary::-webkit-details-marker { display: none; }
    .status-chip {
      display: inline-block;
      padding: 4px 10px;
      border-radius: 999px;
      font-size: 0.8rem;
      font-weight: 600;
    }
    .status-chip.pass { background: #dcfce7; color: var(--pass); }
    .status-chip.fail { background: #fee2e2; color: var(--fail); }
    .subtitle { color: var(--muted); font-size: 0.9rem; }
    .spec-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.92rem;
    }
    .spec-table th, .spec-table td {
      border-top: 1px solid var(--border);
      padding: 10px 12px;
      text-align: left;
      vertical-align: top;
    }
    .spec-table th { background: #f8fafc; color: var(--muted); }
    .state.pass { color: var(--pass); font-weight: 600; }
    .state.fail { color: var(--fail); font-weight: 600; }
    .state.skip { color: var(--skip); }
    pre.failure {
      margin: 8px 0 0;
      padding: 10px;
      background: #fff7ed;
      border-radius: 8px;
      white-space: pre-wrap;
      font-size: 0.82rem;
    }
    .empty {
      background: var(--panel);
      border: 1px dashed var(--border);
      border-radius: 12px;
      padding: 32px;
      text-align: center;
      color: var(--muted);
    }
    @media (max-width: 800px) {
      summary { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <main>
    <h1>{{.ReportTitle}}</h1>
    <p class="meta">Generated {{.GeneratedAt}}{{if .Summary.LatestRun}} · Latest run {{.Summary.LatestRun}}{{end}}</p>

    {{if .HasReports}}
    <section class="cards">
      <div class="card"><strong>{{.Summary.Files}}</strong><span>report files</span></div>
      <div class="card"><strong>{{.Summary.Passed}}</strong><span>specs passed</span></div>
      <div class="card"><strong>{{.Summary.Failed}}</strong><span>specs failed</span></div>
      <div class="card"><strong>{{.Summary.Skipped}}</strong><span>specs skipped</span></div>
      <div class="card"><strong>{{.Summary.PassingReports}}</strong><span>passing reports</span></div>
      <div class="card"><strong>{{.Summary.FailingReports}}</strong><span>failing reports</span></div>
    </section>

    <section class="toolbar">
      <input type="search" id="search" placeholder="Filter reports and specs">
      <select id="status">
        <option value="all">All reports</option>
        <option value="pass">Passing only</option>
        <option value="fail">Failing only</option>
      </select>
    </section>

    {{range .Reports}}
    <details class="report {{.StatusClass}}" data-status="{{.StatusClass}}" data-search="{{.SearchText}}">
      <summary>
        <div>
          <strong>{{.Title}}</strong>
          <div class="subtitle">{{.Suite}}</div>
        </div>
        <div><span class="status-chip {{.StatusClass}}">{{.Status}}</span></div>
        <div>
          <div>{{.Passed}} passed · {{.Failed}} failed</div>
          <div class="subtitle">{{.Scope}}</div>
        </div>
        <div>
          <div>{{.Duration}}</div>
          <div class="subtitle">{{.RawJSON}}{{if .RawXML}} · {{.RawXML}}{{end}}</div>
        </div>
      </summary>
      <table class="spec-table">
        <thead>
          <tr><th>Spec</th><th>State</th><th>Duration</th><th>Details</th></tr>
        </thead>
        <tbody>
          {{range .Specs}}
          <tr>
            <td>{{.Name}}</td>
            <td class="state {{.StateClass}}">{{.State}}</td>
            <td>{{.Duration}}</td>
            <td>
              {{if .FailureMessage}}<div>{{.FailureMessage}}</div>{{end}}
              {{if .FailureLocation}}<div class="subtitle">{{.FailureLocation}}</div>{{end}}
              {{if .FailureDetail}}<pre class="failure">{{.FailureDetail}}</pre>{{end}}
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </details>
    {{end}}
    {{else}}
    <div class="empty">No Ginkgo JSON reports found in {{.ReportDir}}.</div>
    {{end}}
  </main>
  <script>
    const search = document.getElementById('search');
    const status = document.getElementById('status');
    function applyFilters() {
      const q = (search?.value || '').toLowerCase();
      const st = status?.value || 'all';
      document.querySelectorAll('details.report').forEach((el) => {
        const matchesSearch = !q || (el.dataset.search || '').includes(q);
        const matchesStatus = st === 'all' || el.dataset.status === st;
        el.style.display = matchesSearch && matchesStatus ? '' : 'none';
      });
    }
    search?.addEventListener('input', applyFilters);
    status?.addEventListener('change', applyFilters);
  </script>
</body>
</html>`
