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
	bestIndex := -1
	plateauCount := 0
	capacityLower := 0
	capacityUpper := 0
	regressionRate := 0

	for step := 1; step <= cfg.CapacitySweepMaxSteps; step++ {
		current := runCapacitySweepStep(mode, clients, size, rate, step, "coarse", cfg)
		if bestIndex >= 0 {
			best := caseResult.Steps[bestIndex]
			if best.ObservedMeanMPS > 0 {
				current.ObservedGainPercent = 100 * (current.ObservedMeanMPS - best.ObservedMeanMPS) / best.ObservedMeanMPS
			}
		}
		if bestIndex < 0 || capacitySweepImproved(caseResult.Steps[bestIndex], current, cfg.CapacitySweepPlateauThreshold) {
			current.Improved = true
			capacityLower = current.Rate
			plateauCount = 0
		} else {
			capacityUpper = current.Rate
			plateauCount++
			if capacitySweepRegressed(caseResult.Steps[bestIndex], current, cfg.CapacitySweepPlateauThreshold) {
				regressionRate = current.Rate
			}
		}
		caseResult.Steps = append(caseResult.Steps, current)
		if current.Improved {
			bestIndex = len(caseResult.Steps) - 1
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
			caseResult.StopReason = buildCapacityStopReason(bestIndex, caseResult.Steps, capacityLower, capacityUpper, regressionRate, cfg.CapacitySweepPlateauSteps)
			break
		}
		if step == cfg.CapacitySweepMaxSteps {
			caseResult.StopReason = fmt.Sprintf("reached configured max steps %d", cfg.CapacitySweepMaxSteps)
			break
		}

		rate = nextSweepRate(rate, cfg.CapacitySweepGrowthFactor, cfg.CapacitySweepMaxRate)
	}

	if bestIndex >= 0 {
		if capacityLower == 0 {
			capacityLower = caseResult.Steps[bestIndex].Rate
		}
		if capacityUpper > capacityLower && cfg.CapacitySweepRefinementSteps > 0 {
			bestIndex, capacityLower, capacityUpper = refineCapacitySweepCase(mode, clients, size, bestIndex, capacityLower, capacityUpper, &caseResult, cfg)
			caseResult.StopReason = fmt.Sprintf("refinement narrowed the estimated capacity to offered rates %d through %d", capacityLower, capacityUpper)
		}
	}

	if bestIndex >= 0 {
		best := caseResult.Steps[bestIndex]
		caseResult.BestRate = best.Rate
		caseResult.CapacityRateLower = maxInt(capacityLower, best.Rate)
		if capacityUpper > 0 {
			caseResult.CapacityRateUpper = capacityUpper
		} else {
			caseResult.CapacityRateUpper = caseResult.CapacityRateLower
		}
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
		if caseResult.CapacityRateUpper > caseResult.CapacityRateLower {
			caseResult.StopReason = fmt.Sprintf("estimated capacity bracketed between offered rates %d and %d", caseResult.CapacityRateLower, caseResult.CapacityRateUpper)
		} else {
			caseResult.StopReason = "completed sweep"
		}
	}
	return caseResult
}

func runCapacitySweepStep(mode string, clients int, size int, rate int, step int, phase string, cfg suiteConfig) capacitySweepStepResult {
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
		Phase:               phase,
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

func refineCapacitySweepCase(mode string, clients int, size int, bestIndex int, lower int, upper int, caseResult *capacitySweepCaseResult, cfg suiteConfig) (int, int, int) {
	for refinement := 0; refinement < cfg.CapacitySweepRefinementSteps; refinement++ {
		if upper-lower <= cfg.CapacitySweepMinRateDelta {
			break
		}
		rate := midpointRate(lower, upper)
		if rate <= lower || rate >= upper {
			break
		}
		stepNumber := len(caseResult.Steps) + 1
		current := runCapacitySweepStep(mode, clients, size, rate, stepNumber, "refine", cfg)
		best := caseResult.Steps[bestIndex]
		if best.ObservedMeanMPS > 0 {
			current.ObservedGainPercent = 100 * (current.ObservedMeanMPS - best.ObservedMeanMPS) / best.ObservedMeanMPS
		}
		if capacitySweepImproved(best, current, cfg.CapacitySweepPlateauThreshold) {
			current.Improved = true
			lower = current.Rate
			caseResult.Steps = append(caseResult.Steps, current)
			bestIndex = len(caseResult.Steps) - 1
		} else {
			upper = current.Rate
			caseResult.Steps = append(caseResult.Steps, current)
		}
		logCapacitySweepStep(mode, clients, size, current)
	}
	return bestIndex, lower, upper
}

func capacitySweepImproved(best capacitySweepStepResult, current capacitySweepStepResult, threshold float64) bool {
	if best.ObservedMeanMPS <= 0 {
		return current.ObservedMeanMPS > 0
	}
	return current.ObservedMeanMPS > best.ObservedMeanMPS*(1+threshold)
}

func capacitySweepRegressed(best capacitySweepStepResult, current capacitySweepStepResult, threshold float64) bool {
	if best.ObservedMeanMPS <= 0 {
		return false
	}
	return current.ObservedMeanMPS < best.ObservedMeanMPS*(1-threshold)
}

func midpointRate(lower int, upper int) int {
	return lower + (upper-lower)/2
}

func buildCapacityStopReason(bestIndex int, steps []capacitySweepStepResult, lower int, upper int, regressionRate int, plateauSteps int) string {
	if lower > 0 && upper > lower {
		if regressionRate > 0 && bestIndex >= 0 {
			return fmt.Sprintf("effective throughput regressed at rate %d; capacity is bracketed between offered rates %d and %d with best prior rate %d", regressionRate, lower, upper, steps[bestIndex].Rate)
		}
		return fmt.Sprintf("effective throughput plateaued for %d consecutive steps; capacity is bracketed between offered rates %d and %d", plateauSteps, lower, upper)
	}
	if bestIndex >= 0 {
		return fmt.Sprintf("effective throughput plateaued for %d consecutive steps; best prior rate remained %d", plateauSteps, steps[bestIndex].Rate)
	}
	return fmt.Sprintf("effective throughput plateaued for %d consecutive steps", plateauSteps)
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
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
