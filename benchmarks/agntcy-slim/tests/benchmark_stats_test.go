package tests

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

type sampleStats struct {
	Count    int
	Mean     float64
	Variance float64
	StdDev   float64
	CILow    float64
	CIHigh   float64
}

func senderMPSValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.SenderMPS)
	}
	return values
}

func observedMPSValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, benchmarkObservedMPS(row))
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

func observedMessageValues(rows []benchmarkRunResult) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, benchmarkObservedMessages(row))
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
	if count == 0 {
		return sampleStats{}
	}

	mean := stat.Mean(values, nil)
	variance := 0.0
	if count > 1 {
		variance = stat.Variance(values, nil)
	}
	stddev := math.Sqrt(variance)
	margin := 0.0
	if count > 1 {
		z975 := distuv.UnitNormal.Quantile(0.975)
		margin = z975 * stddev / math.Sqrt(float64(count))
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

func formatCI(stats sampleStats) string {
	return fmt.Sprintf("[%s, %s]", formatFloat(stats.CILow), formatFloat(stats.CIHigh))
}
