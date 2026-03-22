package tests

import "testing"

func TestCapacitySweepImproved(t *testing.T) {
	best := capacitySweepStepResult{ObservedMeanMPS: 100}
	if !capacitySweepImproved(best, capacitySweepStepResult{ObservedMeanMPS: 106}, 0.05) {
		t.Fatal("expected 6% observed throughput gain to count as improvement")
	}
	if capacitySweepImproved(best, capacitySweepStepResult{ObservedMeanMPS: 104}, 0.05) {
		t.Fatal("expected 4% observed throughput gain to remain within plateau threshold")
	}
}

func TestCapacitySweepRegressed(t *testing.T) {
	best := capacitySweepStepResult{ObservedMeanMPS: 100}
	if !capacitySweepRegressed(best, capacitySweepStepResult{ObservedMeanMPS: 94}, 0.05) {
		t.Fatal("expected throughput below the regression threshold to count as regression")
	}
	if capacitySweepRegressed(best, capacitySweepStepResult{ObservedMeanMPS: 97}, 0.05) {
		t.Fatal("expected throughput within the plateau band not to count as regression")
	}
}

func TestMidpointRate(t *testing.T) {
	if got := midpointRate(4000, 8000); got != 6000 {
		t.Fatalf("midpointRate(4000, 8000) = %d, want 6000", got)
	}
}
