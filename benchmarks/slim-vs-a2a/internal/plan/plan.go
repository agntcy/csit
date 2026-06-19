// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ExecutionPlan struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
	Agents     []Agent  `yaml:"agents"`
	Tasks      []Task   `yaml:"tasks"`
	ContextUpdates []ContextUpdate `yaml:"contextUpdates"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Domain      string `yaml:"domain"`
	Description string `yaml:"description"`
}

type Spec struct {
	Defaults      TaskTiming `yaml:"defaults"`
	MaxRetries    int        `yaml:"maxRetries"`
	RoundBudgetMS int64      `yaml:"roundBudgetMs"`
	Sweep         *SweepSpec `yaml:"sweep,omitempty"`
}

// SweepSpec records parameters used to generate or classify sweep plans.
type SweepSpec struct {
	Family          string  `yaml:"family,omitempty"`
	Agents          int     `yaml:"agents,omitempty"`
	FanOut          int     `yaml:"fanOut,omitempty"`
	RoundBudgetMS   int64   `yaml:"roundBudgetMs,omitempty"`
	ExecuteSleepSec float64 `yaml:"executeSleepSec,omitempty"`
}

type Agent struct {
	ID       string `yaml:"id"`
	SlimName string `yaml:"slimName"`
	A2APort  int    `yaml:"a2aPort"`
}

type Task struct {
	ID                   string   `yaml:"id"`
	Name                 string   `yaml:"name"`
	Agent                string   `yaml:"agent"`
	DependsOn            []string `yaml:"dependsOn"`
	CompletionTimeSec    float64  `yaml:"completionTimeSec"`
	MaxCompletionTimeSec float64  `yaml:"maxCompletionTimeSec"`
	InjectFailure        bool     `yaml:"injectFailure"`
	Output               string   `yaml:"output"`
}

type TaskTiming struct {
	CompletionTimeSec    float64 `yaml:"completionTimeSec"`
	MaxCompletionTimeSec float64 `yaml:"maxCompletionTimeSec"`
}

type ContextUpdate struct {
	AfterTask    string   `yaml:"afterTask"`
	Payload      string   `yaml:"payload"`
	TargetAgents []string `yaml:"targetAgents"`
}

func LoadFile(path string) (*ExecutionPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p ExecutionPlan
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", filepath.Base(path), err)
	}
	p.applyDefaults()
	return &p, nil
}

// WriteFile marshals an execution plan to a YAML file.
func WriteFile(path string, p *ExecutionPlan) error {
	if err := p.Validate(); err != nil {
		return err
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadDir(dir string) ([]*ExecutionPlan, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var plans []*ExecutionPlan
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		p, err := LoadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	if len(plans) == 0 {
		return nil, fmt.Errorf("no plan yaml files in %s", dir)
	}
	return plans, nil
}

func (p *ExecutionPlan) applyDefaults() {
	for i := range p.Tasks {
		if p.Tasks[i].CompletionTimeSec == 0 {
			p.Tasks[i].CompletionTimeSec = p.Spec.Defaults.CompletionTimeSec
		}
		if p.Tasks[i].MaxCompletionTimeSec == 0 {
			p.Tasks[i].MaxCompletionTimeSec = p.Spec.Defaults.MaxCompletionTimeSec
		}
	}
}

func (p *ExecutionPlan) Validate() error {
	if p.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if len(p.Agents) == 0 {
		return fmt.Errorf("at least one agent is required")
	}
	if len(p.Tasks) == 0 {
		return fmt.Errorf("at least one task is required")
	}

	agentIDs := map[string]Agent{}
	for _, a := range p.Agents {
		if a.ID == "" {
			return fmt.Errorf("agent id is required")
		}
		if _, ok := agentIDs[a.ID]; ok {
			return fmt.Errorf("duplicate agent id %q", a.ID)
		}
		agentIDs[a.ID] = a
	}

	taskIDs := map[string]struct{}{}
	for _, t := range p.Tasks {
		if t.ID == "" {
			return fmt.Errorf("task id is required")
		}
		if _, ok := taskIDs[t.ID]; ok {
			return fmt.Errorf("duplicate task id %q", t.ID)
		}
		taskIDs[t.ID] = struct{}{}
	}

	for _, t := range p.Tasks {
		if _, ok := agentIDs[t.Agent]; !ok {
			return fmt.Errorf("task %q references unknown agent %q", t.ID, t.Agent)
		}
		for _, dep := range t.DependsOn {
			if _, ok := taskIDs[dep]; !ok {
				return fmt.Errorf("task %q depends on unknown task %q", t.ID, dep)
			}
		}
		if t.MaxCompletionTimeSec > 0 && t.MaxCompletionTimeSec < t.CompletionTimeSec {
			return fmt.Errorf("task %q maxCompletionTimeSec < completionTimeSec", t.ID)
		}
	}

	for _, cu := range p.ContextUpdates {
		if _, ok := taskIDs[cu.AfterTask]; !ok {
			return fmt.Errorf("contextUpdate references unknown task %q", cu.AfterTask)
		}
		for _, agentID := range cu.TargetAgents {
			if _, ok := agentIDs[agentID]; !ok {
				return fmt.Errorf("contextUpdate references unknown agent %q", agentID)
			}
		}
	}

	if err := detectCycle(p.Tasks); err != nil {
		return err
	}
	return nil
}

func (p *ExecutionPlan) AgentByID(id string) (Agent, bool) {
	for _, a := range p.Agents {
		if a.ID == id {
			return a, true
		}
	}
	return Agent{}, false
}

func (p *ExecutionPlan) TaskByID(id string) (Task, bool) {
	for _, t := range p.Tasks {
		if t.ID == id {
			return t, true
		}
	}
	return Task{}, false
}

func (p *ExecutionPlan) ContextUpdatesAfter(taskID string) []ContextUpdate {
	var out []ContextUpdate
	for _, cu := range p.ContextUpdates {
		if cu.AfterTask == taskID {
			out = append(out, cu)
		}
	}
	return out
}

func (p *ExecutionPlan) CardPort(agent Agent) int {
	return agent.A2APort - 1000
}

func (p *ExecutionPlan) CardBaseURL(agent Agent) string {
	return fmt.Sprintf("http://127.0.0.1:%d", p.CardPort(agent))
}

func detectCycle(tasks []Task) error {
	graph := map[string][]string{}
	for _, t := range tasks {
		graph[t.ID] = append([]string(nil), t.DependsOn...)
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var dfs func(string) error
	dfs = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("cycle detected at task %q", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, dep := range graph[id] {
			if err := dfs(dep); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range graph {
		if err := dfs(id); err != nil {
			return err
		}
	}
	return nil
}

func DownstreamTasks(tasks []Task, rootID string) map[string]struct{} {
	dependents := map[string][]string{}
	for _, t := range tasks {
		for _, dep := range t.DependsOn {
			dependents[dep] = append(dependents[dep], t.ID)
		}
	}
	out := map[string]struct{}{}
	queue := []string{rootID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, child := range dependents[cur] {
			if _, ok := out[child]; ok {
				continue
			}
			out[child] = struct{}{}
			queue = append(queue, child)
		}
	}
	return out
}
