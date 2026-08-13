// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/stat/distuv"
)

func TestComputeSampleStatsEmpty(t *testing.T) {
	got := ComputeSampleStats(nil)
	if got.Count != 0 {
		t.Fatalf("count = %d, want 0", got.Count)
	}
}

func TestComputeSampleStatsSingleValue(t *testing.T) {
	got := ComputeSampleStats([]float64{42})
	if got.Count != 1 || got.Mean != 42 || got.Variance != 0 || got.StdDev != 0 {
		t.Fatalf("unexpected single-value stats: %+v", got)
	}
	if got.CILow != 42 || got.CIHigh != 42 {
		t.Fatalf("unexpected single-value CI: [%f, %f]", got.CILow, got.CIHigh)
	}
}

func TestComputeSampleStatsSampleVariance(t *testing.T) {
	got := ComputeSampleStats([]float64{1, 2, 3, 4})
	if got.Count != 4 {
		t.Fatalf("count = %d, want 4", got.Count)
	}
	if math.Abs(got.Mean-2.5) > 1e-9 {
		t.Fatalf("mean = %f, want 2.5", got.Mean)
	}
	if math.Abs(got.Variance-1.6666666666666667) > 1e-9 {
		t.Fatalf("variance = %f, want sample variance 1.6666666666666667", got.Variance)
	}
	if math.Abs(got.StdDev-math.Sqrt(1.6666666666666667)) > 1e-9 {
		t.Fatalf("stddev = %f, want %f", got.StdDev, math.Sqrt(1.6666666666666667))
	}
	t975 := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: 3}.Quantile(1 - ConfidenceIntervalAlpha/2)
	expectedMargin := t975 * math.Sqrt(1.6666666666666667) / math.Sqrt(4)
	if math.Abs(got.CILow-(2.5-expectedMargin)) > 1e-9 {
		t.Fatalf("ci low = %f, want %f", got.CILow, 2.5-expectedMargin)
	}
	if math.Abs(got.CIHigh-(2.5+expectedMargin)) > 1e-9 {
		t.Fatalf("ci high = %f, want %f", got.CIHigh, 2.5+expectedMargin)
	}
	if got.CILow > got.Mean || got.CIHigh < got.Mean {
		t.Fatalf("mean should be inside CI: %+v", got)
	}
}
