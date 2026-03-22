package tests

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/onsi/gomega"
)

func writeSuiteSummary(cfg suiteConfig, results []benchmarkRunResult, capacitySweepResults []capacitySweepCaseResult) {
	header := []string{
		"# SLIM Benchmark Statistical Summary",
		"",
		fmt.Sprintf("**Generated:** %s", time.Now().Format("2006-01-02 15:04:05")),
		"",
		fmt.Sprintf("**Server:** %s", serverEndpoint),
		fmt.Sprintf("**Destination:** %s", cfg.Destination),
		fmt.Sprintf("**Modes:** %s", cfg.ModesDisplay),
		fmt.Sprintf("**Clients:** %s", cfg.ClientsDisplay),
		fmt.Sprintf("**Sizes:** %s", cfg.SizesDisplay),
		fmt.Sprintf("**Request-Reply Rates:** %s", cfg.RequestRatesDisplay),
		fmt.Sprintf("**One-Way Rates:** %s", cfg.PubRatesDisplay),
		fmt.Sprintf("**Write Rates:** %s", cfg.WriteRatesDisplay),
		fmt.Sprintf("**Duration Per Run:** %s", cfg.DurationDisplay),
		fmt.Sprintf("**Repeats Per Case:** %d", cfg.Repeats),
		"",
		"This summary reports mean, sample variance, and Gaussian 95% confidence intervals over repeated executions of each benchmark case.",
		"",
	}

	sections := buildModeSections(results)
	if len(capacitySweepResults) > 0 {
		sections = append(sections, buildCapacitySweepSummarySection(capacitySweepResults))
	}
	content := strings.Join(append(header, sections...), "\n")
	gomega.Expect(os.WriteFile(cfg.SummaryFile, []byte(content), 0644)).To(gomega.Succeed())
}

func writeTechnicalReport(cfg suiteConfig, results []benchmarkRunResult, capacitySweepResults []capacitySweepCaseResult) {
	sections := buildModeSections(results)
	parts := []string{
		"# SLIM Benchmark Technical Report",
		"",
		"## Scope",
		"",
		"This report documents the repeated benchmark campaign executed by the Ginkgo benchmark suite against a local SLIM node. Each case in the suite matrix is rerun multiple times to estimate mean performance, sample variance, and Gaussian confidence intervals.",
		"",
		"## Test Setup",
		"",
		fmt.Sprintf("- Runtime: local SLIM node on `%s`", serverEndpoint),
		fmt.Sprintf("- Destination identity: `%s`", cfg.Destination),
		"- Sender: `tests/rate-client`",
		"- Sink / responder: `tests/echo-client` (used by request-reply and fire-and-forget; write mode runs without a responder)",
		"- Suite driver: Ginkgo spec in `benchmarks/agntcy-slim/tests/benchmark_suite_test.go`",
		fmt.Sprintf("- Modes: `%s`", cfg.ModesDisplay),
		fmt.Sprintf("- Client counts: `%s`", cfg.ClientsDisplay),
		fmt.Sprintf("- Payload sizes: `%s` bytes", cfg.SizesDisplay),
		fmt.Sprintf("- Request-reply rates: `%s` msg/sec", cfg.RequestRatesDisplay),
		fmt.Sprintf("- One-way rates: `%s`", cfg.PubRatesDisplay),
		fmt.Sprintf("- Write rates: `%s`", cfg.WriteRatesDisplay),
		fmt.Sprintf("- Duration per run: `%s`", cfg.DurationDisplay),
		fmt.Sprintf("- Repeats per case: `%d`", cfg.Repeats),
		fmt.Sprintf("- Adaptive capacity sweep enabled: `%t`", cfg.CapacitySweepEnabled),
		"",
		buildMeasurementSection(cfg.DurationDisplay, cfg.Repeats),
		"",
		"## Test Types",
		"",
		"### Request-Reply",
		"",
		"Request-reply sends one message and waits for the echoed reply before sending the next. It measures paced round-trip behavior.",
		"",
		"### Fire-And-Forget",
		"",
		"Fire-and-forget sends one-way traffic to a sink responder. It measures end-to-end one-way delivery through the node without waiting for per-message replies.",
		"",
		"### Write",
		"",
		"Write measures how fast the sender can successfully write messages into the node without any sink or responder process. In this mode, sender-completed throughput is the primary metric.",
		"",
		"## Full Matrix",
		"",
	}
	parts = append(parts, sections...)
	if len(capacitySweepResults) > 0 {
		parts = append(parts, buildCapacitySweepTechnicalSection(cfg, capacitySweepResults))
	}
	parts = append(parts,
		"## Result Interpretation",
		"",
		"- Sender mean msg/sec represents what the sender completed from its own perspective.",
		"- Observed mean msg/sec represents the sink-observed node throughput for request-reply and fire-and-forget, and the sender-completed write throughput for write mode.",
		"- CPU percentages represent average process CPU utilization during the benchmark window for sender, responder, and node processes.",
		"- Confidence intervals estimate the uncertainty around the mean for each case under repeated execution.",
		"- For request-reply and fire-and-forget, sink throughput remains the better end-to-end capacity indicator when it diverges from sender throughput.",
	)
	gomega.Expect(os.WriteFile(cfg.TechnicalReportFile, []byte(strings.Join(parts, "\n")), 0644)).To(gomega.Succeed())
}

func writeCapacitySweepReport(cfg suiteConfig, results []capacitySweepCaseResult) {
	parts := []string{
		"# SLIM Adaptive Capacity Sweep Report",
		"",
		fmt.Sprintf("**Generated:** %s", time.Now().Format("2006-01-02 15:04:05")),
		"",
		fmt.Sprintf("**Modes:** %s", cfg.CapacitySweepModesDisplay),
		fmt.Sprintf("**Clients:** %s", cfg.CapacitySweepClientsDisplay),
		fmt.Sprintf("**Sizes:** %s", cfg.CapacitySweepSizesDisplay),
		fmt.Sprintf("**Start Rate:** %d", cfg.CapacitySweepStartRate),
		fmt.Sprintf("**Max Rate:** %d", cfg.CapacitySweepMaxRate),
		fmt.Sprintf("**Growth Factor:** %.2f", cfg.CapacitySweepGrowthFactor),
		fmt.Sprintf("**Plateau Threshold:** %.2f%%", 100*cfg.CapacitySweepPlateauThreshold),
		fmt.Sprintf("**Plateau Steps:** %d", cfg.CapacitySweepPlateauSteps),
		fmt.Sprintf("**Max Steps:** %d", cfg.CapacitySweepMaxSteps),
		fmt.Sprintf("**Repeats Per Sweep Step:** %d", cfg.CapacitySweepRepeats),
		"",
		buildCapacitySweepTechnicalSection(cfg, results),
	}
	gomega.Expect(os.WriteFile(cfg.CapacitySweepFile, []byte(strings.Join(parts, "\n")), 0644)).To(gomega.Succeed())
}

func buildModeSections(results []benchmarkRunResult) []string {
	rowsByMode := map[string][]benchmarkRunResult{}
	for _, result := range results {
		rowsByMode[result.Mode] = append(rowsByMode[result.Mode], result)
	}

	sections := make([]string, 0, len(benchmarkModeOrder))
	for _, mode := range benchmarkModeOrder {
		modeRows := rowsByMode[mode]
		if len(modeRows) == 0 {
			continue
		}
		sections = append(sections, buildModeTable(modeRows, mode))
	}
	return sections
}

func buildModeTable(rows []benchmarkRunResult, mode string) string {
	type caseKey struct {
		Clients int
		Size    int
		Rate    int
	}

	grouped := map[caseKey][]benchmarkRunResult{}
	for _, row := range rows {
		key := caseKey{Clients: row.Clients, Size: row.Size, Rate: row.Rate}
		grouped[key] = append(grouped[key], row)
	}

	keys := make([]caseKey, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i int, j int) bool {
		if keys[i].Clients != keys[j].Clients {
			return keys[i].Clients < keys[j].Clients
		}
		if keys[i].Size != keys[j].Size {
			return keys[i].Size < keys[j].Size
		}
		return keys[i].Rate < keys[j].Rate
	})

	throughputLabel := modeObservedThroughputLabel(mode)
	lines := []string{
		fmt.Sprintf("### %s Results", modeDisplayTitle(mode)),
		"",
		fmt.Sprintf("| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95%% CI | %s Mean msg/sec | %s Variance | %s 95%% CI | Sender Mean CPU %% | Sender CPU 95%% CI | Responder Mean CPU %% | Responder CPU 95%% CI | Node Mean CPU %% | Node CPU 95%% CI | Total Mean CPU %% | Total CPU 95%% CI | Mean Sender Msgs | Mean Observed Msgs | Total Errors |", throughputLabel, throughputLabel, throughputLabel),
		"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
	}

	for _, key := range keys {
		caseRows := grouped[key]
		sender := computeSampleStats(senderMPSValues(caseRows))
		observed := computeSampleStats(observedMPSValues(caseRows))
		senderCPU := computeSampleStats(senderCPUPercentValues(caseRows))
		responderCPU := computeSampleStats(responderCPUPercentValues(caseRows))
		nodeCPU := computeSampleStats(nodeCPUPercentValues(caseRows))
		totalCPU := computeSampleStats(totalCPUPercentValues(caseRows))
		senderMessages := computeSampleStats(senderMessageValues(caseRows))
		observedMessages := computeSampleStats(observedMessageValues(caseRows))
		totalErrors := int64(0)
		for _, row := range caseRows {
			totalErrors += row.SenderRuntimeErrors + row.SinkErrors
		}
		lines = append(lines, fmt.Sprintf(
			"| %d | %dB | %d | %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %d |",
			key.Clients,
			key.Size,
			key.Rate,
			len(caseRows),
			formatFloat(sender.Mean),
			formatFloat(sender.Variance),
			formatCI(sender),
			formatFloat(observed.Mean),
			formatFloat(observed.Variance),
			formatCI(observed),
			formatFloat(senderCPU.Mean),
			formatCI(senderCPU),
			formatFloat(responderCPU.Mean),
			formatCI(responderCPU),
			formatFloat(nodeCPU.Mean),
			formatCI(nodeCPU),
			formatFloat(totalCPU.Mean),
			formatCI(totalCPU),
			formatFloat(senderMessages.Mean),
			formatFloat(observedMessages.Mean),
			totalErrors,
		))
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func buildCapacitySweepSummarySection(results []capacitySweepCaseResult) string {
	sinkBacked := make([]capacitySweepCaseResult, 0)
	writeOnly := make([]capacitySweepCaseResult, 0)
	for _, result := range sortCapacitySweepResults(results) {
		if result.Mode == "write" {
			writeOnly = append(writeOnly, result)
			continue
		}
		sinkBacked = append(sinkBacked, result)
	}

	lines := []string{
		"## Adaptive Capacity Sweep Summary",
		"",
		"Each row is a separate fixed `(mode, clients, payload)` case. `Best Offered Rate` is the aggregate configured send rate for the whole node run. Sink-backed modes report node-observed throughput, while write mode is reported separately using sender-completed throughput.",
		"",
	}
	if len(sinkBacked) > 0 {
		lines = append(lines,
			"### Sink-Backed Modes",
			"",
			"| Mode | Clients | Payload | Best Offered Rate | Best Observed Node Throughput | Observed 95% CI | Best Sender Completed Throughput | Sender 95% CI | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Steps | Stop Reason |",
			"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
		)
	}
	for _, result := range sinkBacked {
		lines = append(lines, fmt.Sprintf(
			"| %s | %d | %dB | %d | %s | [%s, %s] | %s | [%s, %s] | %s | [%s, %s] | %s | [%s, %s] | %d | %s |",
			result.Mode,
			result.Clients,
			result.Size,
			result.BestRate,
			formatFloat(result.BestObservedMeanMPS),
			formatFloat(result.BestObservedCILow),
			formatFloat(result.BestObservedCIHigh),
			formatFloat(result.BestSenderMeanMPS),
			formatFloat(result.BestSenderCILow),
			formatFloat(result.BestSenderCIHigh),
			formatFloat(result.BestNodeCPUPercent),
			formatFloat(result.BestNodeCILow),
			formatFloat(result.BestNodeCIHigh),
			formatFloat(result.BestTotalCPUPercent),
			formatFloat(result.BestTotalCILow),
			formatFloat(result.BestTotalCIHigh),
			len(result.Steps),
			result.StopReason,
		))
	}
	if len(writeOnly) > 0 {
		lines = append(lines,
			"",
			"### Write Mode",
			"",
			"| Mode | Clients | Payload | Best Offered Rate | Best Sender Write Throughput | Throughput 95% CI | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Steps | Stop Reason |",
			"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
		)
	}
	for _, result := range writeOnly {
		lines = append(lines, fmt.Sprintf(
			"| %s | %d | %dB | %d | %s | [%s, %s] | %s | [%s, %s] | %s | [%s, %s] | %d | %s |",
			result.Mode,
			result.Clients,
			result.Size,
			result.BestRate,
			formatFloat(result.BestObservedMeanMPS),
			formatFloat(result.BestObservedCILow),
			formatFloat(result.BestObservedCIHigh),
			formatFloat(result.BestNodeCPUPercent),
			formatFloat(result.BestNodeCILow),
			formatFloat(result.BestNodeCIHigh),
			formatFloat(result.BestTotalCPUPercent),
			formatFloat(result.BestTotalCILow),
			formatFloat(result.BestTotalCIHigh),
			len(result.Steps),
			result.StopReason,
		))
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func buildCapacitySweepTechnicalSection(cfg suiteConfig, results []capacitySweepCaseResult) string {
	lines := []string{
		"## Adaptive Capacity Sweep",
		"",
		"This sweep increases the configured send rate geometrically and stops when the mode-specific effective throughput no longer improves by the configured threshold for the configured number of consecutive steps.",
		"Results are reported separately for each fixed `(mode, clients, payload)` case. The reported rate is the aggregate offered load across all clients in that case. For request-reply and fire-and-forget, effective throughput is sink-observed total node throughput. For write mode, effective throughput is sender-completed write throughput because no responder is running.",
		"",
		fmt.Sprintf("- Modes: `%s`", cfg.CapacitySweepModesDisplay),
		fmt.Sprintf("- Clients: `%s`", cfg.CapacitySweepClientsDisplay),
		fmt.Sprintf("- Sizes: `%s` bytes", cfg.CapacitySweepSizesDisplay),
		fmt.Sprintf("- Start rate: `%d` msg/sec", cfg.CapacitySweepStartRate),
		fmt.Sprintf("- Max rate: `%d` msg/sec (0 means unbounded by rate cap)", cfg.CapacitySweepMaxRate),
		fmt.Sprintf("- Growth factor: `%.2f`", cfg.CapacitySweepGrowthFactor),
		fmt.Sprintf("- Plateau threshold: `%.2f%%` effective throughput gain", 100*cfg.CapacitySweepPlateauThreshold),
		fmt.Sprintf("- Plateau steps: `%d`", cfg.CapacitySweepPlateauSteps),
		fmt.Sprintf("- Max steps: `%d`", cfg.CapacitySweepMaxSteps),
		fmt.Sprintf("- Repeats per sweep step: `%d`", cfg.CapacitySweepRepeats),
		"",
	}

	appendCapacitySweepModeSection := func(title string, sectionResults []capacitySweepCaseResult) {
		if len(sectionResults) == 0 {
			return
		}
		lines = append(lines, fmt.Sprintf("### %s", title), "")
		for _, result := range sectionResults {
			throughputLabel := modeObservedThroughputLabel(result.Mode)
			lines = append(lines,
				fmt.Sprintf("#### %s Clients=%d Payload=%dB", modeDisplayTitle(result.Mode), result.Clients, result.Size),
				"",
				fmt.Sprintf("Best offered aggregate rate: `%d` msg/sec", result.BestRate),
				fmt.Sprintf("Best %s: `%s` msg/sec with 95%% CI [%s, %s]", strings.ToLower(throughputLabel), formatFloat(result.BestObservedMeanMPS), formatFloat(result.BestObservedCILow), formatFloat(result.BestObservedCIHigh)),
				fmt.Sprintf("Best sender-completed throughput: `%s` msg/sec with 95%% CI [%s, %s]", formatFloat(result.BestSenderMeanMPS), formatFloat(result.BestSenderCILow), formatFloat(result.BestSenderCIHigh)),
				fmt.Sprintf("Best node CPU: `%s` %% with 95%% CI [%s, %s]", formatFloat(result.BestNodeCPUPercent), formatFloat(result.BestNodeCILow), formatFloat(result.BestNodeCIHigh)),
				fmt.Sprintf("Best total CPU: `%s` %% with 95%% CI [%s, %s]", formatFloat(result.BestTotalCPUPercent), formatFloat(result.BestTotalCILow), formatFloat(result.BestTotalCIHigh)),
				fmt.Sprintf("Stop reason: %s", result.StopReason),
				"",
				fmt.Sprintf("| Step | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95%% CI | %s | %s 95%% CI | Observed Variance | Observed Gain %% | Improved | Node CPU %% | Node CPU 95%% CI | Total CPU %% | Total CPU 95%% CI | Errors |", throughputLabel, throughputLabel),
				"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
			)
			for _, step := range result.Steps {
				lines = append(lines, fmt.Sprintf(
					"| %d | %d | %d | %s | [%s, %s] | %s | [%s, %s] | %s | %s | %t | %s | [%s, %s] | %s | [%s, %s] | %d |",
					step.Step,
					step.Rate,
					step.Repeats,
					formatFloat(step.SenderMeanMPS),
					formatFloat(step.SenderCILow),
					formatFloat(step.SenderCIHigh),
					formatFloat(step.ObservedMeanMPS),
					formatFloat(step.ObservedCILow),
					formatFloat(step.ObservedCIHigh),
					formatFloat(step.ObservedVariance),
					formatFloat(step.ObservedGainPercent),
					step.Improved,
					formatFloat(step.NodeMeanCPUPercent),
					formatFloat(step.NodeCILow),
					formatFloat(step.NodeCIHigh),
					formatFloat(step.TotalMeanCPUPercent),
					formatFloat(step.TotalCILow),
					formatFloat(step.TotalCIHigh),
					step.TotalErrors,
				))
			}
			lines = append(lines, "")
		}
	}

	sinkBacked := make([]capacitySweepCaseResult, 0)
	writeOnly := make([]capacitySweepCaseResult, 0)
	for _, result := range sortCapacitySweepResults(results) {
		if result.Mode == "write" {
			writeOnly = append(writeOnly, result)
			continue
		}
		sinkBacked = append(sinkBacked, result)
	}
	appendCapacitySweepModeSection("Sink-Backed Modes", sinkBacked)
	appendCapacitySweepModeSection("Write Mode", writeOnly)
	return strings.Join(lines, "\n")
}

func sortCapacitySweepResults(results []capacitySweepCaseResult) []capacitySweepCaseResult {
	sortedResults := append([]capacitySweepCaseResult(nil), results...)
	sort.Slice(sortedResults, func(i int, j int) bool {
		if sortedResults[i].Mode != sortedResults[j].Mode {
			return sortedResults[i].Mode < sortedResults[j].Mode
		}
		if sortedResults[i].Clients != sortedResults[j].Clients {
			return sortedResults[i].Clients < sortedResults[j].Clients
		}
		return sortedResults[i].Size < sortedResults[j].Size
	})
	return sortedResults
}
