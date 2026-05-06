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
	FailureLocation string
}

func main() {
	reportsDir := flag.String("reports-dir", "./reports", "directory containing Ginkgo JSON and JUnit XML reports")
	outputPath := flag.String("output", "./reports/index.html", "path to the generated HTML dashboard")
	reportTitle := flag.String("title", "A2A Interop Dashboard", "Dashboard title (optional)")
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

	sort.Slice(reports, func(i int, j int) bool {
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
	if err := json.Unmarshal(content, &reports); err != nil {
		return nil, err
	}

	return reports, nil
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

	return specView{
		Name:            specDisplayName(spec),
		State:           displayState(spec.State),
		StateClass:      stateClass(spec.State),
		Duration:        formatDuration(time.Duration(spec.RunTime)),
		Labels:          strings.Join(collectLabels(spec), ", "),
		FailureMessage:  strings.TrimSpace(primaryFailure.Message),
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

func findSiblingXML(jsonFile string) string {
	xmlFile := strings.TrimSuffix(jsonFile, filepath.Ext(jsonFile)) + ".xml"
	if _, err := os.Stat(xmlFile); err == nil {
		return filepath.Base(xmlFile)
	}

	return ""
}

func pickSortTime(primary time.Time, fallback time.Time) time.Time {
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

func titleFromReport(name string, labelFilter string) string {
	trimmed := strings.TrimPrefix(name, "report-")
	trimmed = strings.TrimPrefix(trimmed, "agntcy-a2a")
	trimmed = strings.TrimPrefix(trimmed, "-")
	if trimmed == "" {
		if title := titleFromLabelFilter(labelFilter); title != "" {
			return title
		}
		return "A2A Interop Overview"
	}

	parts := strings.Split(trimmed, "-")
	pretty := make([]string, 0, len(parts))
	for _, part := range parts {
		pretty = append(pretty, prettyToken(part))
	}

	return strings.Join(pretty, " / ")
}

func titleFromLabelFilter(labelFilter string) string {
	for _, clause := range strings.Split(labelFilter, "&&") {
		clause = strings.TrimSpace(clause)
		if !strings.HasPrefix(clause, "suite-") {
			continue
		}

		suiteName := strings.TrimPrefix(clause, "suite-")
		if suiteName == "" {
			continue
		}

		parts := strings.Split(suiteName, "-")
		pretty := make([]string, 0, len(parts))
		for _, part := range parts {
			pretty = append(pretty, prettyToken(part))
		}

		return strings.Join(pretty, " / ")
	}

	return ""
}

func prettyToken(token string) string {
	switch token {
	case "a2a":
		return "A2A"
	case "go":
		return "Go"
	case "dotnet":
		return ".NET"
	case "grpc":
		return "gRPC"
	case "jsonrpc":
		return "JSON-RPC"
	case "python":
		return "Python"
	case "rest":
		return "REST"
	case "rust":
		return "Rust"
	case "behavior":
		return "Behavior"
	case "unary":
		return "Unary"
	case "streaming":
		return "Streaming"
	case "push":
		return "Push"
	case "config":
		return "Config"
	case "parity":
		return "Parity"
	case "lifecycle":
		return "Lifecycle"
	case "core":
		return "Core"
	default:
		if token == "" {
			return ""
		}
		return strings.ToUpper(token[:1]) + token[1:]
	}
}

func formatScope(total int, selected int) string {
	if total == 0 && selected == 0 {
		return "No specs recorded"
	}
	if selected == 0 {
		return fmt.Sprintf("%d total specs", total)
	}
	if selected == total {
		return fmt.Sprintf("Full run of %d specs", total)
	}

	return fmt.Sprintf("%d of %d specs selected", selected, total)
}

func normalizeState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

func displayState(state string) string {
	switch normalizeState(state) {
	case "passed":
		return "Passed"
	case "skipped":
		return "Skipped"
	case "pending":
		return "Pending"
	case "panicked":
		return "Panicked"
	case "timedout":
		return "Timed out"
	case "interrupted":
		return "Interrupted"
	case "aborted":
		return "Aborted"
	case "failed":
		return "Failed"
	default:
		trimmed := strings.TrimSpace(state)
		if trimmed == "" {
			return "Unknown"
		}
		return strings.ToUpper(trimmed[:1]) + trimmed[1:]
	}
}

func stateClass(state string) string {
	switch normalizeState(state) {
	case "passed":
		return "pass"
	case "skipped":
		return "skip"
	case "pending":
		return "pending"
	default:
		return "fail"
	}
}

func formatDuration(duration time.Duration) string {
	if duration <= 0 {
		return "0s"
	}
	if duration < time.Millisecond {
		return duration.Round(time.Microsecond).String()
	}
	return duration.Round(time.Millisecond).String()
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return "n/a"
	}
	return value.Local().Format("2006-01-02 15:04:05 MST")
}

func formatLocation(location failureLocation) string {
	if location.FileName == "" {
		return ""
	}

	workingDir, err := os.Getwd()
	path := location.FileName
	if err == nil {
		if relativePath, relErr := filepath.Rel(workingDir, location.FileName); relErr == nil && !strings.HasPrefix(relativePath, "..") {
			path = relativePath
		}
	}

	if location.LineNumber > 0 {
		return fmt.Sprintf("%s:%d", path, location.LineNumber)
	}

	return path
}

const dashboardTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.ReportTitle}}</title>
  <style>
    :root {
      --bg: #f4efe6;
      --bg-deep: #e9e1d4;
      --panel: rgba(255, 252, 247, 0.86);
      --panel-strong: rgba(255, 252, 247, 0.94);
      --ink: #17312a;
      --muted: #5f6f68;
      --line: rgba(23, 49, 42, 0.12);
      --accent: #0f766e;
      --accent-soft: #daf3ef;
      --warm: #b45309;
      --warm-soft: #fdebd8;
      --danger: #b42318;
      --danger-soft: #fde8e8;
      --shadow: 0 18px 60px rgba(15, 30, 24, 0.12);
      --font-sans: "Avenir Next", "Segoe UI", "Helvetica Neue", sans-serif;
      --font-serif: "Iowan Old Style", "Palatino Linotype", serif;
      --font-mono: "IBM Plex Mono", "SFMono-Regular", "Menlo", monospace;
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      min-height: 100vh;
      font-family: var(--font-sans);
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(15, 118, 110, 0.12), transparent 36%),
        radial-gradient(circle at top right, rgba(180, 83, 9, 0.12), transparent 32%),
        linear-gradient(180deg, var(--bg) 0%, var(--bg-deep) 100%);
    }

    .page {
      width: min(1380px, calc(100vw - 32px));
      margin: 0 auto;
      padding: 32px 0 56px;
    }

    .hero {
      background: linear-gradient(135deg, rgba(15, 118, 110, 0.12), rgba(255, 252, 247, 0.92) 55%, rgba(180, 83, 9, 0.08));
      border: 1px solid rgba(255, 255, 255, 0.6);
      border-radius: 28px;
      box-shadow: var(--shadow);
      padding: 28px;
      backdrop-filter: blur(18px);
    }

    .eyebrow {
      margin: 0 0 10px;
      font-size: 12px;
      letter-spacing: 0.18em;
      text-transform: uppercase;
      color: var(--muted);
    }

    h1 {
      margin: 0;
      font-family: var(--font-serif);
      font-size: clamp(32px, 5vw, 54px);
      line-height: 0.98;
      letter-spacing: -0.03em;
    }

    .lead {
      max-width: 860px;
      margin: 14px 0 0;
      color: var(--muted);
      font-size: 16px;
      line-height: 1.6;
    }

    .hero-meta {
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
      margin-top: 20px;
    }

    .meta-chip,
    .status-chip,
    .pill {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      border-radius: 999px;
      padding: 8px 12px;
      font-size: 13px;
      line-height: 1;
      border: 1px solid var(--line);
      background: rgba(255, 255, 255, 0.72);
    }

    .summary-strip {
      display: grid;
      grid-template-columns: repeat(6, minmax(0, 1fr));
      margin-top: 22px;
      background: var(--panel);
      border: 1px solid rgba(255, 255, 255, 0.72);
      border-radius: 24px;
      box-shadow: var(--shadow);
      overflow: hidden;
      backdrop-filter: blur(16px);
    }

    .summary-item {
      padding: 18px 20px;
      border-right: 1px solid var(--line);
    }

    .summary-item:last-child {
      border-right: 0;
    }

    .summary-item .label {
      display: block;
      font-size: 12px;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 10px;
    }

    .summary-item strong {
      display: block;
      font-size: 28px;
      line-height: 1;
      margin-bottom: 8px;
    }

    .summary-item span {
      color: var(--muted);
      font-size: 13px;
    }

    .toolbar {
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
      align-items: center;
      margin: 26px 0 18px;
    }

    .toolbar label {
      font-size: 13px;
      color: var(--muted);
    }

    .toolbar input,
    .toolbar select {
      min-width: 220px;
      border: 1px solid var(--line);
      border-radius: 999px;
      padding: 11px 14px;
      font: inherit;
      color: var(--ink);
      background: rgba(255, 255, 255, 0.82);
      box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4);
    }

    .report-log-shell {
      margin-top: 6px;
      background: var(--panel-strong);
      border: 1px solid rgba(255, 255, 255, 0.72);
      border-radius: 28px;
      box-shadow: var(--shadow);
      overflow: hidden;
      backdrop-filter: blur(16px);
    }

    .report-log-head,
    .report-row {
      display: grid;
      grid-template-columns: minmax(250px, 1.9fr) 120px minmax(220px, 1.35fr) 140px minmax(180px, 1.15fr) 110px;
      gap: 16px;
      align-items: start;
    }

    .report-log-head {
      padding: 16px 20px 14px 54px;
      background: rgba(23, 49, 42, 0.05);
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.12em;
      font-size: 11px;
    }

    .report-entry {
      border-top: 1px solid var(--line);
    }

    .report-entry:first-of-type {
      border-top: 0;
    }

    .report-entry > summary {
      cursor: pointer;
      list-style: none;
    }

    .report-entry > summary::-webkit-details-marker {
      display: none;
    }

    .report-row {
      position: relative;
      padding: 18px 20px 18px 54px;
    }

    .report-row::before {
      content: "+";
      position: absolute;
      left: 20px;
      top: 18px;
      width: 22px;
      height: 22px;
      display: grid;
      place-items: center;
      border-radius: 999px;
      border: 1px solid var(--line);
      background: rgba(255, 255, 255, 0.88);
      color: var(--muted);
      font-size: 16px;
      font-weight: 700;
      line-height: 1;
    }

    .report-entry[open] .report-row::before {
      content: "−";
      background: rgba(15, 118, 110, 0.1);
      color: var(--accent);
    }

    .report-entry.pass .report-row::before {
      border-color: rgba(15, 118, 110, 0.18);
    }

    .report-entry.fail .report-row::before {
      border-color: rgba(180, 35, 24, 0.18);
    }

    .report-entry[open].pass .report-row {
      background: linear-gradient(180deg, rgba(15, 118, 110, 0.04), rgba(255, 255, 255, 0));
    }

    .report-entry[open].fail .report-row {
      background: linear-gradient(180deg, rgba(180, 35, 24, 0.04), rgba(255, 255, 255, 0));
    }

    .report-cell {
      min-width: 0;
    }

    .cell-label {
      display: none;
      margin-bottom: 6px;
      font-size: 11px;
      letter-spacing: 0.11em;
      text-transform: uppercase;
      color: var(--muted);
    }

    .report-title {
      display: block;
      font-size: 18px;
      line-height: 1.15;
    }

    .report-subtitle {
      margin-top: 6px;
      font-size: 13px;
      line-height: 1.5;
      color: var(--muted);
    }

    .status-chip.pass {
      background: var(--accent-soft);
      color: var(--accent);
      border-color: rgba(15, 118, 110, 0.16);
    }

    .status-chip.fail {
      background: var(--danger-soft);
      color: var(--danger);
      border-color: rgba(180, 35, 24, 0.16);
    }

    .scope-stack {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }

    .pill code,
    .meta-chip code {
      font-family: var(--font-mono);
      font-size: 12px;
    }

    .timing strong {
      display: block;
      font-size: 18px;
      line-height: 1.1;
    }

    .metric-inline {
      display: flex;
      flex-wrap: wrap;
      gap: 8px 12px;
    }

    .metric-inline span {
      font-size: 13px;
      color: var(--muted);
      white-space: nowrap;
    }

    .metric-inline strong {
      color: var(--ink);
    }

    .report-links {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      align-content: flex-start;
    }

    .report-links a {
      color: var(--accent);
      text-decoration: none;
      font-size: 13px;
      font-weight: 600;
    }

    .report-links a:hover {
      text-decoration: underline;
    }

    .report-body {
      padding: 0 20px 22px 54px;
      border-top: 1px solid var(--line);
      background: rgba(255, 255, 255, 0.46);
    }

    .detail-grid {
      display: grid;
      grid-template-columns: minmax(280px, 0.9fr) minmax(0, 1.4fr);
      gap: 18px;
      margin-top: 18px;
    }

    .detail-column {
      display: flex;
      flex-direction: column;
      gap: 18px;
    }

    .detail-block {
      border: 1px solid var(--line);
      border-radius: 22px;
      background: rgba(255, 255, 255, 0.7);
      padding: 16px 18px;
    }

    .detail-block h3 {
      margin: 0 0 12px;
      font-size: 12px;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      color: var(--muted);
    }

    .detail-list {
      display: grid;
      gap: 10px;
      margin: 0;
    }

    .detail-pair {
      display: grid;
      gap: 4px;
    }

    .detail-pair dt {
      font-size: 11px;
      letter-spacing: 0.1em;
      text-transform: uppercase;
      color: var(--muted);
    }

    .detail-pair dd {
      margin: 0;
      font-size: 14px;
      line-height: 1.45;
    }

    .detail-links {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 14px;
    }

    .detail-links a {
      color: var(--accent);
      text-decoration: none;
      font-size: 13px;
      font-weight: 600;
    }

    .detail-links a:hover {
      text-decoration: underline;
    }

    .spec-table-wrap {
      overflow-x: auto;
      border: 1px solid var(--line);
      border-radius: 18px;
      background: rgba(255, 255, 255, 0.72);
    }

    table {
      width: 100%;
      border-collapse: collapse;
      min-width: 620px;
    }

    th,
    td {
      text-align: left;
      vertical-align: top;
      padding: 12px 14px;
      border-bottom: 1px solid var(--line);
      font-size: 13px;
    }

    th {
      position: sticky;
      top: 0;
      background: rgba(244, 239, 230, 0.96);
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.11em;
      font-size: 11px;
    }

    tr:last-child td {
      border-bottom: 0;
    }

    .spec-status {
      display: inline-flex;
      align-items: center;
      border-radius: 999px;
      padding: 6px 10px;
      font-size: 12px;
      font-weight: 600;
      white-space: nowrap;
    }

    .spec-status.pass {
      background: var(--accent-soft);
      color: var(--accent);
    }

    .spec-status.fail {
      background: var(--danger-soft);
      color: var(--danger);
    }

    .spec-status.skip,
    .spec-status.pending {
      background: rgba(95, 111, 104, 0.12);
      color: var(--muted);
    }

    .failure-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .failure-item {
      padding: 14px;
      border-radius: 18px;
      background: var(--danger-soft);
      border: 1px solid rgba(180, 35, 24, 0.12);
    }

    .failure-item strong {
      display: block;
      margin-bottom: 6px;
    }

    .failure-item code {
      display: inline-block;
      margin-top: 6px;
      font-family: var(--font-mono);
      font-size: 12px;
      color: var(--danger);
      word-break: break-word;
    }

    .empty-state {
      margin-top: 18px;
      padding: 26px;
      border-radius: 22px;
      background: rgba(255, 252, 247, 0.8);
      border: 1px dashed rgba(23, 49, 42, 0.24);
      color: var(--muted);
      text-align: center;
    }

    .footer-note {
      margin-top: 10px;
      color: var(--muted);
      font-size: 13px;
    }

    @media (max-width: 1100px) {
      .summary-strip {
        grid-template-columns: repeat(3, minmax(0, 1fr));
      }

      .report-log-head,
      .report-row {
        grid-template-columns: minmax(220px, 1.8fr) 120px minmax(180px, 1.3fr) 140px minmax(160px, 1fr) 110px;
      }

      .detail-grid {
        grid-template-columns: 1fr;
      }
    }

    @media (max-width: 760px) {
      .page {
        width: min(100vw - 20px, 100%);
        padding: 18px 0 34px;
      }

      .hero,
      .report-log-shell,
      .summary-strip,
      .detail-block {
        border-radius: 22px;
      }

      .summary-strip {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }

      .report-log-head {
        display: none;
      }

      .report-row {
        grid-template-columns: 1fr;
        gap: 12px;
        padding: 18px 16px 18px 52px;
      }

      .report-row::before {
        left: 16px;
      }

      .cell-label {
        display: block;
      }

      .report-body {
        padding: 0 16px 18px;
      }
    }
  </style>
</head>
<body>
  <main class="page">
    <section class="hero">
      <p class="eyebrow">AGNTCY CSIT</p>
      <h1>{{.ReportTitle}}</h1>
      <p class="lead">A static HTML view over the Ginkgo JSON and JUnit XML artifacts under {{.ReportDir}}. The dashboard groups every saved run, highlights failures, and keeps the raw machine-readable files one click away.</p>
      <div class="hero-meta">
        <span class="meta-chip">Rendered <code>{{.GeneratedAt}}</code></span>
        {{if .Summary.LatestRun}}<span class="meta-chip">Latest run <code>{{.Summary.LatestRun}}</code></span>{{end}}
        <span class="meta-chip">HTML artifact <code>reports/index.html</code></span>
      </div>
    </section>

    <section class="summary-strip">
      <article class="summary-item">
        <span class="label">Saved Runs</span>
        <strong>{{.Summary.Files}}</strong>
        <span>Parsed suite reports</span>
      </article>
      <article class="summary-item">
        <span class="label">Selected Specs</span>
        <strong>{{.Summary.SelectedSpecs}}</strong>
        <span>Across all saved runs</span>
      </article>
      <article class="summary-item">
        <span class="label">Executed Specs</span>
        <strong>{{.Summary.ExecutedSpecs}}</strong>
        <span>Passed plus failed specs</span>
      </article>
      <article class="summary-item">
        <span class="label">Passing Specs</span>
        <strong>{{.Summary.Passed}}</strong>
        <span>Successful assertions</span>
      </article>
      <article class="summary-item">
        <span class="label">Failing Specs</span>
        <strong>{{.Summary.Failed}}</strong>
        <span>Specs needing attention</span>
      </article>
      <article class="summary-item">
        <span class="label">Passing Reports</span>
        <strong>{{.Summary.PassingReports}}</strong>
        <span>{{.Summary.FailingReports}} failing reports</span>
      </article>
    </section>

    {{if .HasReports}}
    <section class="toolbar">
      <label>
        <span class="eyebrow">Search</span><br>
        <input type="search" data-role="search" placeholder="Filter by report name, label, or suite">
      </label>
      <label>
        <span class="eyebrow">Status</span><br>
        <select data-role="status">
          <option value="all">All reports</option>
          <option value="pass">Passing only</option>
          <option value="fail">Failing only</option>
        </select>
      </label>
    </section>

    <section class="report-log-shell">
      <div class="report-log-head">
        <span>Run</span>
        <span>Status</span>
        <span>Scope</span>
        <span>Timing</span>
        <span>Results</span>
        <span>Artifacts</span>
      </div>
      {{range .Reports}}
      <details class="report-entry {{.StatusClass}}" data-status="{{.StatusClass}}" data-search="{{.SearchText}}">
        <summary class="report-row">
          <div class="report-cell">
            <span class="cell-label">Run</span>
            <strong class="report-title">{{.Title}}</strong>
            <div class="report-subtitle">{{.Suite}}</div>
          </div>
          <div class="report-cell">
            <span class="cell-label">Status</span>
            <span class="status-chip {{.StatusClass}}">{{.Status}}</span>
          </div>
          <div class="report-cell">
            <span class="cell-label">Scope</span>
            <div class="scope-stack">
              <span class="pill">{{.Scope}}</span>
              {{if .LabelFilter}}<span class="pill"><code>{{.LabelFilter}}</code></span>{{end}}
            </div>
          </div>
          <div class="report-cell timing">
            <span class="cell-label">Timing</span>
            <strong>{{.Duration}}</strong>
            <div class="report-subtitle">Updated {{.UpdatedAt}}</div>
          </div>
          <div class="report-cell">
            <span class="cell-label">Results</span>
            <div class="metric-inline">
              <span><strong>{{.Passed}}</strong> passed</span>
              <span><strong>{{.Failed}}</strong> failed</span>
              <span><strong>{{.Skipped}}</strong> skipped</span>
              {{if gt .Pending 0}}<span><strong>{{.Pending}}</strong> pending</span>{{end}}
            </div>
          </div>
          <div class="report-cell">
            <span class="cell-label">Artifacts</span>
            <div class="report-links">
              <a href="{{.RawJSON}}">JSON</a>
              {{if .RawXML}}<a href="{{.RawXML}}">XML</a>{{end}}
            </div>
          </div>
        </summary>

        <div class="report-body">
          <div class="detail-grid">
            <div class="detail-column">
              <section class="detail-block">
                <h3>Run Metadata</h3>
                <dl class="detail-list">
                  <div class="detail-pair">
                    <dt>Saved run</dt>
                    <dd>{{.Name}}</dd>
                  </div>
                  <div class="detail-pair">
                    <dt>Started</dt>
                    <dd>{{.StartedAt}}</dd>
                  </div>
                  <div class="detail-pair">
                    <dt>Finished</dt>
                    <dd>{{.FinishedAt}}</dd>
                  </div>
                  <div class="detail-pair">
                    <dt>Selected specs</dt>
                    <dd>{{.Selected}} of {{.Total}}</dd>
                  </div>
                  <div class="detail-pair">
                    <dt>Executed specs</dt>
                    <dd>{{.Executed}}</dd>
                  </div>
                  {{if .LabelFilter}}
                  <div class="detail-pair">
                    <dt>Label filter</dt>
                    <dd><code>{{.LabelFilter}}</code></dd>
                  </div>
                  {{end}}
                </dl>
                <div class="detail-links">
                  <a href="{{.RawJSON}}">Open raw JSON</a>
                  {{if .RawXML}}<a href="{{.RawXML}}">Open raw XML</a>{{end}}
                </div>
              </section>

              <section class="detail-block">
                <h3>Failures</h3>
                {{if .FailedSpecs}}
                <div class="failure-list">
                  {{range .FailedSpecs}}
                  <div class="failure-item">
                    <strong>{{.Name}}</strong>
                    <div>{{.FailureMessage}}</div>
                    {{if .FailureLocation}}<code>{{.FailureLocation}}</code>{{end}}
                  </div>
                  {{end}}
                </div>
                {{else}}
                <p class="footer-note">No failures recorded for this run.</p>
                {{end}}
              </section>
            </div>

            <section class="detail-block">
              <h3>Executed Specs</h3>
              {{if .Specs}}
              <div class="spec-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Status</th>
                      <th>Spec</th>
                      <th>Duration</th>
                      <th>Labels</th>
                    </tr>
                  </thead>
                  <tbody>
                    {{range .Specs}}
                    <tr>
                      <td><span class="spec-status {{.StateClass}}">{{.State}}</span></td>
                      <td>
                        <div>{{.Name}}</div>
                        {{if .FailureMessage}}
                        <div class="footer-note">{{.FailureMessage}}</div>
                        {{end}}
                      </td>
                      <td>{{.Duration}}</td>
                      <td>{{.Labels}}</td>
                    </tr>
                    {{end}}
                  </tbody>
                </table>
              </div>
              {{else}}
              <p class="footer-note">No executed specs were captured for this report.</p>
              {{end}}
            </section>
          </div>
        </div>
      </details>
      {{end}}
    </section>

    <p class="empty-state is-filtered" hidden>No reports match the current filter.</p>
    {{else}}
    <section class="empty-state">
      <h2>No reports found</h2>
      <p>Run any A2A test task first, then refresh the dashboard with <code>task reports:dashboard</code>.</p>
    </section>
    {{end}}
  </main>

  <script>
    (function () {
      const entries = Array.from(document.querySelectorAll('.report-entry'));
      const searchInput = document.querySelector('[data-role="search"]');
      const statusSelect = document.querySelector('[data-role="status"]');
      const emptyState = document.querySelector('.is-filtered');

      if (!entries.length || !searchInput || !statusSelect) {
        return;
      }

      function applyFilters() {
        const query = searchInput.value.trim().toLowerCase();
        const status = statusSelect.value;
        let visibleCount = 0;

        entries.forEach((entry) => {
          const haystack = (entry.dataset.search || '').toLowerCase();
          const matchesQuery = !query || haystack.includes(query);
          const matchesStatus = status === 'all' || entry.dataset.status === status;
          const visible = matchesQuery && matchesStatus;

          entry.hidden = !visible;
          if (visible) {
            visibleCount += 1;
          }
        });

        if (emptyState) {
          emptyState.hidden = visibleCount !== 0;
        }
      }

      searchInput.addEventListener('input', applyFilters);
      statusSelect.addEventListener('change', applyFilters);
      applyFilters();
    })();
  </script>
</body>
</html>`
