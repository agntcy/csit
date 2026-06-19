// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package plan

import (
	"fmt"
)

// CanonicalFamily identifies paper-aligned micro-plan templates.
type CanonicalFamily string

const (
	FamilyPreferenceAggregation CanonicalFamily = "preference-aggregation"
	FamilySupplyChainCascade    CanonicalFamily = "supply-chain-cascade"
	FamilySustainableResource   CanonicalFamily = "sustainable-resource"
)

// GenerateOptions configures generated execution plans for sweeps.
type GenerateOptions struct {
	Family          CanonicalFamily
	Agents          int
	FanOut          int
	RoundBudgetMS   int64
	ExecuteSleepSec float64
	Rounds          int
}

// GenerateCanonical builds an ExecutionPlan for transport/coordination sweeps.
func GenerateCanonical(opts GenerateOptions) (*ExecutionPlan, error) {
	if opts.Agents < 2 {
		return nil, fmt.Errorf("agents must be >= 2")
	}
	if opts.ExecuteSleepSec <= 0 {
		opts.ExecuteSleepSec = 0.01
	}
	if opts.Rounds <= 0 {
		opts.Rounds = 1
	}

	switch opts.Family {
	case FamilyPreferenceAggregation:
		return generatePreferenceAggregation(opts)
	case FamilySupplyChainCascade:
		return generateSupplyChainCascade(opts)
	case FamilySustainableResource:
		return generateSustainableResource(opts)
	default:
		return nil, fmt.Errorf("unknown family %q", opts.Family)
	}
}

func generatePreferenceAggregation(opts GenerateOptions) (*ExecutionPlan, error) {
	n := opts.Agents
	p := basePlan(string(FamilyPreferenceAggregation), "canonical-preference", opts)
	agents := make([]Agent, n)
	for i := 0; i < n; i++ {
		agents[i] = agentAt(i, 9400+i*10)
	}
	p.Agents = agents

	var tasks []Task
	for i := 0; i < n; i++ {
		deps := []string{}
		if i > 0 {
			deps = []string{fmt.Sprintf("propose-%d", i-1)}
		}
		tasks = append(tasks, Task{
			ID:                   fmt.Sprintf("propose-%d", i),
			Name:                 fmt.Sprintf("Agent %d local proposal", i),
			Agent:                agents[i].ID,
			DependsOn:            deps,
			CompletionTimeSec:    opts.ExecuteSleepSec,
			MaxCompletionTimeSec: opts.ExecuteSleepSec * 4,
			Output:               fmt.Sprintf("proposal=score_%d", i*2),
		})
	}
	p.Tasks = tasks

	lastTask := tasks[len(tasks)-1].ID
	targets := agentIDs(agents)
	if opts.FanOut > 0 && opts.FanOut < len(targets) {
		targets = targets[:opts.FanOut]
	}
	p.ContextUpdates = []ContextUpdate{{
		AfterTask:    lastTask,
		Payload:      "sync=phase=round-finalize proposals=aggregated",
		TargetAgents: targets,
	}}
	return p, p.Validate()
}

func generateSupplyChainCascade(opts GenerateOptions) (*ExecutionPlan, error) {
	n := opts.Agents
	p := basePlan(string(FamilySupplyChainCascade), "canonical-supply-chain", opts)
	agents := make([]Agent, n)
	for i := 0; i < n; i++ {
		agents[i] = agentAt(i, 9500+i*10)
	}
	p.Agents = agents

	var tasks []Task
	for i := 0; i < n; i++ {
		deps := []string{}
		if i > 0 {
			deps = []string{fmt.Sprintf("stage-%d", i-1)}
		}
		tasks = append(tasks, Task{
			ID:                   fmt.Sprintf("stage-%d", i),
			Name:                 fmt.Sprintf("Supply chain stage %d order", i),
			Agent:                agents[i].ID,
			DependsOn:            deps,
			CompletionTimeSec:    opts.ExecuteSleepSec,
			MaxCompletionTimeSec: opts.ExecuteSleepSec * 4,
			Output:               fmt.Sprintf("order=units_%d", 4+i),
		})
	}
	p.Tasks = tasks

	// Per-hop context to downstream neighbor(s).
	for i := 0; i < n-1; i++ {
		targets := []string{agents[i+1].ID}
		p.ContextUpdates = append(p.ContextUpdates, ContextUpdate{
			AfterTask:    fmt.Sprintf("stage-%d", i),
			Payload:      fmt.Sprintf("epoch=%d stock=shared", i+1),
			TargetAgents: targets,
		})
	}
	return p, p.Validate()
}

func generateSustainableResource(opts GenerateOptions) (*ExecutionPlan, error) {
	n := opts.Agents
	p := basePlan(string(FamilySustainableResource), "canonical-sustainable-resource", opts)
	agents := make([]Agent, n)
	for i := 0; i < n; i++ {
		agents[i] = agentAt(i, 9600+i*10)
	}
	p.Agents = agents

	allTargets := agentIDs(agents)
	fanOut := len(allTargets)
	if opts.FanOut > 0 && opts.FanOut < fanOut {
		fanOut = opts.FanOut
	}
	targets := allTargets[:fanOut]

	var tasks []Task
	for r := 0; r < opts.Rounds; r++ {
		for i := 0; i < n; i++ {
			id := fmt.Sprintf("extract-r%d-a%d", r, i)
			deps := []string{}
			if r > 0 {
				deps = append(deps, fmt.Sprintf("extract-r%d-a%d", r-1, n-1))
			} else if i > 0 {
				deps = append(deps, fmt.Sprintf("extract-r%d-a%d", r, i-1))
			}
			tasks = append(tasks, Task{
				ID:                   id,
				Name:                 fmt.Sprintf("Round %d agent %d extraction", r, i),
				Agent:                agents[i].ID,
				DependsOn:            deps,
				CompletionTimeSec:    opts.ExecuteSleepSec,
				MaxCompletionTimeSec: opts.ExecuteSleepSec * 4,
				Output:               fmt.Sprintf("extract=2.0 round=%d", r),
			})
		}
		// Shared stock broadcast after each round's last agent completes.
		p.ContextUpdates = append(p.ContextUpdates, ContextUpdate{
			AfterTask:    fmt.Sprintf("extract-r%d-a%d", r, n-1),
			Payload:      fmt.Sprintf("round=%d stock=shared shadow_price=0.35", r),
			TargetAgents: append([]string(nil), targets...),
		})
	}
	p.Tasks = tasks
	return p, p.Validate()
}

func basePlan(domain, name string, opts GenerateOptions) *ExecutionPlan {
	return &ExecutionPlan{
		APIVersion: "bench.agntcy.io/v1",
		Kind:       "ExecutionPlan",
		Metadata: Metadata{
			Name:        fmt.Sprintf("%s-%dagents", name, opts.Agents),
			Domain:      domain,
			Description: "Generated canonical plan for transport sweeps",
		},
		Spec: Spec{
			Defaults: TaskTiming{
				CompletionTimeSec:    opts.ExecuteSleepSec,
				MaxCompletionTimeSec: opts.ExecuteSleepSec * 4,
			},
			RoundBudgetMS: opts.RoundBudgetMS,
			Sweep: &SweepSpec{
				Family:          string(opts.Family),
				Agents:          opts.Agents,
				FanOut:          opts.FanOut,
				RoundBudgetMS:   opts.RoundBudgetMS,
				ExecuteSleepSec: opts.ExecuteSleepSec,
			},
		},
	}
}

func agentAt(index, portBase int) Agent {
	id := fmt.Sprintf("agent-%d", index)
	return Agent{
		ID:       id,
		SlimName: fmt.Sprintf("agntcy/bench/%s", id),
		A2APort:  portBase + index,
	}
}

func agentIDs(agents []Agent) []string {
	ids := make([]string, len(agents))
	for i, a := range agents {
		ids[i] = a.ID
	}
	return ids
}

// EffectiveRoundBudgetMS returns runner override, plan spec, or zero.
func (p *ExecutionPlan) EffectiveRoundBudgetMS(override int64) int64 {
	if override > 0 {
		return override
	}
	if p.Spec.RoundBudgetMS > 0 {
		return p.Spec.RoundBudgetMS
	}
	return 0
}
