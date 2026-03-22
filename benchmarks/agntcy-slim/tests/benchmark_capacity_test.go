package tests

import (
	"fmt"
	"math"
	"path/filepath"

	ginkgo "github.com/onsi/ginkgo/v2"
)

func runCapacitySweep(cfg suiteConfig) []capacitySweepCaseResult {
	results := make([]capacitySweepCaseResult, 0)
	for _, mode := range cfg.CapacitySweepModes {
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
	bestObserved := -1.0
	bestIndex := -1
	plateauCount := 0

	for step := 1; step <= cfg.CapacitySweepMaxSteps; step++ {
		current := runCapacitySweepStep(mode, clients, size, rate, step, cfg)
		if bestObserved > 0 {
			current.ObservedGainPercent = 100 * (current.ObservedMeanMPS - bestObserved) / bestObserved
		}
		if bestObserved <= 0 || current.ObservedMeanMPS > bestObserved*(1+cfg.CapacitySweepPlateauThreshold) {
			current.Improved = true
			bestObserved = current.ObservedMeanMPS
			bestIndex = len(caseResult.Steps)
			plateauCount = 0
		} else {
			plateauCount++
		}
		caseResult.Steps = append(caseResult.Steps, current)

		logCapacitySweepStep(mode, clients, size, current)

		if cfg.CapacitySweepMaxRate > 0 && rate >= cfg.CapacitySweepMaxRate {
			caseResult.StopReason = fmt.Sprintf("reached configured max rate %d", cfg.CapacitySweepMaxRate)
			if bestIndex >= 0 && caseResult.Steps[bestIndex].Rate != rate {
				caseResult.StopReason = fmt.Sprintf("reached configured max rate %d; effective capacity remained at best prior rate %d", cfg.CapacitySweepMaxRate, caseResult.Steps[bestIndex].Rate)
			}
			break
		}
		if plateauCount >= cfg.CapacitySweepPlateauSteps {
			if bestIndex >= 0 {
				caseResult.StopReason = fmt.Sprintf("effective throughput plateaued for %d consecutive steps; best prior rate remained %d", cfg.CapacitySweepPlateauSteps, caseResult.Steps[bestIndex].Rate)
			} else {
				caseResult.StopReason = fmt.Sprintf("effective throughput plateaued for %d consecutive steps", cfg.CapacitySweepPlateauSteps)
			}
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
		caseResult.BestObservedMeanMPS = best.ObservedMeanMPS
		caseResult.BestObservedCILow = best.ObservedCILow
		caseResult.BestObservedCIHigh = best.ObservedCIHigh
		caseResult.BestSenderMeanMPS = best.SenderMeanMPS
		caseResult.BestSenderCILow = best.SenderCILow
		caseResult.BestSenderCIHigh = best.SenderCIHigh
		caseResult.BestNodeCPUPercent = best.NodeMeanCPUPercent
		caseResult.BestNodeCILow = best.NodeCILow
		caseResult.BestNodeCIHigh = best.NodeCIHigh
		caseResult.BestTotalCPUPercent = best.TotalMeanCPUPercent
		caseResult.BestTotalCILow = best.TotalCILow
		caseResult.BestTotalCIHigh = best.TotalCIHigh
	}
	if caseResult.StopReason == "" {
		caseResult.StopReason = "completed sweep"
	}
	return caseResult
}

func runCapacitySweepStep(mode string, clients int, size int, rate int, step int, cfg suiteConfig) capacitySweepStepResult {
	responderMode := modeResponderKind(mode)

	stepRuns := make([]benchmarkRunResult, 0, cfg.CapacitySweepRepeats)
	for repeat := 1; repeat <= cfg.CapacitySweepRepeats; repeat++ {
		reportFile := filepath.Join(cfg.RawDir, fmt.Sprintf("sweep_%s_c%d_s%d_step%02d_r%d_rep%02d.md", mode, clients, size, step, rate, repeat))
		statsFile := filepath.Join(buildDir, fmt.Sprintf("sweep_stats_%s_c%d_s%d_step%02d_r%d_rep%02d.txt", mode, clients, size, step, rate, repeat))

		stopEchoResponder()
		if responderMode != "" {
			startEchoResponder(responderMode, clients, statsFile)
		}
		stepRuns = append(stepRuns, executeBenchmarkRun(mode, clients, size, rate, repeat, reportFile, statsFile, cfg))
		stopEchoResponder()
	}

	sender := computeSampleStats(senderMPSValues(stepRuns))
	observed := computeSampleStats(observedMPSValues(stepRuns))
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
		SenderVariance:      sender.Variance,
		SenderCILow:         sender.CILow,
		SenderCIHigh:        sender.CIHigh,
		ObservedMeanMPS:     observed.Mean,
		ObservedVariance:    observed.Variance,
		ObservedCILow:       observed.CILow,
		ObservedCIHigh:      observed.CIHigh,
		NodeMeanCPUPercent:  nodeCPU.Mean,
		NodeVariance:        nodeCPU.Variance,
		NodeCILow:           nodeCPU.CILow,
		NodeCIHigh:          nodeCPU.CIHigh,
		TotalMeanCPUPercent: totalCPU.Mean,
		TotalVariance:       totalCPU.Variance,
		TotalCILow:          totalCPU.CILow,
		TotalCIHigh:         totalCPU.CIHigh,
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
