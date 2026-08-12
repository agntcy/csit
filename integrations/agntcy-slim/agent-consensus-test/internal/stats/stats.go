// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

const ConfidenceIntervalAlpha = 0.05

type SampleStats struct {
	Count    int
	Mean     float64
	Variance float64
	StdDev   float64
	CILow    float64
	CIHigh   float64
}

func ComputeSampleStats(values []float64) SampleStats {
	count := len(values)
	if count == 0 {
		return SampleStats{}
	}

	mean := stat.Mean(values, nil)
	if count == 1 {
		return SampleStats{
			Count:    1,
			Mean:     mean,
			Variance: 0,
			StdDev:   0,
			CILow:    mean,
			CIHigh:   mean,
		}
	}

	stddev := stat.StdDev(values, nil)
	variance := stat.Variance(values, nil)
	standardError := stddev / math.Sqrt(float64(count))
	tDist := distuv.StudentsT{Mu: mean, Sigma: standardError, Nu: float64(count - 1)}
	tailProbability := ConfidenceIntervalAlpha / 2
	ciLow := tDist.Quantile(tailProbability)
	ciHigh := tDist.Quantile(1 - tailProbability)

	return SampleStats{
		Count:    count,
		Mean:     mean,
		Variance: variance,
		StdDev:   stddev,
		CILow:    math.Max(0, ciLow),
		CIHigh:   ciHigh,
	}
}

func FormatCI(s SampleStats) string {
	return fmt.Sprintf("[%s, %s]", FormatFloat(s.CILow), FormatFloat(s.CIHigh))
}

func FormatFloat(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "—"
	}
	abs := math.Abs(v)
	switch {
	case abs >= 1000:
		return fmt.Sprintf("%.0f", v)
	case abs >= 100:
		return fmt.Sprintf("%.1f", v)
	case abs >= 10:
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%.3f", v)
	}
}
