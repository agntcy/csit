// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = ginkgo.Describe("SLIM Benchmark Suite Matrix", ginkgo.Label("benchmark-suite"), func() {
	ginkgo.It("runs the repeated benchmark matrix and generates reports", func() {
		if envString("SLIM_RUN_BENCHMARK_SUITE", "") == "" {
			ginkgo.Skip("set SLIM_RUN_BENCHMARK_SUITE=1 to run the benchmark suite matrix")
		}

		cfg := loadSuiteConfig()
		ginkgo.By("resetting report artifacts")
		resetSuiteReports(cfg)

		ginkgo.By("starting the local SLIM stack for the suite run")
		startLocalSlimStack()
		defer stopLocalSlimStack()
		stopEchoResponder()

		results := runSuiteMatrix(cfg)
		capacitySweepResults := []capacitySweepCaseResult(nil)
		if cfg.CapacitySweepEnabled {
			capacitySweepResults = runCapacitySweep(cfg)
			writeCapacitySweepReport(cfg, capacitySweepResults)
		}
		writeResultsTSV(cfg.ResultsTSV, results)
		writeSuiteSummary(cfg, results, capacitySweepResults)
		writeTechnicalReport(cfg, results, capacitySweepResults)

		gomega.Expect(cfg.ResultsTSV).To(gomega.BeAnExistingFile())
		gomega.Expect(cfg.SummaryFile).To(gomega.BeAnExistingFile())
		gomega.Expect(cfg.TechnicalReportFile).To(gomega.BeAnExistingFile())
		if cfg.CapacitySweepEnabled {
			gomega.Expect(cfg.CapacitySweepFile).To(gomega.BeAnExistingFile())
		}

		ginkgo.AddReportEntry("Benchmark Suite Summary", cfg.SummaryFile)
		ginkgo.AddReportEntry("Benchmark Technical Report", cfg.TechnicalReportFile)
		if cfg.CapacitySweepEnabled {
			ginkgo.AddReportEntry("Benchmark Capacity Sweep Report", cfg.CapacitySweepFile)
		}
	})
})

func loadSuiteConfig() suiteConfig {
	outputDir, err := filepath.Abs("../reports")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	templateDir, err := filepath.Abs("../templates")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	rawDir := filepath.Join(outputDir, "raw")
	duration := envDuration("DURATION", 5*time.Second)
	modes := normalizeBenchmarkModes(envStringList("MODES", []string{"request-reply", "fire-and-forget", "write"}))
	clients := envIntList("CLIENTS", []int{1, 10, 50})
	sizes := envIntList("SIZES", []int{16, 128, 1024, 10240})
	requestRates := envIntList("REQUEST_RATES", []int{1000})
	pubRatesRaw := strings.TrimSpace(os.Getenv("PUB_RATES"))
	writeRatesRaw := strings.TrimSpace(os.Getenv("WRITE_RATES"))
	pubRates := []int(nil)
	writeRates := []int(nil)
	pubRatesDisplay := "auto (safe default profile)"
	writeRatesDisplay := "auto (falls back to one-way rate profile)"
	pubAutoProfile := true
	if pubRatesRaw != "" {
		pubRates = mustParseIntList(pubRatesRaw)
		pubRatesDisplay = joinInts(pubRates)
		pubAutoProfile = false
	}
	if writeRatesRaw != "" {
		writeRates = mustParseIntList(writeRatesRaw)
		writeRatesDisplay = joinInts(writeRates)
	} else if pubRatesRaw != "" {
		writeRatesDisplay = joinInts(pubRates)
	}

	capacitySweepEnabled := envBool("CAPACITY_SWEEP", false)
	capacitySweepModes := normalizeBenchmarkModes(envStringList("CAPACITY_SWEEP_MODES", []string{"fire-and-forget"}))
	capacitySweepClients := envIntList("CAPACITY_SWEEP_CLIENTS", clients)
	capacitySweepSizes := envIntList("CAPACITY_SWEEP_SIZES", sizes)
	capacitySweepStartRate := envInt("CAPACITY_SWEEP_START_RATE", 1000)
	capacitySweepMaxRate := envInt("CAPACITY_SWEEP_MAX_RATE", 0)
	capacitySweepGrowthFactor := envFloat("CAPACITY_SWEEP_GROWTH_FACTOR", 2.0)
	capacitySweepPlateauThreshold := envFloat("CAPACITY_SWEEP_PLATEAU_THRESHOLD", 0.05)
	capacitySweepPlateauSteps := envInt("CAPACITY_SWEEP_PLATEAU_STEPS", 2)
	capacitySweepMaxSteps := envInt("CAPACITY_SWEEP_MAX_STEPS", 8)
	capacitySweepRepeats := envInt("CAPACITY_SWEEP_REPEATS", 1)
	capacitySweepRefinementSteps := envInt("CAPACITY_SWEEP_REFINEMENT_STEPS", 4)
	capacitySweepMinRateDelta := envInt("CAPACITY_SWEEP_MIN_RATE_DELTA", 250)

	return suiteConfig{
		OutputDir:                     outputDir,
		RawDir:                        rawDir,
		TemplateDir:                   templateDir,
		SummaryFile:                   filepath.Join(outputDir, "suite_summary.md"),
		TechnicalReportFile:           filepath.Join(outputDir, "technical_report.md"),
		ResultsTSV:                    filepath.Join(outputDir, "results.tsv"),
		CapacitySweepFile:             filepath.Join(outputDir, "capacity_sweep.md"),
		Sizes:                         sizes,
		Clients:                       clients,
		Modes:                         modes,
		RequestRates:                  requestRates,
		PubRates:                      pubRates,
		WriteRates:                    writeRates,
		PubRatesDisplay:               pubRatesDisplay,
		WriteRatesDisplay:             writeRatesDisplay,
		PubRateAutoProfile:            pubAutoProfile,
		Duration:                      duration,
		DurationDisplay:               duration.String(),
		Repeats:                       envInt("REPEATS", 1),
		Destination:                   "agntcy/demo/echo",
		ModesDisplay:                  strings.Join(modes, " "),
		ClientsDisplay:                joinInts(clients),
		SizesDisplay:                  joinInts(sizes),
		RequestRatesDisplay:           joinInts(requestRates),
		CapacitySweepEnabled:          capacitySweepEnabled,
		CapacitySweepModes:            capacitySweepModes,
		CapacitySweepClients:          capacitySweepClients,
		CapacitySweepSizes:            capacitySweepSizes,
		CapacitySweepStartRate:        capacitySweepStartRate,
		CapacitySweepMaxRate:          capacitySweepMaxRate,
		CapacitySweepGrowthFactor:     capacitySweepGrowthFactor,
		CapacitySweepPlateauThreshold: capacitySweepPlateauThreshold,
		CapacitySweepPlateauSteps:     capacitySweepPlateauSteps,
		CapacitySweepMaxSteps:         capacitySweepMaxSteps,
		CapacitySweepRepeats:          capacitySweepRepeats,
		CapacitySweepRefinementSteps:  capacitySweepRefinementSteps,
		CapacitySweepMinRateDelta:     capacitySweepMinRateDelta,
		CapacitySweepModesDisplay:     strings.Join(capacitySweepModes, " "),
		CapacitySweepClientsDisplay:   joinInts(capacitySweepClients),
		CapacitySweepSizesDisplay:     joinInts(capacitySweepSizes),
	}
}

func runSuiteMatrix(cfg suiteConfig) []benchmarkRunResult {
	results := make([]benchmarkRunResult, 0)
	for _, mode := range cfg.Modes {
		modeStart := len(results)
		for _, clients := range cfg.Clients {
			for _, size := range cfg.Sizes {
				rateValues := modeRateValues(cfg, mode, clients, size)

				for _, rate := range rateValues {
					ginkgo.By(fmt.Sprintf("running %s clients=%d size=%d rate=%d repeats=%d", mode, clients, size, rate, cfg.Repeats))
					responderMode := modeResponderKind(mode)

					for repeat := 1; repeat <= cfg.Repeats; repeat++ {
						reportFile := filepath.Join(cfg.RawDir, fmt.Sprintf("report_%s_c%d_s%d_r%d_rep%02d.md", mode, clients, size, rate, repeat))
						statsFile := filepath.Join(buildDir, fmt.Sprintf("stats_%s_c%d_s%d_r%d_rep%02d.txt", mode, clients, size, rate, repeat))

						stopEchoResponder()
						if responderMode != "" {
							startEchoResponder(responderMode, clients, statsFile)
						}

						runResult := executeBenchmarkRun(mode, clients, size, rate, repeat, reportFile, statsFile, cfg)
						results = append(results, runResult)
						stopEchoResponder()
					}
				}
			}
		}
		logModeSummary(mode, results[modeStart:])
	}

	return results
}

func executeBenchmarkRun(mode string, clients int, size int, rate int, repeat int, reportFile string, statsFile string, cfg suiteConfig) benchmarkRunResult {
	slimCPUStart, err := readSessionCPUSeconds(slimSession)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	echoCPUStart := 0.0
	if modeUsesSinkMetrics(mode) {
		echoCPUStart, err = readSessionCPUSeconds(echoSession)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}
	runStarted := time.Now()

	benchCmd := exec.Command(
		rateClientPath,
		"-mode", mode,
		"-clients", strconv.Itoa(clients),
		"-dest-sharded",
		"-size", strconv.Itoa(size),
		"-rate", strconv.Itoa(rate),
		"-duration", cfg.DurationDisplay,
		"-local", "agntcy/demo/client",
		"-server", serverEndpoint,
		"-dest", cfg.Destination,
		"-output", reportFile,
	)
	session, err := gexec.Start(benchCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(session, cfg.Duration+20*time.Second).Should(gexec.Exit())
	runElapsed := time.Since(runStarted)
	cpuUsage := collectProcessCPUUsage(session, runElapsed, slimCPUStart, echoCPUStart, modeUsesSinkMetrics(mode))

	time.Sleep(time.Second)
	appendSinkSummary(reportFile, statsFile, modeUsesSinkMetrics(mode))
	appendProcessCPUSummary(reportFile, cpuUsage)

	reportContent, err := os.ReadFile(reportFile)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	sender := parseSenderReport(string(reportContent))

	sink := sinkStats{}
	if modeUsesSinkMetrics(mode) {
		statsContent, readErr := os.ReadFile(statsFile)
		gomega.Expect(readErr).NotTo(gomega.HaveOccurred())
		sink = parseSinkStats(string(statsContent))
	}

	gomega.Expect(session.ExitCode()).To(gomega.Equal(0), "rate-client failed for %s clients=%d size=%d rate=%d repeat=%d", mode, clients, size, rate, repeat)

	result := benchmarkRunResult{
		Mode:                     mode,
		Clients:                  clients,
		Size:                     size,
		Rate:                     rate,
		Repeat:                   repeat,
		SenderTotalMessages:      sender.TotalMessages,
		SenderMPS:                sender.ThroughputMPS,
		SenderMeanLatencyMS:      sender.MeanLatencyMS,
		SenderP50LatencyMS:       sender.P50LatencyMS,
		SenderP90LatencyMS:       sender.P90LatencyMS,
		SenderP99LatencyMS:       sender.P99LatencyMS,
		SenderMaxLatencyMS:       sender.MaxLatencyMS,
		SenderRuntimeErrors:      sender.RuntimeErrors,
		SenderDuration:           sender.ActualDuration,
		SinkReceivedMessages:     sink.ReceivedMessages,
		SinkErrors:               sink.Errors,
		SinkReceiveMPS:           sink.ReceiveMPS,
		SinkActiveReceiveMPS:     sink.ActiveReceiveMPS,
		SinkElapsedSeconds:       sink.ElapsedSeconds,
		SinkActiveReceiveSeconds: sink.ActiveReceiveSeconds,
		SenderCPUSeconds:         cpuUsage.SenderCPUSeconds,
		SenderCPUPercent:         cpuUsage.SenderCPUPercent,
		ResponderCPUSeconds:      cpuUsage.ResponderCPUSeconds,
		ResponderCPUPercent:      cpuUsage.ResponderCPUPercent,
		NodeCPUSeconds:           cpuUsage.NodeCPUSeconds,
		NodeCPUPercent:           cpuUsage.NodeCPUPercent,
		TotalCPUSeconds:          cpuUsage.TotalCPUSeconds,
		TotalCPUPercent:          cpuUsage.TotalCPUPercent,
	}
	logBenchmarkRunResult(result)
	return result
}

func logBenchmarkRunResult(result benchmarkRunResult) {
	format := "BENCHMARK_RESULT mode=%s clients=%d size=%d rate=%d repeat=%d sender_mps=%.2f observed_mps=%.2f sink_mps=%.2f sink_active_mps=%.2f sender_errors=%d sink_errors=%d node_cpu=%.2f total_cpu=%.2f"
	args := []any{
		result.Mode,
		result.Clients,
		result.Size,
		result.Rate,
		result.Repeat,
		result.SenderMPS,
		benchmarkObservedMPS(result),
		result.SinkReceiveMPS,
		result.SinkActiveReceiveMPS,
		result.SenderRuntimeErrors,
		result.SinkErrors,
		result.NodeCPUPercent,
		result.TotalCPUPercent,
	}
	if result.Mode == "request-reply" {
		format += " mean_latency_ms=%.2f p50_latency_ms=%.2f p99_latency_ms=%.2f"
		args = append(args, result.SenderMeanLatencyMS, result.SenderP50LatencyMS, result.SenderP99LatencyMS)
	}
	err := writeProgressLine(format, args...)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func logCapacitySweepStep(mode string, clients int, size int, step capacitySweepStepResult) {
	err := writeProgressLine(
		"CAPACITY_SWEEP_STEP mode=%s clients=%d size=%d phase=%s step=%d rate=%d repeats=%d sender_mean_mps=%.2f observed_mean_mps=%.2f observed_gain_percent=%.2f node_cpu=%.2f total_cpu=%.2f total_errors=%d improved=%t",
		mode,
		clients,
		size,
		defaultIfEmpty(step.Phase, "coarse"),
		step.Step,
		step.Rate,
		step.Repeats,
		step.SenderMeanMPS,
		step.ObservedMeanMPS,
		step.ObservedGainPercent,
		step.NodeMeanCPUPercent,
		step.TotalMeanCPUPercent,
		step.TotalErrors,
		step.Improved,
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func logModeSummary(mode string, rows []benchmarkRunResult) {
	if len(rows) == 0 {
		return
	}

	nodeCPU := computeSampleStats(nodeCPUPercentValues(rows))
	totalCPU := computeSampleStats(totalCPUPercentValues(rows))
	totalErrors := int64(0)
	caseKeys := make(map[string]struct{})
	for _, row := range rows {
		totalErrors += row.SenderRuntimeErrors + row.SinkErrors
		key := fmt.Sprintf("%d/%d/%d", row.Clients, row.Size, row.Rate)
		caseKeys[key] = struct{}{}
	}

	var err error
	if mode == "request-reply" {
		meanLatency := computeSampleStats(senderMeanLatencyMSValues(rows))
		p50Latency := computeSampleStats(senderP50LatencyMSValues(rows))
		p99Latency := computeSampleStats(senderP99LatencyMSValues(rows))
		err = writeProgressLine(
			"MODE_SUMMARY mode=%s runs=%d cases=%d mean_latency_ms=%.2f p50_latency_ms=%.2f p99_latency_ms=%.2f node_cpu=%.2f total_cpu=%.2f total_errors=%d",
			mode,
			len(rows),
			len(caseKeys),
			meanLatency.Mean,
			p50Latency.Mean,
			p99Latency.Mean,
			nodeCPU.Mean,
			totalCPU.Mean,
			totalErrors,
		)
	} else {
		sender := computeSampleStats(senderMPSValues(rows))
		observed := computeSampleStats(observedMPSValues(rows))
		err = writeProgressLine(
			"MODE_SUMMARY mode=%s runs=%d cases=%d sender_mean_mps=%.2f observed_mean_mps=%.2f node_cpu=%.2f total_cpu=%.2f total_errors=%d",
			mode,
			len(rows),
			len(caseKeys),
			sender.Mean,
			observed.Mean,
			nodeCPU.Mean,
			totalCPU.Mean,
			totalErrors,
		)
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func logCapacityCaseSummary(result capacitySweepCaseResult) {
	err := writeProgressLine(
		"CAPACITY_CASE_SUMMARY mode=%s clients=%d size=%d best_offered_rate=%d capacity_rate_lower=%d capacity_rate_upper=%d best_effective_throughput_mps=%.2f best_sender_completed_mps=%.2f best_node_cpu=%.2f best_total_cpu=%.2f steps=%d stop_reason=%q",
		result.Mode,
		result.Clients,
		result.Size,
		result.BestRate,
		result.CapacityRateLower,
		result.CapacityRateUpper,
		result.BestObservedMeanMPS,
		result.BestSenderMeanMPS,
		result.BestNodeCPUPercent,
		result.BestTotalCPUPercent,
		len(result.Steps),
		result.StopReason,
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func writeProgressLine(format string, args ...any) error {
	line := fmt.Sprintf(format, args...)
	if _, err := fmt.Fprintf(os.Stderr, "\n%s\n", line); err != nil {
		return err
	}
	return nil
}

func writeResultsTSV(path string, results []benchmarkRunResult) {
	file, err := os.Create(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = '\t'
	gomega.Expect(writer.Write([]string{
		"mode",
		"clients",
		"size",
		"rate",
		"repeat",
		"sender_total_messages",
		"sender_mps",
		"sender_mean_latency_ms",
		"sender_p50_latency_ms",
		"sender_p90_latency_ms",
		"sender_p99_latency_ms",
		"sender_max_latency_ms",
		"sender_runtime_errors",
		"sender_duration",
		"sink_received_messages",
		"sink_errors",
		"sink_receive_mps",
		"sink_active_receive_mps",
		"sink_elapsed_seconds",
		"sink_active_receive_seconds",
		"sender_cpu_seconds",
		"sender_cpu_percent",
		"responder_cpu_seconds",
		"responder_cpu_percent",
		"node_cpu_seconds",
		"node_cpu_percent",
		"total_cpu_seconds",
		"total_cpu_percent",
	})).To(gomega.Succeed())

	for _, result := range results {
		record := []string{
			result.Mode,
			strconv.Itoa(result.Clients),
			strconv.Itoa(result.Size),
			strconv.Itoa(result.Rate),
			strconv.Itoa(result.Repeat),
			strconv.FormatInt(result.SenderTotalMessages, 10),
			formatFloat(result.SenderMPS),
			formatFloat(result.SenderMeanLatencyMS),
			formatFloat(result.SenderP50LatencyMS),
			formatFloat(result.SenderP90LatencyMS),
			formatFloat(result.SenderP99LatencyMS),
			formatFloat(result.SenderMaxLatencyMS),
			strconv.FormatInt(result.SenderRuntimeErrors, 10),
			result.SenderDuration,
			strconv.FormatInt(result.SinkReceivedMessages, 10),
			strconv.FormatInt(result.SinkErrors, 10),
			formatFloat(result.SinkReceiveMPS),
			formatFloat(result.SinkActiveReceiveMPS),
			formatFloat(result.SinkElapsedSeconds),
			formatFloat(result.SinkActiveReceiveSeconds),
			formatFloat(result.SenderCPUSeconds),
			formatFloat(result.SenderCPUPercent),
			formatFloat(result.ResponderCPUSeconds),
			formatFloat(result.ResponderCPUPercent),
			formatFloat(result.NodeCPUSeconds),
			formatFloat(result.NodeCPUPercent),
			formatFloat(result.TotalCPUSeconds),
			formatFloat(result.TotalCPUPercent),
		}
		gomega.Expect(writer.Write(record)).To(gomega.Succeed())
	}
	writer.Flush()
	gomega.Expect(writer.Error()).NotTo(gomega.HaveOccurred())
}

func buildMeasurementSection(duration string, repeats int) string {
	cpuFormula := "\\text{cpu percent} = 100 \\cdot \\frac{\\text{cpu time consumed during benchmark}}{\\text{benchmark wall-clock duration}}"

	return fmt.Sprintf(`## Measurement Methodology

### Execution Model

Each benchmark case in the matrix is executed %d times. A benchmark case is uniquely identified by:

- mode
- client count
- payload size
- configured rate

For this statistical rerun, each individual run uses a configured sender duration of %s.

### Sender-Side Measurement

Sender throughput is measured by tests/rate-client.

For each run:

1. The sender starts its timed send loop.
2. It records the actual wall-clock run duration.
3. It counts the total number of successfully completed sends.
4. It computes sender throughput as:

$$
\text{sender mps} = \frac{\text{total successful messages}}{\text{actual run duration in seconds}}
$$

### Responder-Side Measurement

For request-reply and fire-and-forget, responder throughput is measured by tests/echo-client.

For each run:

1. The sink counts received messages and received bytes.
2. It records the timestamp of the first payload message received.
3. It records the timestamp of the last payload message received.
4. It computes active receive throughput over the active message window, not over sink process lifetime:

$$
\text{sink mps} = \frac{\text{received messages}}{\text{last message time} - \text{first message time}}
$$

If only one message is observed, the sink falls back to elapsed lifetime-based timing to avoid division by zero.

Write mode does not start a responder. In that mode, the sender-completed write rate is the only throughput measurement and represents how fast the sender can successfully enqueue writes into the node.

### CPU Measurement

CPU usage is collected for the three benchmark processes involved in each run:

- sender process: tests/rate-client
- responder process: tests/echo-client
- node process: slimctl slim start

The sender CPU time is read from the child process state after exit as user time plus system time.

The responder and node CPU time are read as deltas of cumulative process CPU time between the start and end of the benchmark window.

Average CPU percent for each process is computed as:

$$
%s
$$

The total CPU percent for a run is the sum of sender, responder, and node average CPU percent.

### Statistical Treatment

For each case, the report computes:

- mean
- sample variance
- standard deviation
- Student's t 95%% confidence interval for the mean

The sample variance is:

$$
s^2 = \frac{1}{n-1} \sum_{i=1}^n (x_i - \bar{x})^2
$$

The Student's t 95%% confidence interval is:

$$
\bar{x} \pm t_{1-\alpha/2, n-1} \cdot \frac{s}{\sqrt{n}}
$$

where $\alpha = 0.05$ and $n = %d$ for each case in this report.
`, repeats, duration, cpuFormula, repeats)
}

func resetSuiteReports(cfg suiteConfig) {
	gomega.Expect(os.MkdirAll(cfg.RawDir, 0755)).To(gomega.Succeed())
	matches, err := filepath.Glob(filepath.Join(cfg.OutputDir, "report_*.md"))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	for _, match := range matches {
		gomega.Expect(os.Remove(match)).To(gomega.Succeed())
	}
	_ = os.Remove(cfg.SummaryFile)
	_ = os.Remove(cfg.TechnicalReportFile)
	_ = os.Remove(cfg.ResultsTSV)
	_ = os.Remove(cfg.CapacitySweepFile)
	_ = os.RemoveAll(cfg.RawDir)
	gomega.Expect(os.MkdirAll(cfg.RawDir, 0755)).To(gomega.Succeed())
}
