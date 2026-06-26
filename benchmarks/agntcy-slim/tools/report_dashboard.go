package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	rendererhtml "github.com/yuin/goldmark/renderer/html"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

const (
	confidenceIntervalAlpha              = 0.05
	defaultMinimumConfidenceIntervalRuns = 20
)

var configuredMinimumConfidenceIntervalRuns = defaultMinimumConfidenceIntervalRuns

var benchmarkModeOrder = []string{"request-reply", "fire-and-forget", "write"}

type benchmarkModeSpec struct {
	Name                    string
	Title                   string
	ObservedThroughputLabel string
	UsesSinkMetrics         bool
}

var benchmarkModeSpecs = map[string]benchmarkModeSpec{
	"request-reply": {
		Name:                    "request-reply",
		Title:                   "Request-Reply",
		ObservedThroughputLabel: "Observed Node Throughput",
		UsesSinkMetrics:         true,
	},
	"fire-and-forget": {
		Name:                    "fire-and-forget",
		Title:                   "Fire-And-Forget",
		ObservedThroughputLabel: "Observed Node Throughput",
		UsesSinkMetrics:         true,
	},
	"write": {
		Name:                    "write",
		Title:                   "Write",
		ObservedThroughputLabel: "Sender Write Throughput",
		UsesSinkMetrics:         false,
	},
}

type benchmarkRow struct {
	Mode                 string
	Clients              int
	Size                 int
	Rate                 int
	Repeat               int
	SenderTotalMessages  int64
	SenderMPS            float64
	SenderMeanLatencyMS  float64
	SenderP50LatencyMS   float64
	SenderP99LatencyMS   float64
	SenderRuntimeErrors  int64
	SinkReceivedMessages int64
	SinkErrors           int64
	SinkActiveReceiveMPS float64
	NodeCPUPercent       float64
	TotalCPUPercent      float64
}

type slimRepoCSVRow struct {
	Senders           int
	MessagesPerSender int
	PayloadBytes      int
	TotalMessages     int64
	Run               int
	SendElapsedS      float64
	TotalElapsedS     float64
	TotalReceived     int64
	SendMPS           float64
	RecvMPS           float64
}

type sampleStats struct {
	Count    int
	Mean     float64
	Variance float64
	CILow    float64
	CIHigh   float64
}

type summaryCardView struct {
	Label  string
	Value  string
	Detail string
}

type tableView struct {
	Title   string
	Columns []string
	Rows    [][]string
}

type documentView struct {
	Title string
	Path  string
	HTML  template.HTML
	Open  bool
}

type artifactLinkView struct {
	Label string
	Path  string
}

type sectionView struct {
	ID          string
	Title       string
	Description string
	Cards       []summaryCardView
	Tables      []tableView
	Documents   []documentView
	Artifacts   []artifactLinkView
}

type dashboardView struct {
	GeneratedAt string
	Sections    []sectionView
	HasSections bool
}

func main() {
	smokeDir := flag.String("smoke-dir", "", "directory containing the benchmark smoke report bundle")
	capacityDir := flag.String("capacity-dir", "", "directory containing the benchmark capacity report bundle")
	slimCSV := flag.String("slim-csv", "", "path to an optional agntcy/slim benchmark-results.csv file")
	basicCSV := flag.String("basic-csv", "", "path to a CSIT-produced basic-benchmark-results.csv file")
	minCIRuns := flag.Int("min-ci-runs", defaultMinimumConfidenceIntervalRuns, "minimum repeated runs required before confidence intervals are shown")
	outputPath := flag.String("output", "./reports/index.html", "path to the generated HTML dashboard")
	flag.Parse()
	if *minCIRuns > 0 {
		configuredMinimumConfidenceIntervalRuns = *minCIRuns
	}

	view, err := buildDashboard(*smokeDir, *capacityDir, *slimCSV, *basicCSV, *outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build dashboard: %v\n", err)
		os.Exit(1)
	}

	htmlOutput, err := renderDashboard(view)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render dashboard: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputPath, htmlOutput, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write dashboard: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %s\n", *outputPath)
}

func buildDashboard(smokeDir string, capacityDir string, slimCSV string, basicCSV string, outputPath string) (dashboardView, error) {
	outputDir := filepath.Dir(outputPath)
	view := dashboardView{GeneratedAt: time.Now().Format("2006-01-02 15:04:05 MST")}

	if section, ok, err := buildSmokeSection(smokeDir, outputDir); err != nil {
		return dashboardView{}, err
	} else if ok {
		view.Sections = append(view.Sections, section)
	}

	if section, ok, err := buildCapacitySection(capacityDir, outputDir); err != nil {
		return dashboardView{}, err
	} else if ok {
		view.Sections = append(view.Sections, section)
	}

	if section, ok, err := buildBasicBenchmarkSection(basicCSV, outputDir); err != nil {
		return dashboardView{}, err
	} else if ok {
		view.Sections = append(view.Sections, section)
	}

	if section, ok, err := buildSlimRepoSection(slimCSV, outputDir); err != nil {
		return dashboardView{}, err
	} else if ok {
		view.Sections = append(view.Sections, section)
	}

	view.HasSections = len(view.Sections) > 0
	return view, nil
}

func buildSmokeSection(smokeDir string, outputDir string) (sectionView, bool, error) {
	if smokeDir == "" {
		return sectionView{}, false, nil
	}
	if _, err := os.Stat(smokeDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return sectionView{}, false, nil
		}
		return sectionView{}, false, err
	}

	rows, err := readBenchmarkTSV(filepath.Join(smokeDir, "results.tsv"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return sectionView{}, false, err
	}

	section := sectionView{
		ID:          "smoke-suite",
		Title:       "CSIT SLIM Smoke Suite",
		Description: "Repeated benchmark results for the CSIT smoke matrix across request-reply, fire-and-forget, and write workloads.",
		Cards:       buildSmokeCards(rows),
		Tables:      buildModeTables(rows),
		Artifacts:   collectArtifactLinks(smokeDir, outputDir, []string{"index.html", "results.tsv", "suite_summary.md", "technical_report.md", "ci-smoke-report.md", "ci-smoke.log"}),
	}
	section.Documents, err = collectDocuments(smokeDir, outputDir, []documentSpec{{Title: "Suite Summary", FileName: "suite_summary.md", Open: true}, {Title: "Technical Report", FileName: "technical_report.md", Open: false}, {Title: "CI Smoke Report", FileName: "ci-smoke-report.md", Open: false}})
	if err != nil {
		return sectionView{}, false, err
	}

	if len(rows) == 0 && len(section.Documents) == 0 && len(section.Artifacts) == 0 {
		return sectionView{}, false, nil
	}
	return section, true, nil
}

func buildCapacitySection(capacityDir string, outputDir string) (sectionView, bool, error) {
	if capacityDir == "" {
		return sectionView{}, false, nil
	}
	if _, err := os.Stat(capacityDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return sectionView{}, false, nil
		}
		return sectionView{}, false, err
	}

	rows, err := readCapacityRows(capacityDir)
	if err != nil {
		return sectionView{}, false, err
	}

	section := sectionView{
		ID:          "capacity-suite",
		Title:       "CSIT SLIM Capacity Sweeps",
		Description: "Adaptive capacity sweeps across sink-backed and write workloads, rendered from the published markdown reports and per-mode TSV samples.",
		Cards:       buildCapacityCards(rows),
		Tables:      buildModeTables(rows),
		Artifacts:   collectArtifactLinks(capacityDir, outputDir, []string{"index.html", "capacity_sweep.md", "capacity-fire-and-forget.md", "capacity-request-reply.md", "capacity-write.md", "results-fire-and-forget.tsv", "results-request-reply.tsv", "results-write.tsv", "ci-capacity-report.md", "ci-capacity.log"}),
	}
	section.Documents, err = collectDocuments(capacityDir, outputDir, []documentSpec{{Title: "Adaptive Capacity Sweep", FileName: "capacity_sweep.md", Open: true}, {Title: "Fire-And-Forget Capacity", FileName: "capacity-fire-and-forget.md", Open: false}, {Title: "Request-Reply Capacity", FileName: "capacity-request-reply.md", Open: false}, {Title: "Write Capacity", FileName: "capacity-write.md", Open: false}, {Title: "CI Capacity Report", FileName: "ci-capacity-report.md", Open: false}})
	if err != nil {
		return sectionView{}, false, err
	}

	if len(rows) == 0 && len(section.Documents) == 0 && len(section.Artifacts) == 0 {
		return sectionView{}, false, nil
	}
	return section, true, nil
}

func buildSlimRepoSection(slimCSV string, outputDir string) (sectionView, bool, error) {
	if slimCSV == "" {
		return sectionView{}, false, nil
	}
	rows, err := readSlimRepoCSV(slimCSV)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return sectionView{}, false, nil
		}
		return sectionView{}, false, err
	}
	if len(rows) == 0 {
		return sectionView{}, false, nil
	}

	artifactPath, err := relativePath(outputDir, slimCSV)
	if err != nil {
		return sectionView{}, false, err
	}

	section := sectionView{
		ID:          "slim-repo",
		Title:       "SLIM Repo Data-Plane Benchmark",
		Description: "Optional in-process throughput results imported from the agntcy/slim benchmark workflow for side-by-side visibility on the same Pages site.",
		Cards:       buildSlimRepoCards(rows),
		Tables:      []tableView{buildSlimRepoTable(rows)},
		Artifacts:   []artifactLinkView{{Label: "benchmark-results.csv", Path: artifactPath}},
	}
	return section, true, nil
}

func buildBasicBenchmarkSection(basicCSV string, outputDir string) (sectionView, bool, error) {
	if basicCSV == "" {
		return sectionView{}, false, nil
	}
	rows, err := readSlimRepoCSV(basicCSV)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return sectionView{}, false, nil
		}
		return sectionView{}, false, err
	}
	if len(rows) == 0 {
		return sectionView{}, false, nil
	}

	artifactPath, err := relativePath(outputDir, basicCSV)
	if err != nil {
		return sectionView{}, false, err
	}

	section := sectionView{
		ID:          "basic-benchmark",
		Title:       "SLIM Basic Throughput Benchmark",
		Description: "CSIT-measured baseline throughput results. N senders publish to a single sink over a local SLIM node, with unlimited rate, reproducing the agntcy/slim stress_publish benchmark topology.",
		Cards:       buildSlimRepoCards(rows),
		Tables:      []tableView{buildSlimRepoTable(rows)},
		Artifacts:   []artifactLinkView{{Label: "basic-benchmark-results.csv", Path: artifactPath}},
	}
	return section, true, nil
}

type documentSpec struct {
	Title    string
	FileName string
	Open     bool
}

func collectDocuments(baseDir string, outputDir string, specs []documentSpec) ([]documentView, error) {
	docs := make([]documentView, 0, len(specs))
	for _, spec := range specs {
		path := filepath.Join(baseDir, spec.FileName)
		content, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		relPath, err := relativePath(outputDir, path)
		if err != nil {
			return nil, err
		}
		htmlContent, err := markdownToHTML(content)
		if err != nil {
			return nil, err
		}
		docs = append(docs, documentView{Title: spec.Title, Path: relPath, HTML: htmlContent, Open: spec.Open})
	}
	return docs, nil
}

func markdownToHTML(content []byte) (template.HTML, error) {
	engine := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(rendererhtml.WithUnsafe()),
	)
	var buffer bytes.Buffer
	if err := engine.Convert(content, &buffer); err != nil {
		return "", err
	}
	return template.HTML(buffer.String()), nil
}

func collectArtifactLinks(baseDir string, outputDir string, files []string) []artifactLinkView {
	links := make([]artifactLinkView, 0, len(files))
	for _, name := range files {
		path := filepath.Join(baseDir, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		relPath, err := relativePath(outputDir, path)
		if err != nil {
			continue
		}
		links = append(links, artifactLinkView{Label: filepath.Base(name), Path: relPath})
	}
	return links
}

func relativePath(outputDir string, target string) (string, error) {
	rel, err := filepath.Rel(outputDir, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func readCapacityRows(baseDir string) ([]benchmarkRow, error) {
	rows := make([]benchmarkRow, 0)
	files := []string{"results-fire-and-forget.tsv", "results-request-reply.tsv", "results-write.tsv"}
	found := false
	for _, name := range files {
		fileRows, err := readBenchmarkTSV(filepath.Join(baseDir, name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		rows = append(rows, fileRows...)
		found = true
	}
	if found {
		return rows, nil
	}
	fileRows, err := readBenchmarkTSV(filepath.Join(baseDir, "results.tsv"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return fileRows, nil
}

func readBenchmarkTSV(path string) ([]benchmarkRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	index := makeHeaderIndex(headers)

	required := []string{"mode", "clients", "size", "rate", "repeat", "sender_total_messages", "sender_mps", "sender_mean_latency_ms", "sender_p50_latency_ms", "sender_p99_latency_ms", "sender_runtime_errors", "sink_received_messages", "sink_errors", "sink_active_receive_mps", "node_cpu_percent", "total_cpu_percent"}
	for _, key := range required {
		if _, ok := index[key]; !ok {
			return nil, fmt.Errorf("missing %q header in %s", key, path)
		}
	}

	rows := []benchmarkRow{}
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		row := benchmarkRow{
			Mode:                 field(record, index, "mode"),
			Clients:              mustParseInt(field(record, index, "clients")),
			Size:                 mustParseInt(field(record, index, "size")),
			Rate:                 mustParseInt(field(record, index, "rate")),
			Repeat:               mustParseInt(field(record, index, "repeat")),
			SenderTotalMessages:  mustParseInt64(field(record, index, "sender_total_messages")),
			SenderMPS:            mustParseFloat(field(record, index, "sender_mps")),
			SenderMeanLatencyMS:  mustParseFloat(field(record, index, "sender_mean_latency_ms")),
			SenderP50LatencyMS:   mustParseFloat(field(record, index, "sender_p50_latency_ms")),
			SenderP99LatencyMS:   mustParseFloat(field(record, index, "sender_p99_latency_ms")),
			SenderRuntimeErrors:  mustParseInt64(field(record, index, "sender_runtime_errors")),
			SinkReceivedMessages: mustParseInt64(field(record, index, "sink_received_messages")),
			SinkErrors:           mustParseInt64(field(record, index, "sink_errors")),
			SinkActiveReceiveMPS: mustParseFloat(field(record, index, "sink_active_receive_mps")),
			NodeCPUPercent:       mustParseFloat(field(record, index, "node_cpu_percent")),
			TotalCPUPercent:      mustParseFloat(field(record, index, "total_cpu_percent")),
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func readSlimRepoCSV(path string) ([]slimRepoCSVRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	index := makeHeaderIndex(headers)
	required := []string{"senders", "messages_per_sender", "payload_bytes", "total_messages", "run", "send_elapsed_s", "total_elapsed_s", "total_received", "send_mps", "recv_mps"}
	for _, key := range required {
		if _, ok := index[key]; !ok {
			return nil, fmt.Errorf("missing %q header in %s", key, path)
		}
	}

	rows := []slimRepoCSVRow{}
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		rows = append(rows, slimRepoCSVRow{
			Senders:           mustParseInt(field(record, index, "senders")),
			MessagesPerSender: mustParseInt(field(record, index, "messages_per_sender")),
			PayloadBytes:      mustParseInt(field(record, index, "payload_bytes")),
			TotalMessages:     mustParseInt64(field(record, index, "total_messages")),
			Run:               mustParseInt(field(record, index, "run")),
			SendElapsedS:      mustParseFloat(field(record, index, "send_elapsed_s")),
			TotalElapsedS:     mustParseFloat(field(record, index, "total_elapsed_s")),
			TotalReceived:     mustParseInt64(field(record, index, "total_received")),
			SendMPS:           mustParseFloat(field(record, index, "send_mps")),
			RecvMPS:           mustParseFloat(field(record, index, "recv_mps")),
		})
	}
	return rows, nil
}

func makeHeaderIndex(headers []string) map[string]int {
	index := make(map[string]int, len(headers))
	for i, header := range headers {
		index[strings.TrimSpace(header)] = i
	}
	return index
}

func field(record []string, index map[string]int, key string) string {
	position, ok := index[key]
	if !ok || position >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[position])
}

func mustParseInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func mustParseInt64(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func mustParseFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

type caseKey struct {
	Mode    string
	Clients int
	Size    int
	Rate    int
}

type caseSummary struct {
	Key           caseKey
	Rows          []benchmarkRow
	Sender        sampleStats
	Observed      sampleStats
	MeanLatency   sampleStats
	P50Latency    sampleStats
	P99Latency    sampleStats
	NodeCPU       sampleStats
	TotalCPU      sampleStats
	TotalErrors   int64
	ObservedLabel string
}

func buildModeTables(rows []benchmarkRow) []tableView {
	caseSummaries := summarizeCases(rows)
	byMode := map[string][]caseSummary{}
	for _, summary := range caseSummaries {
		byMode[summary.Key.Mode] = append(byMode[summary.Key.Mode], summary)
	}

	tables := []tableView{}
	for _, mode := range benchmarkModeOrder {
		modeRows := byMode[mode]
		if len(modeRows) == 0 {
			continue
		}
		if mode == "request-reply" {
			tables = append(tables, buildRequestReplyTable(modeRows))
			continue
		}
		tables = append(tables, buildThroughputTable(modeRows))
	}
	return tables
}

func summarizeCases(rows []benchmarkRow) []caseSummary {
	grouped := map[caseKey][]benchmarkRow{}
	for _, row := range rows {
		key := caseKey{Mode: row.Mode, Clients: row.Clients, Size: row.Size, Rate: row.Rate}
		grouped[key] = append(grouped[key], row)
	}

	keys := make([]caseKey, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i int, j int) bool {
		if modeSortIndex(keys[i].Mode) != modeSortIndex(keys[j].Mode) {
			return modeSortIndex(keys[i].Mode) < modeSortIndex(keys[j].Mode)
		}
		if keys[i].Clients != keys[j].Clients {
			return keys[i].Clients < keys[j].Clients
		}
		if keys[i].Size != keys[j].Size {
			return keys[i].Size < keys[j].Size
		}
		return keys[i].Rate < keys[j].Rate
	})

	summaries := make([]caseSummary, 0, len(keys))
	for _, key := range keys {
		rows := grouped[key]
		totalErrors := int64(0)
		for _, row := range rows {
			totalErrors += row.SenderRuntimeErrors + row.SinkErrors
		}
		summaries = append(summaries, caseSummary{
			Key:           key,
			Rows:          rows,
			Sender:        computeSampleStats(senderValues(rows)),
			Observed:      computeSampleStats(observedValues(rows)),
			MeanLatency:   computeSampleStats(meanLatencyValues(rows)),
			P50Latency:    computeSampleStats(p50LatencyValues(rows)),
			P99Latency:    computeSampleStats(p99LatencyValues(rows)),
			NodeCPU:       computeSampleStats(nodeCPUValues(rows)),
			TotalCPU:      computeSampleStats(totalCPUValues(rows)),
			TotalErrors:   totalErrors,
			ObservedLabel: modeObservedThroughputLabel(key.Mode),
		})
	}
	return summaries
}

func buildThroughputTable(rows []caseSummary) tableView {
	columns := []string{"Clients", "Payload", "Rate", "Repeats", "Sender Mean msg/sec", "Sender 95% CI", rows[0].ObservedLabel, "Observed 95% CI", "Node CPU %", "Total CPU %", "Errors"}
	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, []string{
			strconv.Itoa(row.Key.Clients),
			fmt.Sprintf("%dB", row.Key.Size),
			strconv.Itoa(row.Key.Rate),
			strconv.Itoa(len(row.Rows)),
			formatFloat(row.Sender.Mean),
			formatCI(row.Sender),
			formatFloat(row.Observed.Mean),
			formatCI(row.Observed),
			formatFloat(row.NodeCPU.Mean),
			formatFloat(row.TotalCPU.Mean),
			strconv.FormatInt(row.TotalErrors, 10),
		})
	}
	return tableView{Title: fmt.Sprintf("%s Results", modeDisplayTitle(rows[0].Key.Mode)), Columns: columns, Rows: tableRows}
}

func buildRequestReplyTable(rows []caseSummary) tableView {
	columns := []string{"Clients", "Payload", "Rate", "Repeats", "Mean Latency ms", "Mean 95% CI", "P50 Latency ms", "P50 95% CI", "P99 Latency ms", "P99 95% CI", "Node CPU %", "Errors"}
	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, []string{
			strconv.Itoa(row.Key.Clients),
			fmt.Sprintf("%dB", row.Key.Size),
			strconv.Itoa(row.Key.Rate),
			strconv.Itoa(len(row.Rows)),
			formatFloat(row.MeanLatency.Mean),
			formatCI(row.MeanLatency),
			formatFloat(row.P50Latency.Mean),
			formatCI(row.P50Latency),
			formatFloat(row.P99Latency.Mean),
			formatCI(row.P99Latency),
			formatFloat(row.NodeCPU.Mean),
			strconv.FormatInt(row.TotalErrors, 10),
		})
	}
	return tableView{Title: fmt.Sprintf("%s Results", modeDisplayTitle(rows[0].Key.Mode)), Columns: columns, Rows: tableRows}
}

func buildSmokeCards(rows []benchmarkRow) []summaryCardView {
	if len(rows) == 0 {
		return []summaryCardView{{Label: "Smoke data", Value: "Unavailable", Detail: "No smoke TSV artifact was found in this report bundle."}}
	}
	caseSummaries := summarizeCases(rows)
	bestObserved := bestObservedCase(caseSummaries)
	lowestLatency := lowestLatencyCase(caseSummaries)
	totalErrors := int64(0)
	for _, row := range rows {
		totalErrors += row.SenderRuntimeErrors + row.SinkErrors
	}
	cards := []summaryCardView{{Label: "Repeated measurements", Value: strconv.Itoa(len(rows)), Detail: fmt.Sprintf("%d unique benchmark cases across %d modes.", len(caseSummaries), countModes(rows))}, {Label: "Best observed throughput", Value: formatFloat(bestObserved.Observed.Mean) + " msg/sec", Detail: describeCase(bestObserved.Key)}, {Label: "Total runtime errors", Value: strconv.FormatInt(totalErrors, 10), Detail: "Combined sender and sink runtime error count."}}
	if lowestLatency.Key.Mode != "" {
		cards = append(cards, summaryCardView{Label: "Lowest P50 latency", Value: formatFloat(lowestLatency.P50Latency.Mean) + " ms", Detail: describeCase(lowestLatency.Key)})
	}
	return cards
}

func buildCapacityCards(rows []benchmarkRow) []summaryCardView {
	if len(rows) == 0 {
		return []summaryCardView{{Label: "Capacity data", Value: "Rendered from markdown", Detail: "Structured TSV samples were not found, so the dashboard falls back to the published capacity markdown."}}
	}
	caseSummaries := summarizeCases(rows)
	bestObserved := bestObservedCase(caseSummaries)
	totalErrors := int64(0)
	for _, row := range rows {
		totalErrors += row.SenderRuntimeErrors + row.SinkErrors
	}
	cards := []summaryCardView{{Label: "Capacity mode samples", Value: strconv.Itoa(len(rows)), Detail: fmt.Sprintf("%d unique sampled cases across %d modes.", len(caseSummaries), countModes(rows))}, {Label: "Strongest measured throughput", Value: formatFloat(bestObserved.Observed.Mean) + " msg/sec", Detail: describeCase(bestObserved.Key)}, {Label: "Total runtime errors", Value: strconv.FormatInt(totalErrors, 10), Detail: "Combined sender and sink runtime error count from the sampled capacity runs."}}
	if latency := lowestLatencyCase(caseSummaries); latency.Key.Mode != "" {
		cards = append(cards, summaryCardView{Label: "Request-reply P50 latency", Value: formatFloat(latency.P50Latency.Mean) + " ms", Detail: describeCase(latency.Key)})
	}
	return cards
}

func buildSlimRepoCards(rows []slimRepoCSVRow) []summaryCardView {
	if len(rows) == 0 {
		return nil
	}
	grouped := summarizeSlimRepoRows(rows)
	bestRecv := grouped[0]
	bestSend := grouped[0]
	bestDelivery := grouped[0]
	for _, row := range grouped[1:] {
		if row.RecvMPS.Mean > bestRecv.RecvMPS.Mean {
			bestRecv = row
		}
		if row.SendMPS.Mean > bestSend.SendMPS.Mean {
			bestSend = row
		}
		if row.DeliveryPct.Mean > bestDelivery.DeliveryPct.Mean {
			bestDelivery = row
		}
	}
	return []summaryCardView{{Label: "Configurations", Value: strconv.Itoa(len(grouped)), Detail: fmt.Sprintf("%d raw CSV samples imported from agntcy/slim.", len(rows))}, {Label: "Best receive throughput", Value: formatFloat(bestRecv.RecvMPS.Mean) + " msg/sec", Detail: describeSlimRepoCase(bestRecv.Key)}, {Label: "Best send throughput", Value: formatFloat(bestSend.SendMPS.Mean) + " msg/sec", Detail: describeSlimRepoCase(bestSend.Key)}, {Label: "Highest delivery", Value: formatFloat(bestDelivery.DeliveryPct.Mean) + "%", Detail: describeSlimRepoCase(bestDelivery.Key)}}
}

func bestObservedCase(rows []caseSummary) caseSummary {
	best := caseSummary{}
	for _, row := range rows {
		if row.Observed.Mean > best.Observed.Mean {
			best = row
		}
	}
	return best
}

func lowestLatencyCase(rows []caseSummary) caseSummary {
	best := caseSummary{}
	initialized := false
	for _, row := range rows {
		if row.Key.Mode != "request-reply" || row.P50Latency.Count == 0 {
			continue
		}
		if !initialized || row.P50Latency.Mean < best.P50Latency.Mean {
			best = row
			initialized = true
		}
	}
	return best
}

func countModes(rows []benchmarkRow) int {
	seen := map[string]struct{}{}
	for _, row := range rows {
		seen[row.Mode] = struct{}{}
	}
	return len(seen)
}

func describeCase(key caseKey) string {
	if key.Mode == "" {
		return ""
	}
	return fmt.Sprintf("%s, %d client(s), %dB payload, %d msg/sec.", modeDisplayTitle(key.Mode), key.Clients, key.Size, key.Rate)
}

func modeDisplayTitle(mode string) string {
	if spec, ok := benchmarkModeSpecs[mode]; ok {
		return spec.Title
	}
	return strings.Title(mode)
}

func modeObservedThroughputLabel(mode string) string {
	if spec, ok := benchmarkModeSpecs[mode]; ok {
		return spec.ObservedThroughputLabel
	}
	return "Observed Throughput"
}

func modeSortIndex(mode string) int {
	for i, candidate := range benchmarkModeOrder {
		if candidate == mode {
			return i
		}
	}
	return len(benchmarkModeOrder)
}

func senderValues(rows []benchmarkRow) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.SenderMPS)
	}
	return values
}

func observedValues(rows []benchmarkRow) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		if row.Mode == "write" {
			values = append(values, row.SenderMPS)
			continue
		}
		values = append(values, row.SinkActiveReceiveMPS)
	}
	return values
}

func meanLatencyValues(rows []benchmarkRow) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		if row.SenderMeanLatencyMS > 0 {
			values = append(values, row.SenderMeanLatencyMS)
		}
	}
	return values
}

func p50LatencyValues(rows []benchmarkRow) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		if row.SenderP50LatencyMS > 0 {
			values = append(values, row.SenderP50LatencyMS)
		}
	}
	return values
}

func p99LatencyValues(rows []benchmarkRow) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		if row.SenderP99LatencyMS > 0 {
			values = append(values, row.SenderP99LatencyMS)
		}
	}
	return values
}

func nodeCPUValues(rows []benchmarkRow) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.NodeCPUPercent)
	}
	return values
}

func totalCPUValues(rows []benchmarkRow) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.TotalCPUPercent)
	}
	return values
}

func computeSampleStats(values []float64) sampleStats {
	count := len(values)
	if count == 0 {
		return sampleStats{}
	}
	mean := stat.Mean(values, nil)
	if count == 1 {
		return sampleStats{Count: 1, Mean: mean, Variance: 0, CILow: mean, CIHigh: mean}
	}
	variance := stat.Variance(values, nil)
	stddev := stat.StdDev(values, nil)
	standardError := stddev / math.Sqrt(float64(count))
	tDist := distuv.StudentsT{Mu: mean, Sigma: standardError, Nu: float64(count - 1)}
	tailProbability := confidenceIntervalAlpha / 2
	ciLow := math.Max(0, tDist.Quantile(tailProbability))
	ciHigh := tDist.Quantile(1 - tailProbability)
	return sampleStats{Count: count, Mean: mean, Variance: variance, CILow: ciLow, CIHigh: ciHigh}
}

func confidenceIntervalAvailable(count int) bool {
	return count >= configuredMinimumConfidenceIntervalRuns
}

func unavailableConfidenceIntervalLabel() string {
	return fmt.Sprintf("n/a (need >=%d runs)", configuredMinimumConfidenceIntervalRuns)
}

func formatCI(stats sampleStats) string {
	if stats.Count == 0 {
		return "-"
	}
	if !confidenceIntervalAvailable(stats.Count) {
		return unavailableConfidenceIntervalLabel()
	}
	return fmt.Sprintf("[%s, %s]", formatFloat(stats.CILow), formatFloat(stats.CIHigh))
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

type slimRepoCaseKey struct {
	Senders           int
	PayloadBytes      int
	MessagesPerSender int
}

type slimRepoCaseSummary struct {
	Key          slimRepoCaseKey
	Rows         []slimRepoCSVRow
	SendMPS      sampleStats
	RecvMPS      sampleStats
	DeliveryPct  sampleStats
	SendElapsed  sampleStats
	TotalElapsed sampleStats
}

func summarizeSlimRepoRows(rows []slimRepoCSVRow) []slimRepoCaseSummary {
	grouped := map[slimRepoCaseKey][]slimRepoCSVRow{}
	for _, row := range rows {
		key := slimRepoCaseKey{Senders: row.Senders, PayloadBytes: row.PayloadBytes, MessagesPerSender: row.MessagesPerSender}
		grouped[key] = append(grouped[key], row)
	}

	keys := make([]slimRepoCaseKey, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i int, j int) bool {
		if keys[i].Senders != keys[j].Senders {
			return keys[i].Senders < keys[j].Senders
		}
		return keys[i].PayloadBytes < keys[j].PayloadBytes
	})

	summaries := make([]slimRepoCaseSummary, 0, len(keys))
	for _, key := range keys {
		group := grouped[key]
		delivery := make([]float64, 0, len(group))
		sendElapsed := make([]float64, 0, len(group))
		totalElapsed := make([]float64, 0, len(group))
		sendMPS := make([]float64, 0, len(group))
		recvMPS := make([]float64, 0, len(group))
		for _, row := range group {
			sendElapsed = append(sendElapsed, row.SendElapsedS)
			totalElapsed = append(totalElapsed, row.TotalElapsedS)
			sendMPS = append(sendMPS, row.SendMPS)
			recvMPS = append(recvMPS, row.RecvMPS)
			if row.TotalMessages > 0 {
				delivery = append(delivery, 100*float64(row.TotalReceived)/float64(row.TotalMessages))
			}
		}
		summaries = append(summaries, slimRepoCaseSummary{Key: key, Rows: group, SendMPS: computeSampleStats(sendMPS), RecvMPS: computeSampleStats(recvMPS), DeliveryPct: computeSampleStats(delivery), SendElapsed: computeSampleStats(sendElapsed), TotalElapsed: computeSampleStats(totalElapsed)})
	}
	return summaries
}

func buildSlimRepoTable(rows []slimRepoCSVRow) tableView {
	summaries := summarizeSlimRepoRows(rows)
	tableRows := make([][]string, 0, len(summaries))
	for _, summary := range summaries {
		tableRows = append(tableRows, []string{
			strconv.Itoa(summary.Key.Senders),
			fmt.Sprintf("%dB", summary.Key.PayloadBytes),
			strconv.Itoa(summary.Key.MessagesPerSender),
			strconv.Itoa(len(summary.Rows)),
			formatFloat(summary.SendMPS.Mean),
			formatFloat(summary.RecvMPS.Mean),
			formatFloat(summary.DeliveryPct.Mean) + "%",
			formatFloat(summary.SendElapsed.Mean) + "s",
			formatFloat(summary.TotalElapsed.Mean) + "s",
		})
	}
	return tableView{Title: "Data-Plane Benchmark CSV Summary", Columns: []string{"Senders", "Payload", "Messages/Sender", "Runs", "Avg Send MPS", "Avg Receive MPS", "Avg Delivery", "Avg Send Elapsed", "Avg Total Elapsed"}, Rows: tableRows}
}

func describeSlimRepoCase(key slimRepoCaseKey) string {
	return fmt.Sprintf("%d senders, %dB payload, %d messages per sender.", key.Senders, key.PayloadBytes, key.MessagesPerSender)
}

func renderDashboard(view dashboardView) ([]byte, error) {
	tmpl, err := template.New("benchmark-dashboard").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>SLIM Benchmark Dashboard</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f3eee4;
      --panel: rgba(255, 252, 246, 0.94);
      --panel-strong: rgba(255, 255, 255, 0.92);
      --text: #1d2731;
      --muted: #5b6773;
      --accent: #0f766e;
      --accent-strong: #134e4a;
      --warm: #b45309;
      --border: rgba(15, 118, 110, 0.18);
      --shadow: 0 28px 70px rgba(29, 39, 49, 0.12);
      --code: #f3f6f8;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", Georgia, serif;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(15, 118, 110, 0.15), transparent 28%),
        radial-gradient(circle at bottom right, rgba(180, 83, 9, 0.14), transparent 22%),
        linear-gradient(180deg, #faf7f2 0%, var(--bg) 100%);
      padding: 40px 18px 56px;
    }
    main {
      max-width: 1180px;
      margin: 0 auto;
      display: grid;
      gap: 24px;
    }
    .hero, .section {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 28px;
      box-shadow: var(--shadow);
      padding: 32px;
    }
    .hero h1 {
      margin: 0 0 12px;
      font-size: clamp(2.7rem, 4vw, 4.5rem);
      line-height: 0.92;
      letter-spacing: -0.05em;
      max-width: 12ch;
    }
    .eyebrow {
      display: inline-block;
      text-transform: uppercase;
      letter-spacing: 0.16em;
      font-size: 0.78rem;
      color: var(--accent);
      margin-bottom: 14px;
    }
    .lead {
      font-size: 1.08rem;
      line-height: 1.75;
      color: var(--muted);
      max-width: 70ch;
      margin: 0;
    }
    .nav {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 24px;
    }
    .nav a, .artifact-list a {
      color: var(--accent-strong);
      text-decoration: none;
    }
    .nav a {
      padding: 10px 14px;
      border-radius: 999px;
      background: rgba(15, 118, 110, 0.08);
      border: 1px solid rgba(15, 118, 110, 0.14);
      font-size: 0.95rem;
    }
    .meta {
      margin-top: 18px;
      color: var(--muted);
      font-size: 0.94rem;
    }
    .section-head {
      display: flex;
      justify-content: space-between;
      gap: 16px;
      align-items: flex-start;
      margin-bottom: 18px;
    }
    .section h2 {
      margin: 0 0 8px;
      font-size: clamp(1.7rem, 2.2vw, 2.4rem);
      letter-spacing: -0.04em;
    }
    .section p {
      margin: 0;
      color: var(--muted);
      line-height: 1.7;
    }
    .card-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 14px;
      margin: 20px 0 24px;
    }
    .card {
      background: var(--panel-strong);
      border: 1px solid rgba(15, 118, 110, 0.12);
      border-radius: 22px;
      padding: 18px;
    }
    .card .label {
      color: var(--muted);
      font-size: 0.92rem;
      display: block;
      margin-bottom: 8px;
    }
    .card .value {
      display: block;
      font-size: 1.55rem;
      font-weight: 700;
      letter-spacing: -0.03em;
      margin-bottom: 8px;
    }
    .card .detail {
      color: var(--muted);
      font-size: 0.92rem;
      line-height: 1.55;
    }
    .table-stack {
      display: grid;
      gap: 18px;
      margin-bottom: 22px;
    }
    .table-panel {
      background: rgba(255, 255, 255, 0.74);
      border: 1px solid rgba(15, 118, 110, 0.12);
      border-radius: 22px;
      overflow: hidden;
    }
    .table-panel h3 {
      margin: 0;
      padding: 18px 20px 0;
      font-size: 1.18rem;
    }
    .table-wrap {
      overflow-x: auto;
      padding: 14px 18px 18px;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.95rem;
    }
    th, td {
      text-align: left;
      padding: 12px 10px;
      border-bottom: 1px solid rgba(15, 118, 110, 0.1);
      vertical-align: top;
    }
    th {
      color: var(--accent-strong);
      font-size: 0.84rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }
    details {
      background: rgba(255, 255, 255, 0.76);
      border: 1px solid rgba(15, 118, 110, 0.12);
      border-radius: 22px;
      padding: 0 18px 18px;
      margin-bottom: 14px;
    }
    summary {
      cursor: pointer;
      list-style: none;
      padding: 18px 0;
      font-weight: 700;
      color: var(--accent-strong);
    }
    summary::-webkit-details-marker { display: none; }
    .doc-link {
      display: inline-block;
      margin-left: 10px;
      color: var(--warm);
      font-size: 0.9rem;
      text-decoration: none;
    }
    .markdown {
      color: var(--text);
      line-height: 1.7;
    }
    .markdown h1, .markdown h2, .markdown h3, .markdown h4 {
      letter-spacing: -0.03em;
      margin-top: 1.6em;
      margin-bottom: 0.55em;
    }
    .markdown h1:first-child, .markdown h2:first-child, .markdown h3:first-child, .markdown h4:first-child {
      margin-top: 0;
    }
    .markdown p, .markdown li { color: var(--text); }
    .markdown ul, .markdown ol { padding-left: 1.35rem; }
    .markdown code {
      background: var(--code);
      padding: 0.14rem 0.34rem;
      border-radius: 6px;
      font-size: 0.92em;
    }
    .markdown pre {
      overflow-x: auto;
      background: #15212b;
      color: #f2f5f7;
      padding: 16px;
      border-radius: 16px;
    }
    .markdown table {
      display: block;
      overflow-x: auto;
      border-collapse: collapse;
      margin: 1rem 0;
    }
    .markdown thead th {
      position: sticky;
      top: 0;
      background: rgba(15, 118, 110, 0.06);
    }
    .artifact-list {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 14px;
    }
    .artifact-list a {
      padding: 8px 12px;
      background: rgba(15, 118, 110, 0.08);
      border-radius: 999px;
      border: 1px solid rgba(15, 118, 110, 0.14);
      font-size: 0.92rem;
    }
    .empty-state {
      padding: 22px;
      border-radius: 22px;
      background: rgba(255,255,255,0.74);
      border: 1px dashed rgba(15,118,110,0.24);
      color: var(--muted);
      line-height: 1.7;
    }
    @media (max-width: 720px) {
      body { padding: 18px 12px 30px; }
      .hero, .section { padding: 22px 18px; border-radius: 24px; }
      .section-head { display: block; }
    }
  </style>
</head>
<body>
  <main>
    <section class="hero">
      <div class="eyebrow">Static Benchmark Dashboard</div>
      <h1>SLIM benchmark reports.</h1>
	<p class="lead">This dashboard turns the benchmark artifacts into a browsable report surface. It keeps the raw markdown, logs, and TSV or CSV files one click away, while surfacing the benchmark tables and summary statistics in HTML. Confidence intervals are shown only when a case has at least 20 repeated runs.</p>
      <div class="nav">
        {{range .Sections}}<a href="#{{.ID}}">{{.Title}}</a>{{end}}
      </div>
      <div class="meta">Generated: {{.GeneratedAt}}</div>
    </section>
    {{if .HasSections}}
      {{range .Sections}}
      <section class="section" id="{{.ID}}">
        <div class="section-head">
          <div>
            <h2>{{.Title}}</h2>
            <p>{{.Description}}</p>
          </div>
        </div>
        {{if .Cards}}
        <div class="card-grid">
          {{range .Cards}}
          <article class="card">
            <span class="label">{{.Label}}</span>
            <span class="value">{{.Value}}</span>
            <span class="detail">{{.Detail}}</span>
          </article>
          {{end}}
        </div>
        {{end}}
        {{if .Tables}}
        <div class="table-stack">
          {{range .Tables}}
          <section class="table-panel">
            <h3>{{.Title}}</h3>
            <div class="table-wrap">
              <table>
                <thead>
                  <tr>{{range .Columns}}<th>{{.}}</th>{{end}}</tr>
                </thead>
                <tbody>
                  {{range .Rows}}
                  <tr>{{range .}}<td>{{.}}</td>{{end}}</tr>
                  {{end}}
                </tbody>
              </table>
            </div>
          </section>
          {{end}}
        </div>
        {{end}}
        {{range .Documents}}
        <details {{if .Open}}open{{end}}>
          <summary>{{.Title}} <a class="doc-link" href="{{.Path}}">raw file</a></summary>
          <div class="markdown">{{.HTML}}</div>
        </details>
        {{end}}
        {{if .Artifacts}}
        <div class="artifact-list">
          {{range .Artifacts}}<a href="{{.Path}}">{{.Label}}</a>{{end}}
        </div>
        {{end}}
      </section>
      {{end}}
    {{else}}
      <section class="section">
        <div class="empty-state">No benchmark artifacts were found. Generate the SLIM benchmark reports first, then rerun the dashboard renderer.</div>
      </section>
    {{end}}
  </main>
</body>
</html>`)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, view); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
