// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package plan_test

import (
	"testing"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/plan"
)

func TestGenerateCanonicalFamilies(t *testing.T) {
	families := []plan.CanonicalFamily{
		plan.FamilyPreferenceAggregation,
		plan.FamilySupplyChainCascade,
		plan.FamilySustainableResource,
	}
	for _, family := range families {
		p, err := plan.GenerateCanonical(plan.GenerateOptions{
			Family:          family,
			Agents:          5,
			RoundBudgetMS:   20,
			ExecuteSleepSec: 0.01,
			Rounds:          1,
		})
		if err != nil {
			t.Fatalf("family %s: %v", family, err)
		}
		if len(p.Agents) != 5 {
			t.Fatalf("family %s: expected 5 agents, got %d", family, len(p.Agents))
		}
		if p.Spec.RoundBudgetMS != 20 {
			t.Fatalf("family %s: round budget not set", family)
		}
	}
}

func TestEffectiveRoundBudgetMS(t *testing.T) {
	p := &plan.ExecutionPlan{Spec: plan.Spec{RoundBudgetMS: 15}}
	if got := p.EffectiveRoundBudgetMS(0); got != 15 {
		t.Fatalf("expected 15, got %d", got)
	}
	if got := p.EffectiveRoundBudgetMS(10); got != 10 {
		t.Fatalf("override expected 10, got %d", got)
	}
}
