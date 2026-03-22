package tests

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/stat/distuv"
)

func TestComputeSampleStatsEmpty(t *testing.T) {
	stats := computeSampleStats(nil)
	if stats.Count != 0 {
		t.Fatalf("count = %d, want 0", stats.Count)
	}
}

func TestComputeSampleStatsSingleValue(t *testing.T) {
	stats := computeSampleStats([]float64{42})
	if stats.Count != 1 || stats.Mean != 42 || stats.Variance != 0 || stats.StdDev != 0 {
		t.Fatalf("unexpected single-value stats: %+v", stats)
	}
	if stats.CILow != 42 || stats.CIHigh != 42 {
		t.Fatalf("unexpected single-value CI: [%f, %f]", stats.CILow, stats.CIHigh)
	}
}

func TestComputeSampleStatsSampleVariance(t *testing.T) {
	stats := computeSampleStats([]float64{1, 2, 3, 4})
	if stats.Count != 4 {
		t.Fatalf("count = %d, want 4", stats.Count)
	}
	if math.Abs(stats.Mean-2.5) > 1e-9 {
		t.Fatalf("mean = %f, want 2.5", stats.Mean)
	}
	if math.Abs(stats.Variance-1.6666666666666667) > 1e-9 {
		t.Fatalf("variance = %f, want sample variance 1.6666666666666667", stats.Variance)
	}
	if math.Abs(stats.StdDev-math.Sqrt(1.6666666666666667)) > 1e-9 {
		t.Fatalf("stddev = %f, want %f", stats.StdDev, math.Sqrt(1.6666666666666667))
	}
	t975 := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: 3}.Quantile(1 - confidenceIntervalAlpha/2)
	expectedMargin := t975 * math.Sqrt(1.6666666666666667) / math.Sqrt(4)
	if math.Abs(stats.CILow-(2.5-expectedMargin)) > 1e-9 {
		t.Fatalf("ci low = %f, want %f", stats.CILow, 2.5-expectedMargin)
	}
	if math.Abs(stats.CIHigh-(2.5+expectedMargin)) > 1e-9 {
		t.Fatalf("ci high = %f, want %f", stats.CIHigh, 2.5+expectedMargin)
	}
	if stats.CILow > stats.Mean || stats.CIHigh < stats.Mean {
		t.Fatalf("mean should be inside CI: %+v", stats)
	}
}
