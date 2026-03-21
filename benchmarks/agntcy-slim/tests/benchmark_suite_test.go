// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type suiteConfig struct {
	OutputDir                     string
	RawDir                        string
	SummaryFile                   string
	TechnicalReportFile           string
	ResultsTSV                    string
	CapacitySweepFile             string
	Sizes                         []int
	Clients                       []int
	Modes                         []string
	RequestRates                  []int
	PubRates                      []int
	PubRatesDisplay               string
	PubRateAutoProfile            bool
	Duration                      time.Duration
	DurationDisplay               string
	Repeats                       int
	ServerEndpoint                string
	Destination                   string
	ModesDisplay                  string
	ClientsDisplay                string
	SizesDisplay                  string
	RequestRatesDisplay           string
	CapacitySweepEnabled          bool
	CapacitySweepModes            []string
	CapacitySweepClients          []int
	CapacitySweepSizes            []int
	CapacitySweepStartRate        int
	CapacitySweepMaxRate          int
	CapacitySweepGrowthFactor     float64
	CapacitySweepPlateauThreshold float64
	CapacitySweepPlateauSteps     int
	CapacitySweepMaxSteps         int
	CapacitySweepRepeats          int
	CapacitySweepModesDisplay     string
	CapacitySweepClientsDisplay   string
	CapacitySweepSizesDisplay     string
}

type benchmarkRunResult struct {
	Mode                     string
	Clients                  int
	Size                     int
	Rate                     int
	Repeat                   int
	SenderTotalMessages      int64
	SenderMPS                float64
	SenderRuntimeErrors      int64
	SenderDuration           string
	SinkReceivedMessages     int64
	SinkErrors               int64
	SinkReceiveMPS           float64
	SinkActiveReceiveMPS     float64
	SinkElapsedSeconds       float64
	SinkActiveReceiveSeconds float64
	SenderCPUSeconds         float64
	SenderCPUPercent         float64
	ResponderCPUSeconds      float64
	ResponderCPUPercent      float64
	NodeCPUSeconds           float64
	NodeCPUPercent           float64
	TotalCPUSeconds          float64
	TotalCPUPercent          float64
}

type processCPUUsage struct {
	SenderCPUSeconds    float64
	SenderCPUPercent    float64
	ResponderCPUSeconds float64
	ResponderCPUPercent float64
	NodeCPUSeconds      float64
	NodeCPUPercent      float64
	TotalCPUSeconds     float64
	TotalCPUPercent     float64
}

type senderReport struct {
	TotalMessages  int64
	ThroughputMPS  float64
	RuntimeErrors  int64
	ActualDuration string
}

type sinkStats struct {
	Mode                 string
	ReceivedMessages     int64
	ReceivedBytes        int64
	ReplyMessages        int64
	Errors               int64
	WarmupMessages       int64
	WarmupReplies        int64
	DrainMessages        int64
	DrainReplies         int64
	ElapsedSeconds       float64
	ActiveReceiveSeconds float64
	ReceiveMPS           float64
	ReceiveMBPS          float64
	ActiveReceiveMPS     float64
	ActiveReceiveMBPS    float64
}

type sampleStats struct {
	Count    int
	Mean     float64
	Variance float64
	StdDev   float64
	CILow    float64
	CIHigh   float64
}

type capacitySweepStepResult struct {
	Step                int
	Rate                int
	Repeats             int
	SenderMeanMPS       float64
	SinkMeanMPS         float64
	SinkVariance        float64
	NodeMeanCPUPercent  float64
	TotalMeanCPUPercent float64
	TotalErrors         int64
	SinkGainPercent     float64
	Improved            bool
}

type capacitySweepCaseResult struct {
	Mode                string
	Clients             int
	Size                int
	BestRate            int
	BestSinkMeanMPS     float64
	BestSenderMeanMPS   float64
	BestNodeCPUPercent  float64
	BestTotalCPUPercent float64
	Steps               []capacitySweepStepResult
	StopReason          string
}

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
	outputDir, err := filepath.Abs("../tools/slim-bench/reports")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	rawDir := filepath.Join(outputDir, "raw")
	duration := envDuration("DURATION", 5*time.Second)
	modes := envStringList("MODES", []string{"request", "ping-pong", "pub"})
	clients := envIntList("CLIENTS", []int{1, 10, 50})
	sizes := envIntList("SIZES", []int{16, 128, 1024, 10240})
	requestRates := envIntList("REQUEST_RATES", []int{1000})
	pubRatesRaw := strings.TrimSpace(os.Getenv("PUB_RATES"))
	pubRates := []int(nil)
	pubRatesDisplay := "auto (safe default profile)"
	pubAutoProfile := true
	if pubRatesRaw != "" {
		pubRates = mustParseIntList(pubRatesRaw)
		pubRatesDisplay = joinInts(pubRates)
		pubAutoProfile = false
	}

	capacitySweepEnabled := envBool("CAPACITY_SWEEP", false)
	capacitySweepModes := envStringList("CAPACITY_SWEEP_MODES", []string{"pub"})
	capacitySweepClients := envIntList("CAPACITY_SWEEP_CLIENTS", clients)
	capacitySweepSizes := envIntList("CAPACITY_SWEEP_SIZES", sizes)
	capacitySweepStartRate := envInt("CAPACITY_SWEEP_START_RATE", 1000)
	capacitySweepMaxRate := envInt("CAPACITY_SWEEP_MAX_RATE", 0)
	capacitySweepGrowthFactor := envFloat("CAPACITY_SWEEP_GROWTH_FACTOR", 2.0)
	capacitySweepPlateauThreshold := envFloat("CAPACITY_SWEEP_PLATEAU_THRESHOLD", 0.05)
	capacitySweepPlateauSteps := envInt("CAPACITY_SWEEP_PLATEAU_STEPS", 2)
	capacitySweepMaxSteps := envInt("CAPACITY_SWEEP_MAX_STEPS", 8)
	capacitySweepRepeats := envInt("CAPACITY_SWEEP_REPEATS", 1)

	return suiteConfig{
		OutputDir:                     outputDir,
		RawDir:                        rawDir,
		SummaryFile:                   filepath.Join(outputDir, "suite_summary.md"),
		TechnicalReportFile:           filepath.Join(outputDir, "technical_report.md"),
		ResultsTSV:                    filepath.Join(outputDir, "results.tsv"),
		CapacitySweepFile:             filepath.Join(outputDir, "capacity_sweep.md"),
		Sizes:                         sizes,
		Clients:                       clients,
		Modes:                         modes,
		RequestRates:                  requestRates,
		PubRates:                      pubRates,
		PubRatesDisplay:               pubRatesDisplay,
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
		CapacitySweepModesDisplay:     strings.Join(capacitySweepModes, " "),
		CapacitySweepClientsDisplay:   joinInts(capacitySweepClients),
		CapacitySweepSizesDisplay:     joinInts(capacitySweepSizes),
	}
}

func runSuiteMatrix(cfg suiteConfig) []benchmarkRunResult {
	results := make([]benchmarkRunResult, 0)
	for _, mode := range cfg.Modes {
		modeStart := len(results)
		gomega.Expect(mode).To(gomega.Or(gomega.Equal("request"), gomega.Equal("ping-pong"), gomega.Equal("pub")))
		for _, clients := range cfg.Clients {
			for _, size := range cfg.Sizes {
				rateValues := cfg.RequestRates
				if mode == "pub" {
					rateValues = pubRatesForCase(cfg, clients, size)
				}

				for _, rate := range rateValues {
					ginkgo.By(fmt.Sprintf("running %s clients=%d size=%d rate=%d repeats=%d", mode, clients, size, rate, cfg.Repeats))
					responderMode := "echo"
					if mode == "pub" {
						responderMode = "sink"
					}

					for repeat := 1; repeat <= cfg.Repeats; repeat++ {
						reportFile := filepath.Join(cfg.RawDir, fmt.Sprintf("report_%s_c%d_s%d_r%d_rep%02d.md", mode, clients, size, rate, repeat))
						statsFile := filepath.Join(buildDir, fmt.Sprintf("stats_%s_c%d_s%d_r%d_rep%02d.txt", mode, clients, size, rate, repeat))

						stopEchoResponder()
						startEchoResponder(responderMode, clients, statsFile)

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

func runCapacitySweep(cfg suiteConfig) []capacitySweepCaseResult {
	results := make([]capacitySweepCaseResult, 0)
	for _, mode := range cfg.CapacitySweepModes {
		gomega.Expect(mode).To(gomega.Or(gomega.Equal("pub"), gomega.Equal("request"), gomega.Equal("ping-pong")))
		for _, clients := range cfg.CapacitySweepClients {
			for _, size := range cfg.CapacitySweepSizes {
				ginkgo.By(fmt.Sprintf("running adaptive capacity sweep %s clients=%d size=%d", mode, clients, size))
				caseResult := runCapacitySweepCase(mode, clients, size, cfg)
				logCapacityCaseSummary(caseResult)
				results = append(results, caseResult)
			}
		}
	}
	return results
}

func runCapacitySweepCase(mode string, clients int, size int, cfg suiteConfig) capacitySweepCaseResult {
	rate := cfg.CapacitySweepStartRate
	if rate <= 0 {
		rate = 1
	}

	caseResult := capacitySweepCaseResult{
		Mode:    mode,
		Clients: clients,
		Size:    size,
		Steps:   make([]capacitySweepStepResult, 0, cfg.CapacitySweepMaxSteps),
	}
	bestSink := -1.0
	bestIndex := -1
	plateauCount := 0

	for step := 1; step <= cfg.CapacitySweepMaxSteps; step++ {
		current := runCapacitySweepStep(mode, clients, size, rate, step, cfg)
		if bestSink > 0 {
			current.SinkGainPercent = 100 * (current.SinkMeanMPS - bestSink) / bestSink
		}
		if bestSink <= 0 || current.SinkMeanMPS > bestSink*(1+cfg.CapacitySweepPlateauThreshold) {
			current.Improved = true
			bestSink = current.SinkMeanMPS
			bestIndex = len(caseResult.Steps)
			plateauCount = 0
		} else {
			plateauCount++
		}
		caseResult.Steps = append(caseResult.Steps, current)

		if bestIndex >= 0 && current.SinkMeanMPS < bestSink {
			caseResult.StopReason = fmt.Sprintf("sink throughput regressed at rate %d; effective capacity remains at best prior rate %d", current.Rate, caseResult.Steps[bestIndex].Rate)
			logCapacitySweepStep(mode, clients, size, current)
			break
		}

		logCapacitySweepStep(mode, clients, size, current)

		if cfg.CapacitySweepMaxRate > 0 && rate >= cfg.CapacitySweepMaxRate {
			caseResult.StopReason = fmt.Sprintf("reached configured max rate %d", cfg.CapacitySweepMaxRate)
			if bestIndex >= 0 && caseResult.Steps[bestIndex].Rate != rate {
				caseResult.StopReason = fmt.Sprintf("reached configured max rate %d; effective capacity remained at best prior rate %d", cfg.CapacitySweepMaxRate, caseResult.Steps[bestIndex].Rate)
			}
			break
		}
		if plateauCount >= cfg.CapacitySweepPlateauSteps {
			caseResult.StopReason = fmt.Sprintf("sink throughput plateaued for %d consecutive steps", cfg.CapacitySweepPlateauSteps)
			break
		}
		if step == cfg.CapacitySweepMaxSteps {
			caseResult.StopReason = fmt.Sprintf("reached configured max steps %d", cfg.CapacitySweepMaxSteps)
			break
		}

		rate = nextSweepRate(rate, cfg.CapacitySweepGrowthFactor, cfg.CapacitySweepMaxRate)
	}

	if bestIndex >= 0 {
		best := caseResult.Steps[bestIndex]
		caseResult.BestRate = best.Rate
		caseResult.BestSinkMeanMPS = best.SinkMeanMPS
		caseResult.BestSenderMeanMPS = best.SenderMeanMPS
		caseResult.BestNodeCPUPercent = best.NodeMeanCPUPercent
		caseResult.BestTotalCPUPercent = best.TotalMeanCPUPercent
	}
	if caseResult.StopReason == "" {
		caseResult.StopReason = "completed sweep"
	}
	return caseResult
}

func runCapacitySweepStep(mode string, clients int, size int, rate int, step int, cfg suiteConfig) capacitySweepStepResult {
	responderMode := "echo"
	if mode == "pub" {
		responderMode = "sink"
	}

	stepRuns := make([]benchmarkRunResult, 0, cfg.CapacitySweepRepeats)
	for repeat := 1; repeat <= cfg.CapacitySweepRepeats; repeat++ {
		reportFile := filepath.Join(cfg.RawDir, fmt.Sprintf("sweep_%s_c%d_s%d_step%02d_r%d_rep%02d.md", mode, clients, size, step, rate, repeat))
		statsFile := filepath.Join(buildDir, fmt.Sprintf("sweep_stats_%s_c%d_s%d_step%02d_r%d_rep%02d.txt", mode, clients, size, step, rate, repeat))

		stopEchoResponder()
		startEchoResponder(responderMode, clients, statsFile)
		stepRuns = append(stepRuns, executeBenchmarkRun(mode, clients, size, rate, repeat, reportFile, statsFile, cfg))
		stopEchoResponder()
	}

	sender := computeSampleStats(senderMPSValues(stepRuns))
	sink := computeSampleStats(sinkMPSValues(stepRuns))
	nodeCPU := computeSampleStats(nodeCPUPercentValues(stepRuns))
	totalCPU := computeSampleStats(totalCPUPercentValues(stepRuns))
	totalErrors := int64(0)
	for _, run := range stepRuns {
		totalErrors += run.SenderRuntimeErrors + run.SinkErrors
	}

	return capacitySweepStepResult{
		Step:                step,
		Rate:                rate,
		Repeats:             cfg.CapacitySweepRepeats,
		SenderMeanMPS:       sender.Mean,
		SinkMeanMPS:         sink.Mean,
		SinkVariance:        sink.Variance,
		NodeMeanCPUPercent:  nodeCPU.Mean,
		TotalMeanCPUPercent: totalCPU.Mean,
		TotalErrors:         totalErrors,
	}
}

func nextSweepRate(current int, growthFactor float64, maxRate int) int {
	next := int(math.Ceil(float64(current) * growthFactor))
	if next <= current {
		next = current + 1
	}
	if maxRate > 0 && next > maxRate {
		next = maxRate
	}
	return next
}

func executeBenchmarkRun(mode string, clients int, size int, rate int, repeat int, reportFile string, statsFile string, cfg suiteConfig) benchmarkRunResult {
	slimCPUStart, err := readSessionCPUSeconds(slimSession)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	echoCPUStart, err := readSessionCPUSeconds(echoSession)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
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
	cpuUsage := collectProcessCPUUsage(session, runElapsed, slimCPUStart, echoCPUStart)

	time.Sleep(time.Second)
	appendSinkSummary(reportFile, statsFile)
	appendProcessCPUSummary(reportFile, cpuUsage)

	reportContent, err := os.ReadFile(reportFile)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	sender := parseSenderReport(string(reportContent))

	statsContent, err := os.ReadFile(statsFile)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	sink := parseSinkStats(string(statsContent))

	gomega.Expect(session.ExitCode()).To(gomega.Equal(0), "rate-client failed for %s clients=%d size=%d rate=%d repeat=%d", mode, clients, size, rate, repeat)

	result := benchmarkRunResult{
		Mode:                     mode,
		Clients:                  clients,
		Size:                     size,
		Rate:                     rate,
		Repeat:                   repeat,
		SenderTotalMessages:      sender.TotalMessages,
		SenderMPS:                sender.ThroughputMPS,
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
	err := writeProgressLine(
		"BENCHMARK_RESULT mode=%s clients=%d size=%d rate=%d repeat=%d sender_mps=%.2f sink_mps=%.2f sink_active_mps=%.2f sender_errors=%d sink_errors=%d node_cpu=%.2f total_cpu=%.2f",
		result.Mode,
		result.Clients,
		result.Size,
		result.Rate,
		result.Repeat,
		result.SenderMPS,
		result.SinkReceiveMPS,
		result.SinkActiveReceiveMPS,
		result.SenderRuntimeErrors,
		result.SinkErrors,
		result.NodeCPUPercent,
		result.TotalCPUPercent,
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func logCapacitySweepStep(mode string, clients int, size int, step capacitySweepStepResult) {
	err := writeProgressLine(
		"CAPACITY_SWEEP_STEP mode=%s clients=%d size=%d step=%d rate=%d repeats=%d sender_mean_mps=%.2f sink_mean_mps=%.2f sink_gain_percent=%.2f node_cpu=%.2f total_cpu=%.2f total_errors=%d improved=%t",
		mode,
		clients,
		size,
		step.Step,
		step.Rate,
		step.Repeats,
		step.SenderMeanMPS,
		step.SinkMeanMPS,
		step.SinkGainPercent,
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

	sender := computeSampleStats(senderMPSValues(rows))
	sink := computeSampleStats(sinkMPSValues(rows))
	nodeCPU := computeSampleStats(nodeCPUPercentValues(rows))
	totalCPU := computeSampleStats(totalCPUPercentValues(rows))
	totalErrors := int64(0)
	caseKeys := make(map[string]struct{})
	for _, row := range rows {
		totalErrors += row.SenderRuntimeErrors + row.SinkErrors
		key := fmt.Sprintf("%d/%d/%d", row.Clients, row.Size, row.Rate)
		caseKeys[key] = struct{}{}
	}

	err := writeProgressLine(
		"MODE_SUMMARY mode=%s runs=%d cases=%d sender_mean_mps=%.2f sink_mean_mps=%.2f node_cpu=%.2f total_cpu=%.2f total_errors=%d",
		mode,
		len(rows),
		len(caseKeys),
		sender.Mean,
		sink.Mean,
		nodeCPU.Mean,
		totalCPU.Mean,
		totalErrors,
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func logCapacityCaseSummary(result capacitySweepCaseResult) {
	err := writeProgressLine(
		"CAPACITY_CASE_SUMMARY mode=%s clients=%d size=%d best_offered_rate=%d best_node_throughput_mps=%.2f best_sender_completed_mps=%.2f best_node_cpu=%.2f best_total_cpu=%.2f steps=%d stop_reason=%q",
		result.Mode,
		result.Clients,
		result.Size,
		result.BestRate,
		result.BestSinkMeanMPS,
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
		fmt.Sprintf("**Request Rates:** %s", cfg.RequestRatesDisplay),
		fmt.Sprintf("**Pub Rates:** %s", cfg.PubRatesDisplay),
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
		"- Sink / responder: `tests/echo-client`",
		"- Suite driver: Ginkgo spec in `benchmarks/agntcy-slim/tests/benchmark_suite_test.go`",
		fmt.Sprintf("- Modes: `%s`", cfg.ModesDisplay),
		fmt.Sprintf("- Client counts: `%s`", cfg.ClientsDisplay),
		fmt.Sprintf("- Payload sizes: `%s` bytes", cfg.SizesDisplay),
		fmt.Sprintf("- Request rates: `%s` msg/sec", cfg.RequestRatesDisplay),
		fmt.Sprintf("- Pub rates: `%s`", cfg.PubRatesDisplay),
		fmt.Sprintf("- Duration per run: `%s`", cfg.DurationDisplay),
		fmt.Sprintf("- Repeats per case: `%d`", cfg.Repeats),
		fmt.Sprintf("- Adaptive capacity sweep enabled: `%t`", cfg.CapacitySweepEnabled),
		"",
		buildMeasurementSection(cfg.DurationDisplay, cfg.Repeats),
		"",
		"## Test Types",
		"",
		"### Request",
		"",
		"Request mode sends one message and waits for a reply before sending the next. It measures paced round-trip behavior.",
		"",
		"### Ping-Pong",
		"",
		"Ping-pong mode exercises the same round-trip pattern through the ping-pong alias while preserving paced operation.",
		"",
		"### Pub",
		"",
		"Pub mode measures one-way publish behavior. In the suite it uses the configured or auto-selected pub rate for each case.",
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
		"- Sink mean msg/sec represents what the sink sustained over the active receive window.",
		"- CPU percentages represent average process CPU utilization during the benchmark window for sender, responder, and node processes.",
		"- Confidence intervals estimate the uncertainty around the mean for each case under repeated execution.",
		"- When sender and sink results diverge, sink throughput is the better end-to-end capacity indicator.",
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

	sections := make([]string, 0, 3)
	for _, mode := range []string{"request", "ping-pong", "pub"} {
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

	lines := []string{
		fmt.Sprintf("### %s Results", strings.Title(mode)),
		"",
		"| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95% CI | Sink Mean msg/sec | Sink Variance | Sink 95% CI | Sender Mean CPU % | Responder Mean CPU % | Node Mean CPU % | Total Mean CPU % | Mean Sender Msgs | Mean Sink Msgs | Total Errors |",
		"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
	}

	for _, key := range keys {
		caseRows := grouped[key]
		sender := computeSampleStats(senderMPSValues(caseRows))
		sink := computeSampleStats(sinkMPSValues(caseRows))
		senderCPU := computeSampleStats(senderCPUPercentValues(caseRows))
		responderCPU := computeSampleStats(responderCPUPercentValues(caseRows))
		nodeCPU := computeSampleStats(nodeCPUPercentValues(caseRows))
		totalCPU := computeSampleStats(totalCPUPercentValues(caseRows))
		senderMessages := computeSampleStats(senderMessageValues(caseRows))
		sinkMessages := computeSampleStats(sinkMessageValues(caseRows))
		totalErrors := int64(0)
		for _, row := range caseRows {
			totalErrors += row.SenderRuntimeErrors + row.SinkErrors
		}
		lines = append(lines, fmt.Sprintf(
			"| %d | %dB | %d | %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %d |",
			key.Clients,
			key.Size,
			key.Rate,
			len(caseRows),
			formatFloat(sender.Mean),
			formatFloat(sender.Variance),
			formatCI(sender),
			formatFloat(sink.Mean),
			formatFloat(sink.Variance),
			formatCI(sink),
			formatFloat(senderCPU.Mean),
			formatFloat(responderCPU.Mean),
			formatFloat(nodeCPU.Mean),
			formatFloat(totalCPU.Mean),
			formatFloat(senderMessages.Mean),
			formatFloat(sinkMessages.Mean),
			totalErrors,
		))
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func buildCapacitySweepSummarySection(results []capacitySweepCaseResult) string {
	lines := []string{
		"## Adaptive Capacity Sweep Summary",
		"",
		"Each row is a separate fixed `(mode, clients, payload)` case. `Best Offered Rate` is the aggregate configured send rate for the whole node run, and `Best Observed Node Throughput` is the sink-observed total node throughput at the best step for that case.",
		"",
		"| Mode | Clients | Payload | Best Offered Rate | Best Observed Node Throughput | Best Sender Completed Throughput | Node CPU % | Total CPU % | Steps | Stop Reason |",
		"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
	}
	for _, result := range sortCapacitySweepResults(results) {
		lines = append(lines, fmt.Sprintf(
			"| %s | %d | %dB | %d | %s | %s | %s | %s | %d | %s |",
			result.Mode,
			result.Clients,
			result.Size,
			result.BestRate,
			formatFloat(result.BestSinkMeanMPS),
			formatFloat(result.BestSenderMeanMPS),
			formatFloat(result.BestNodeCPUPercent),
			formatFloat(result.BestTotalCPUPercent),
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
		"This sweep increases the configured send rate geometrically and stops when sink throughput no longer improves by the configured threshold for the configured number of consecutive steps.",
		"Results are reported separately for each fixed `(mode, clients, payload)` case. The reported rate is the aggregate offered load across all clients in that case, while sink throughput is the observed total node throughput.",
		"",
		fmt.Sprintf("- Modes: `%s`", cfg.CapacitySweepModesDisplay),
		fmt.Sprintf("- Clients: `%s`", cfg.CapacitySweepClientsDisplay),
		fmt.Sprintf("- Sizes: `%s` bytes", cfg.CapacitySweepSizesDisplay),
		fmt.Sprintf("- Start rate: `%d` msg/sec", cfg.CapacitySweepStartRate),
		fmt.Sprintf("- Max rate: `%d` msg/sec (0 means unbounded by rate cap)", cfg.CapacitySweepMaxRate),
		fmt.Sprintf("- Growth factor: `%.2f`", cfg.CapacitySweepGrowthFactor),
		fmt.Sprintf("- Plateau threshold: `%.2f%%` sink throughput gain", 100*cfg.CapacitySweepPlateauThreshold),
		fmt.Sprintf("- Plateau steps: `%d`", cfg.CapacitySweepPlateauSteps),
		fmt.Sprintf("- Max steps: `%d`", cfg.CapacitySweepMaxSteps),
		fmt.Sprintf("- Repeats per sweep step: `%d`", cfg.CapacitySweepRepeats),
		"",
	}
	for _, result := range sortCapacitySweepResults(results) {
		lines = append(lines,
			fmt.Sprintf("### %s Clients=%d Payload=%dB", strings.Title(result.Mode), result.Clients, result.Size),
			"",
			fmt.Sprintf("Best offered aggregate rate: `%d` msg/sec", result.BestRate),
			fmt.Sprintf("Best observed node throughput: `%s` msg/sec", formatFloat(result.BestSinkMeanMPS)),
			fmt.Sprintf("Best sender-completed throughput: `%s` msg/sec", formatFloat(result.BestSenderMeanMPS)),
			fmt.Sprintf("Stop reason: %s", result.StopReason),
			"",
			"| Step | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Observed Node Throughput | Sink Variance | Sink Gain % | Improved | Node CPU % | Total CPU % | Errors |",
			"| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
		)
		for _, step := range result.Steps {
			lines = append(lines, fmt.Sprintf(
				"| %d | %d | %d | %s | %s | %s | %s | %t | %s | %s | %d |",
				step.Step,
				step.Rate,
				step.Repeats,
				formatFloat(step.SenderMeanMPS),
				formatFloat(step.SinkMeanMPS),
				formatFloat(step.SinkVariance),
				formatFloat(step.SinkGainPercent),
				step.Improved,
				formatFloat(step.NodeMeanCPUPercent),
				formatFloat(step.TotalMeanCPUPercent),
				step.TotalErrors,
			))
		}
		lines = append(lines, "")
	}
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

func senderMPSValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.SenderMPS)
	}
	return values
}

func sinkMPSValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.SinkActiveReceiveMPS)
	}
	return values
}

func senderMessageValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, float64(row.SenderTotalMessages))
	}
	return values
}

func sinkMessageValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, float64(row.SinkReceivedMessages))
	}
	return values
}

func senderCPUPercentValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.SenderCPUPercent)
	}
	return values
}

func responderCPUPercentValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.ResponderCPUPercent)
	}
	return values
}

func nodeCPUPercentValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.NodeCPUPercent)
	}
	return values
}

func totalCPUPercentValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.TotalCPUPercent)
	}
	return values
}

func computeSampleStats(values []float64) sampleStats {
	count := len(values)
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(count)

	variance := 0.0
	if count > 1 {
		for _, value := range values {
			delta := value - mean
			variance += delta * delta
		}
		variance /= float64(count - 1)
	}
	stddev := math.Sqrt(variance)
	margin := 0.0
	if count > 0 {
		margin = 1.96 * stddev / math.Sqrt(float64(count))
	}
	return sampleStats{
		Count:    count,
		Mean:     mean,
		Variance: variance,
		StdDev:   stddev,
		CILow:    math.Max(0, mean-margin),
		CIHigh:   mean + margin,
	}
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

### Sink-Side Measurement

Sink throughput is measured by tests/echo-client.

For each run:

1. The sink counts received messages and received bytes.
2. It records the timestamp of the first payload message received.
3. It records the timestamp of the last payload message received.
4. It computes active receive throughput over the active message window, not over sink process lifetime:

$$
\text{sink mps} = \frac{\text{received messages}}{\text{last message time} - \text{first message time}}
$$

If only one message is observed, the sink falls back to elapsed lifetime-based timing to avoid division by zero.

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
- Gaussian 95%% confidence interval for the mean

The sample variance is:

$$
s^2 = \frac{1}{n-1} \sum_{i=1}^n (x_i - \bar{x})^2
$$

The Gaussian 95%% confidence interval is:

$$
\bar{x} \pm 1.96 \cdot \frac{s}{\sqrt{n}}
$$

where $n = %d$ for each case in this report.
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

func pubRatesForCase(cfg suiteConfig, clients int, size int) []int {
	if !cfg.PubRateAutoProfile {
		return cfg.PubRates
	}
	return []int{pubRateForCase(clients, size)}
}

func pubRateForCase(clients int, size int) int {
	if size >= 10240 {
		switch {
		case clients >= 50:
			return 100
		case clients >= 10:
			return 200
		default:
			return 500
		}
	}
	if size >= 1024 {
		switch {
		case clients >= 50:
			return 250
		case clients >= 10:
			return 500
		default:
			return 1000
		}
	}
	if clients >= 50 {
		return 500
	}
	return 1000
}

func appendSinkSummary(reportFile string, statsFile string) {
	statsContent, err := os.ReadFile(statsFile)
	if err != nil {
		file, createErr := os.OpenFile(reportFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		gomega.Expect(createErr).NotTo(gomega.HaveOccurred())
		defer file.Close()
		_, writeErr := fmt.Fprintln(file, "\n## Sink Summary\n- **Sink Stats:** unavailable")
		gomega.Expect(writeErr).NotTo(gomega.HaveOccurred())
		return
	}

	stats := parseSinkStats(string(statsContent))
	file, openErr := os.OpenFile(reportFile, os.O_APPEND|os.O_WRONLY, 0644)
	gomega.Expect(openErr).NotTo(gomega.HaveOccurred())
	defer file.Close()

	_, writeErr := fmt.Fprintf(file, `

## Sink Summary
| Parameter | Value |
| :--- | :--- |
| **Mode** | %s |
| **Received Messages** | %d |
| **Received Bytes** | %d |
| **Reply Messages** | %d |
| **Errors** | %d |
| **Warmup Messages** | %d |
| **Warmup Replies** | %d |
| **Drain Messages** | %d |
| **Drain Replies** | %d |
| **Elapsed Seconds** | %s |
| **Active Receive Seconds** | %s |
| **Receive Throughput** | %s msg/sec |
| **Receive Bandwidth** | %s MB/sec |
| **Active Receive Throughput** | %s msg/sec |
| **Active Receive Bandwidth** | %s MB/sec |
`,
		defaultIfEmpty(stats.Mode, "unknown"),
		stats.ReceivedMessages,
		stats.ReceivedBytes,
		stats.ReplyMessages,
		stats.Errors,
		stats.WarmupMessages,
		stats.WarmupReplies,
		stats.DrainMessages,
		stats.DrainReplies,
		formatFloat(stats.ElapsedSeconds),
		formatFloat(stats.ActiveReceiveSeconds),
		formatFloat(stats.ReceiveMPS),
		formatFloat(stats.ReceiveMBPS),
		formatFloat(stats.ActiveReceiveMPS),
		formatFloat(stats.ActiveReceiveMBPS),
	)
	gomega.Expect(writeErr).NotTo(gomega.HaveOccurred())
}

func collectProcessCPUUsage(rateSession *gexec.Session, runElapsed time.Duration, slimCPUStart float64, echoCPUStart float64) processCPUUsage {
	senderCPUSeconds := 0.0
	if rateSession != nil && rateSession.Command != nil && rateSession.Command.ProcessState != nil {
		senderCPUSeconds = rateSession.Command.ProcessState.UserTime().Seconds() + rateSession.Command.ProcessState.SystemTime().Seconds()
	}

	echoCPUEnd, err := readSessionCPUSeconds(echoSession)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	slimCPUEnd, err := readSessionCPUSeconds(slimSession)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	responderCPUSeconds := maxFloat(0, echoCPUEnd-echoCPUStart)
	nodeCPUSeconds := maxFloat(0, slimCPUEnd-slimCPUStart)
	elapsedSeconds := runElapsed.Seconds()
	if elapsedSeconds <= 0 {
		elapsedSeconds = 1e-9
	}

	totalCPUSeconds := senderCPUSeconds + responderCPUSeconds + nodeCPUSeconds
	return processCPUUsage{
		SenderCPUSeconds:    senderCPUSeconds,
		SenderCPUPercent:    100 * senderCPUSeconds / elapsedSeconds,
		ResponderCPUSeconds: responderCPUSeconds,
		ResponderCPUPercent: 100 * responderCPUSeconds / elapsedSeconds,
		NodeCPUSeconds:      nodeCPUSeconds,
		NodeCPUPercent:      100 * nodeCPUSeconds / elapsedSeconds,
		TotalCPUSeconds:     totalCPUSeconds,
		TotalCPUPercent:     100 * totalCPUSeconds / elapsedSeconds,
	}
}

func appendProcessCPUSummary(reportFile string, usage processCPUUsage) {
	file, err := os.OpenFile(reportFile, os.O_APPEND|os.O_WRONLY, 0644)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer file.Close()

	_, writeErr := fmt.Fprintf(file, `

## Process CPU Summary
| Process | CPU Time (sec) | Avg CPU %% |
| :--- | :--- | :--- |
| **Sender (tests/rate-client)** | %s | %s |
| **Responder (tests/echo-client)** | %s | %s |
| **Node (slimctl)** | %s | %s |
| **Total** | %s | %s |
`,
		formatFloat(usage.SenderCPUSeconds),
		formatFloat(usage.SenderCPUPercent),
		formatFloat(usage.ResponderCPUSeconds),
		formatFloat(usage.ResponderCPUPercent),
		formatFloat(usage.NodeCPUSeconds),
		formatFloat(usage.NodeCPUPercent),
		formatFloat(usage.TotalCPUSeconds),
		formatFloat(usage.TotalCPUPercent),
	)
	gomega.Expect(writeErr).NotTo(gomega.HaveOccurred())
}

func parseSenderReport(report string) senderReport {
	lines := strings.Split(report, "\n")
	return senderReport{
		TotalMessages:  mustParseInt(extractMarkdownValue(lines, "- **Total Messages:** ")),
		ThroughputMPS:  mustParseReportFloat(extractThroughputValue(lines)),
		RuntimeErrors:  mustParseInt(extractMarkdownValue(lines, "- **Runtime Errors:** ")),
		ActualDuration: extractMarkdownValue(lines, "- **Actual Duration:** "),
	}
}

func parseSinkStats(content string) sinkStats {
	values := parseKeyValueLines(content)
	return sinkStats{
		Mode:                 values["mode"],
		ReceivedMessages:     mustParseIntWithDefault(values["received_messages"], 0),
		ReceivedBytes:        mustParseIntWithDefault(values["received_bytes"], 0),
		ReplyMessages:        mustParseIntWithDefault(values["reply_messages"], 0),
		Errors:               mustParseIntWithDefault(values["errors"], 0),
		WarmupMessages:       mustParseIntWithDefault(values["warmup_messages"], 0),
		WarmupReplies:        mustParseIntWithDefault(values["warmup_replies"], 0),
		DrainMessages:        mustParseIntWithDefault(values["drain_messages"], 0),
		DrainReplies:         mustParseIntWithDefault(values["drain_replies"], 0),
		ElapsedSeconds:       mustParseFloatWithDefault(values["elapsed_seconds"], 0),
		ActiveReceiveSeconds: mustParseFloatWithDefault(values["active_receive_seconds"], 0),
		ReceiveMPS:           mustParseFloatWithDefault(values["receive_mps"], 0),
		ReceiveMBPS:          mustParseFloatWithDefault(values["receive_mbps"], 0),
		ActiveReceiveMPS:     mustParseFloatWithDefault(values["active_receive_mps"], 0),
		ActiveReceiveMBPS:    mustParseFloatWithDefault(values["active_receive_mbps"], 0),
	}
}

func parseKeyValueLines(content string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		values[parts[0]] = parts[1]
	}
	return values
}

func extractMarkdownValue(lines []string, prefix string) string {
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	ginkgo.Fail(fmt.Sprintf("missing report field %q", prefix))
	return ""
}

func extractThroughputValue(lines []string) string {
	value := extractMarkdownValue(lines, "- **Throughput:** ")
	re := regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?) msg/sec`)
	matches := re.FindStringSubmatch(value)
	gomega.Expect(matches).To(gomega.HaveLen(2), "expected throughput line to contain msg/sec value")
	return matches[1]
}

func envStringList(key string, defaults []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]string(nil), defaults...)
	}
	parts := strings.Fields(value)
	gomega.Expect(parts).NotTo(gomega.BeEmpty(), "invalid %s value", key)
	return parts
}

func envIntList(key string, defaults []int) []int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]int(nil), defaults...)
	}
	return mustParseIntList(value)
}

func envBool(key string, defaultValue bool) bool {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		default:
			ginkgo.Fail(fmt.Sprintf("invalid %s value %q", key, value))
		}
	}
	return defaultValue
}

func envFloat(key string, defaultValue float64) float64 {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid %s value %q", key, value)
		return parsed
	}
	return defaultValue
}

func mustParseIntList(value string) []int {
	fields := strings.Fields(value)
	gomega.Expect(fields).NotTo(gomega.BeEmpty())
	parsed := make([]int, 0, len(fields))
	for _, field := range fields {
		current, err := strconv.Atoi(field)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid integer list value %q", field)
		parsed = append(parsed, current)
	}
	return parsed
}

func mustParseInt(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid integer value %q", value)
	return parsed
}

func mustParseIntWithDefault(value string, defaultValue int64) int64 {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return mustParseInt(value)
}

func mustParseReportFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid float value %q", value)
	return parsed
}

func mustParseFloatWithDefault(value string, defaultValue float64) float64 {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return mustParseReportFloat(value)
}

func joinInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, " ")
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func formatCI(stats sampleStats) string {
	return fmt.Sprintf("[%s, %s]", formatFloat(stats.CILow), formatFloat(stats.CIHigh))
}

func defaultIfEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func maxFloat(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
