package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const benchmarkTSVHeader = "mode\tclients\tsize\trate\trepeat\tsender_total_messages\tsender_mps\tsender_mean_latency_ms\tsender_p50_latency_ms\tsender_p90_latency_ms\tsender_p99_latency_ms\tsender_max_latency_ms\tsender_runtime_errors\tsender_duration\tsink_received_messages\tsink_errors\tsink_receive_mps\tsink_active_receive_mps\tsink_elapsed_seconds\tsink_active_receive_seconds\tsender_cpu_seconds\tsender_cpu_percent\tresponder_cpu_seconds\tresponder_cpu_percent\tnode_cpu_seconds\tnode_cpu_percent\ttotal_cpu_seconds\ttotal_cpu_percent"

func TestBuildDashboardWithSmokeCapacityAndSlimRepo(t *testing.T) {
	tempDir := t.TempDir()
	smokeDir := filepath.Join(tempDir, "smoke")
	capacityDir := filepath.Join(tempDir, "capacity")
	externalDir := filepath.Join(tempDir, "external")
	if err := os.MkdirAll(smokeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(capacityDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(externalDir, 0o755); err != nil {
		t.Fatal(err)
	}

	smokeRows := []string{benchmarkTSVHeader}
	smokeRows = append(smokeRows, buildRepeatedBenchmarkRows("request-reply", 1, 16, 100, defaultMinimumConfidenceIntervalRuns, "1s", 98, 99, 2.10, 1.90, 3.30, 22, 46, true)...)
	smokeRows = append(smokeRows, buildRepeatedBenchmarkRows("fire-and-forget", 1, 16, 1000, defaultMinimumConfidenceIntervalRuns, "1s", 940, 925, 0, 0, 0, 28, 46, true)...)
	smokeRows = append(smokeRows, buildRepeatedBenchmarkRows("write", 1, 16, 1000, defaultMinimumConfidenceIntervalRuns, "1s", 980, 0, 0, 0, 0, 18, 27, false)...)
	writeFile(t, filepath.Join(smokeDir, "results.tsv"), strings.Join(smokeRows, "\n"))
	writeFile(t, filepath.Join(smokeDir, "suite_summary.md"), "# Smoke Summary\n\n| Column | Value |\n| --- | --- |\n| Modes | 3 |\n")
	writeFile(t, filepath.Join(smokeDir, "technical_report.md"), "# Smoke Technical\n\nRendered markdown body.")

	capacityRows := []string{benchmarkTSVHeader}
	capacityRows = append(capacityRows, buildRepeatedBenchmarkRows("fire-and-forget", 1, 16384, 128000, defaultMinimumConfidenceIntervalRuns, "5s", 118000, 109500, 0, 0, 0, 42, 74, true)...)
	writeFile(t, filepath.Join(capacityDir, "results-fire-and-forget.tsv"), strings.Join(capacityRows, "\n"))
	writeFile(t, filepath.Join(capacityDir, "capacity_sweep.md"), "# Capacity Sweep\n\n## Sink-Backed Modes\n\n| Mode | Best Offered Rate |\n| --- | --- |\n| fire-and-forget | 128000 |\n")

	writeFile(t, filepath.Join(externalDir, "benchmark-results.csv"), strings.Join([]string{
		"senders,messages_per_sender,payload_bytes,total_messages,run,send_elapsed_s,total_elapsed_s,total_received,send_mps,recv_mps",
		"1,500000,8,500000,1,0.80,0.95,500000,625000,526315",
		"4,500000,64,2000000,1,1.90,2.20,1980000,1052631,900000",
	}, "\n"))

	outputPath := filepath.Join(tempDir, "site", "index.html")
	view, err := buildDashboard(smokeDir, capacityDir, filepath.Join(externalDir, "benchmark-results.csv"), "", outputPath)
	if err != nil {
		t.Fatalf("buildDashboard returned error: %v", err)
	}
	html, err := renderDashboard(view)
	if err != nil {
		t.Fatalf("renderDashboard returned error: %v", err)
	}
	content := string(html)
	checks := []string{
		"CSIT SLIM Smoke Suite",
		"CSIT SLIM Capacity Sweeps",
		"SLIM Repo Data-Plane Benchmark",
		"Data-Plane Benchmark CSV Summary",
		"Confidence intervals are shown only when a case has at least 20 repeated runs.",
		"Smoke Summary",
		"Capacity Sweep",
		"benchmark-results.csv",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Fatalf("expected HTML to contain %q", check)
		}
	}
}

func TestBuildDashboardWithoutInputsProducesEmptyState(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "index.html")
	view, err := buildDashboard(filepath.Join(tempDir, "missing-smoke"), filepath.Join(tempDir, "missing-capacity"), filepath.Join(tempDir, "missing.csv"), "", outputPath)
	if err != nil {
		t.Fatalf("buildDashboard returned error: %v", err)
	}
	html, err := renderDashboard(view)
	if err != nil {
		t.Fatalf("renderDashboard returned error: %v", err)
	}
	if !strings.Contains(string(html), "No benchmark artifacts were found") {
		t.Fatalf("expected empty-state text in generated HTML")
	}
}

func TestFormatCIRequiresMinimumSampleCount(t *testing.T) {
	configuredMinimumConfidenceIntervalRuns = defaultMinimumConfidenceIntervalRuns
	if got := formatCI(sampleStats{Count: defaultMinimumConfidenceIntervalRuns - 1, CILow: 1.23, CIHigh: 4.56}); got != unavailableConfidenceIntervalLabel() {
		t.Fatalf("formatCI below threshold = %q, want %q", got, unavailableConfidenceIntervalLabel())
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func buildRepeatedBenchmarkRows(mode string, clients int, size int, rate int, repeats int, duration string, baseSenderMPS float64, baseObservedMPS float64, baseMeanLatencyMS float64, baseP50LatencyMS float64, baseP99LatencyMS float64, baseNodeCPUPercent float64, baseTotalCPUPercent float64, hasSink bool) []string {
	rows := make([]string, 0, repeats)
	for repeat := 1; repeat <= repeats; repeat++ {
		jitter := float64((repeat % 5) - 2)
		senderMPS := baseSenderMPS + jitter
		observedMPS := 0.0
		sinkReceivedMessages := int64(0)
		sinkReceiveMPS := 0.0
		sinkActiveReceiveMPS := 0.0
		responderCPUSeconds := 0.0
		responderCPUPercent := 0.0
		if hasSink {
			observedMPS = baseObservedMPS + jitter
			sinkReceivedMessages = int64(rate)
			sinkReceiveMPS = observedMPS
			sinkActiveReceiveMPS = observedMPS
			responderCPUSeconds = 0.2
			responderCPUPercent = 8 + jitter/2
		}

		meanLatency := 0.0
		p50Latency := 0.0
		p99Latency := 0.0
		maxLatency := 0.0
		if baseMeanLatencyMS > 0 {
			meanLatency = baseMeanLatencyMS + 0.03*jitter
			p50Latency = baseP50LatencyMS + 0.02*jitter
			p99Latency = baseP99LatencyMS + 0.04*jitter
			maxLatency = p99Latency + 0.8
		}

		nodeCPUPercent := baseNodeCPUPercent + jitter/2
		totalCPUPercent := baseTotalCPUPercent + jitter/2
		rows = append(rows, fmt.Sprintf(
			"%s\t%d\t%d\t%d\t%d\t%d\t%.2f\t%.2f\t%.2f\t0\t%.2f\t%.2f\t0\t%s\t%d\t0\t%.2f\t%.2f\t1.0\t1.0\t0.3\t%.2f\t%.1f\t%.2f\t0.5\t%.2f\t1.0\t%.2f",
			mode,
			clients,
			size,
			rate,
			repeat,
			rate,
			senderMPS,
			meanLatency,
			p50Latency,
			p99Latency,
			maxLatency,
			duration,
			sinkReceivedMessages,
			sinkReceiveMPS,
			sinkActiveReceiveMPS,
			8+jitter/2,
			responderCPUSeconds,
			responderCPUPercent,
			nodeCPUPercent,
			totalCPUPercent,
		))
	}
	return rows
}
