package tests

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/onsi/gomega"
)

func writeSuiteSummary(cfg suiteConfig, results []benchmarkRunResult, capacitySweepResults []capacitySweepCaseResult) {
	sections := buildModeSections(results)
	if len(capacitySweepResults) > 0 {
		sections = append(sections, buildCapacitySweepSummarySection(capacitySweepResults))
	}
	content := renderReportTemplate(filepath.Join(cfg.TemplateDir, "suite_summary.md.tmpl"), map[string]any{
		"Generated":           time.Now().Format("2006-01-02 15:04:05"),
		"Server":              serverEndpoint,
		"Destination":         cfg.Destination,
		"Modes":               cfg.ModesDisplay,
		"Clients":             cfg.ClientsDisplay,
		"Sizes":               cfg.SizesDisplay,
		"RequestRates":        cfg.RequestRatesDisplay,
		"PubRates":            cfg.PubRatesDisplay,
		"WriteRates":          cfg.WriteRatesDisplay,
		"Duration":            cfg.DurationDisplay,
		"Repeats":             cfg.Repeats,
		"ModeSections":        strings.Join(sections, "\n"),
		"HasCapacitySections": len(capacitySweepResults) > 0,
	})
	gomega.Expect(os.WriteFile(cfg.SummaryFile, []byte(content), 0644)).To(gomega.Succeed())
}

func writeTechnicalReport(cfg suiteConfig, results []benchmarkRunResult, capacitySweepResults []capacitySweepCaseResult) {
	sections := buildModeSections(results)
	parts := []string{strings.Join(sections, "\n")}
	if len(capacitySweepResults) > 0 {
		parts = append(parts, buildCapacitySweepTechnicalSection(cfg, capacitySweepResults))
	}
	content := renderReportTemplate(filepath.Join(cfg.TemplateDir, "technical_report.md.tmpl"), map[string]any{
		"Server":               serverEndpoint,
		"Destination":          cfg.Destination,
		"Modes":                cfg.ModesDisplay,
		"Clients":              cfg.ClientsDisplay,
		"Sizes":                cfg.SizesDisplay,
		"RequestRates":         cfg.RequestRatesDisplay,
		"PubRates":             cfg.PubRatesDisplay,
		"WriteRates":           cfg.WriteRatesDisplay,
		"Duration":             cfg.DurationDisplay,
		"Repeats":              cfg.Repeats,
		"CapacitySweepEnabled": cfg.CapacitySweepEnabled,
		"MeasurementSection":   buildMeasurementSection(cfg.DurationDisplay, cfg.Repeats),
		"ModeSections":         strings.Join(sections, "\n"),
		"CapacitySections":     strings.Join(parts[1:], "\n"),
		"HasCapacitySections":  len(capacitySweepResults) > 0,
	})
	gomega.Expect(os.WriteFile(cfg.TechnicalReportFile, []byte(content), 0644)).To(gomega.Succeed())
}

func writeCapacitySweepReport(cfg suiteConfig, results []capacitySweepCaseResult) {
	content := renderReportTemplate(filepath.Join(cfg.TemplateDir, "capacity_sweep.md.tmpl"), map[string]any{
		"Generated":             time.Now().Format("2006-01-02 15:04:05"),
		"Modes":                 cfg.CapacitySweepModesDisplay,
		"Clients":               cfg.CapacitySweepClientsDisplay,
		"Sizes":                 cfg.CapacitySweepSizesDisplay,
		"StartRate":             cfg.CapacitySweepStartRate,
		"MaxRate":               cfg.CapacitySweepMaxRate,
		"GrowthFactor":          fmt.Sprintf("%.2f", cfg.CapacitySweepGrowthFactor),
		"PlateauThreshold":      fmt.Sprintf("%.2f%%", 100*cfg.CapacitySweepPlateauThreshold),
		"PlateauSteps":          cfg.CapacitySweepPlateauSteps,
		"MaxSteps":              cfg.CapacitySweepMaxSteps,
		"RepeatsPerSweepStep":   cfg.CapacitySweepRepeats,
		"CapacitySweepSections": buildCapacitySweepTechnicalSection(cfg, results),
	})
	gomega.Expect(os.WriteFile(cfg.CapacitySweepFile, []byte(content), 0644)).To(gomega.Succeed())
}

func renderReportTemplate(path string, data any) string {
	tmplContent, err := os.ReadFile(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	tmpl, err := template.New(filepath.Base(path)).Parse(string(tmplContent))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	buffer := &bytes.Buffer{}
	gomega.Expect(tmpl.Execute(buffer, data)).To(gomega.Succeed())
	return strings.TrimSpace(buffer.String()) + "\n"
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
	if mode == "request-reply" {
		return buildRequestReplyLatencyTable(rows, mode)
	}
	return buildThroughputModeTable(rows, mode)
}

func buildThroughputModeTable(rows []benchmarkRunResult, mode string) string {
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

func buildRequestReplyLatencyTable(rows []benchmarkRunResult, mode string) string {
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

	lines := []string{
		fmt.Sprintf("### %s Results", modeDisplayTitle(mode)),
		"",
		"Request-reply prioritizes latency statistics. The configured rate is retained as load context, but the primary reported metrics are mean, p50, and p99 latency.",
		"",
		"| Clients | Payload | Rate | Repeats | Mean Latency ms | Mean Latency Variance | Mean Latency 95% CI | P50 Latency ms | P50 Latency 95% CI | P99 Latency ms | P99 Latency 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Total Errors |",
		"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
	}

	for _, key := range keys {
		caseRows := grouped[key]
		meanLatency := computeSampleStats(senderMeanLatencyMSValues(caseRows))
		p50Latency := computeSampleStats(senderP50LatencyMSValues(caseRows))
		p99Latency := computeSampleStats(senderP99LatencyMSValues(caseRows))
		senderCPU := computeSampleStats(senderCPUPercentValues(caseRows))
		nodeCPU := computeSampleStats(nodeCPUPercentValues(caseRows))
		totalCPU := computeSampleStats(totalCPUPercentValues(caseRows))
		totalErrors := int64(0)
		for _, row := range caseRows {
			totalErrors += row.SenderRuntimeErrors + row.SinkErrors
		}
		lines = append(lines, fmt.Sprintf(
			"| %d | %dB | %d | %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %d |",
			key.Clients,
			key.Size,
			key.Rate,
			len(caseRows),
			formatFloat(meanLatency.Mean),
			formatFloat(meanLatency.Variance),
			formatCI(meanLatency),
			formatFloat(p50Latency.Mean),
			formatCI(p50Latency),
			formatFloat(p99Latency.Mean),
			formatCI(p99Latency),
			formatFloat(senderCPU.Mean),
			formatCI(senderCPU),
			formatFloat(nodeCPU.Mean),
			formatCI(nodeCPU),
			formatFloat(totalCPU.Mean),
			formatCI(totalCPU),
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
		"Each row is a separate fixed `(mode, clients, payload)` case. `Best Offered Rate` is the aggregate configured send rate for the best measured throughput point. `Capacity Offered Interval` brackets the offered-rate range where the node appears saturated after the coarse sweep and refinement pass. Sink-backed modes report node-observed throughput, while write mode is reported separately using sender-completed throughput.",
		"",
	}
	if len(sinkBacked) > 0 {
		lines = append(lines,
			"### Sink-Backed Modes",
			"",
			"| Mode | Clients | Payload | Best Offered Rate | Capacity Offered Interval | Best Observed Node Throughput | Observed 95% CI | Best Sender Completed Throughput | Sender 95% CI | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Steps | Stop Reason |",
			"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
		)
	}
	for _, result := range sinkBacked {
		lines = append(lines, fmt.Sprintf(
			"| %s | %d | %dB | %d | [%d, %d] | %s | [%s, %s] | %s | [%s, %s] | %s | [%s, %s] | %s | [%s, %s] | %d | %s |",
			result.Mode,
			result.Clients,
			result.Size,
			result.BestRate,
			result.CapacityRateLower,
			result.CapacityRateUpper,
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
			"| Mode | Clients | Payload | Best Offered Rate | Capacity Offered Interval | Best Sender Write Throughput | Throughput 95% CI | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Steps | Stop Reason |",
			"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
		)
	}
	for _, result := range writeOnly {
		lines = append(lines, fmt.Sprintf(
			"| %s | %d | %dB | %d | [%d, %d] | %s | [%s, %s] | %s | [%s, %s] | %s | [%s, %s] | %d | %s |",
			result.Mode,
			result.Clients,
			result.Size,
			result.BestRate,
			result.CapacityRateLower,
			result.CapacityRateUpper,
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
		"This sweep first increases the configured send rate geometrically to find the saturation region, then performs midpoint refinement to narrow the offered-rate interval that saturates the node.",
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
		fmt.Sprintf("- Refinement steps after coarse sweep: `%d`", cfg.CapacitySweepRefinementSteps),
		fmt.Sprintf("- Minimum offered-rate interval after refinement: `%d` msg/sec", cfg.CapacitySweepMinRateDelta),
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
				fmt.Sprintf("Estimated capacity offered-rate interval: `[%d, %d]` msg/sec", result.CapacityRateLower, result.CapacityRateUpper),
				fmt.Sprintf("Best %s: `%s` msg/sec with 95%% CI [%s, %s]", strings.ToLower(throughputLabel), formatFloat(result.BestObservedMeanMPS), formatFloat(result.BestObservedCILow), formatFloat(result.BestObservedCIHigh)),
				fmt.Sprintf("Best sender-completed throughput: `%s` msg/sec with 95%% CI [%s, %s]", formatFloat(result.BestSenderMeanMPS), formatFloat(result.BestSenderCILow), formatFloat(result.BestSenderCIHigh)),
				fmt.Sprintf("Best node CPU: `%s` %% with 95%% CI [%s, %s]", formatFloat(result.BestNodeCPUPercent), formatFloat(result.BestNodeCILow), formatFloat(result.BestNodeCIHigh)),
				fmt.Sprintf("Best total CPU: `%s` %% with 95%% CI [%s, %s]", formatFloat(result.BestTotalCPUPercent), formatFloat(result.BestTotalCILow), formatFloat(result.BestTotalCIHigh)),
				fmt.Sprintf("Stop reason: %s", result.StopReason),
				"",
				fmt.Sprintf("| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95%% CI | %s | %s 95%% CI | Observed Variance | Observed Gain %% | Improved | Node CPU %% | Node CPU 95%% CI | Total CPU %% | Total CPU 95%% CI | Errors |", throughputLabel, throughputLabel),
				"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
			)
			for _, step := range result.Steps {
				lines = append(lines, fmt.Sprintf(
					"| %d | %s | %d | %d | %s | [%s, %s] | %s | [%s, %s] | %s | %s | %t | %s | [%s, %s] | %s | [%s, %s] | %d |",
					step.Step,
					defaultIfEmpty(step.Phase, "coarse"),
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
