// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scenario

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	TargetModeMajority = "majority"
)

type ConsensusScenario struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
	Agents     []Agent  `yaml:"agents"`
}

type Metadata struct {
	Name        string `yaml:"name"`
	Domain      string `yaml:"domain"`
	Description string `yaml:"description"`
}

type Spec struct {
	Agents             int    `yaml:"agents"`
	ThinkTimeMs        int64  `yaml:"thinkTimeMs"`
	FindingEmitDelayMs int64  `yaml:"findingEmitDelayMs"`
	MaxRounds          int    `yaml:"maxRounds"`
	TargetMode         string `yaml:"targetMode"`
	Seed               int64  `yaml:"seed"`
	ValueSpace         int    `yaml:"valueSpace"`
}

type Agent struct {
	ID       string `yaml:"id"`
	SlimName string `yaml:"slimName"`
	A2APort  int    `yaml:"a2aPort"`
	CardPort int    `yaml:"cardPort,omitempty"`
	Role     string `yaml:"role,omitempty"`
}

func LoadFile(path string) (*ConsensusScenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s ConsensusScenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *ConsensusScenario) Validate() error {
	if s.APIVersion != "bench.agntcy.io/v2" {
		return fmt.Errorf("unsupported apiVersion %q", s.APIVersion)
	}
	if s.Kind != "ConsensusScenario" {
		return fmt.Errorf("unsupported kind %q", s.Kind)
	}
	if s.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if s.Spec.Agents < 2 {
		return fmt.Errorf("spec.agents must be >= 2")
	}
	if len(s.Agents) != s.Spec.Agents {
		return fmt.Errorf("agents list length %d != spec.agents %d", len(s.Agents), s.Spec.Agents)
	}
	if s.Spec.ThinkTimeMs <= 0 {
		return fmt.Errorf("spec.thinkTimeMs must be > 0")
	}
	if s.Spec.MaxRounds <= 0 {
		return fmt.Errorf("spec.maxRounds must be > 0")
	}
	if s.Spec.TargetMode == "" {
		s.Spec.TargetMode = TargetModeMajority
	}
	if s.Spec.ValueSpace <= 0 {
		s.Spec.ValueSpace = 3
	}
	if s.Spec.FindingEmitDelayMs <= 0 {
		s.Spec.FindingEmitDelayMs = 1
	}
	for i, a := range s.Agents {
		if a.ID == "" {
			return fmt.Errorf("agents[%d].id is required", i)
		}
		if a.SlimName == "" {
			return fmt.Errorf("agents[%d].slimName is required", i)
		}
		if a.A2APort <= 0 {
			return fmt.Errorf("agents[%d].a2aPort is required", i)
		}
	}
	return nil
}

func (s *ConsensusScenario) CardPort(agent Agent) int {
	if agent.CardPort > 0 {
		return agent.CardPort
	}
	return agent.A2APort + 1000
}

func (s *ConsensusScenario) CardBaseURL(agent Agent) string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.CardPort(agent))
}

func (s *ConsensusScenario) AgentByID(id string) (Agent, bool) {
	for _, a := range s.Agents {
		if a.ID == id {
			return a, true
		}
	}
	return Agent{}, false
}

func (s *ConsensusScenario) Coordinator() Agent {
	for _, a := range s.Agents {
		if a.Role == "coordinator" || a.ID == "agent-0" {
			return a
		}
	}
	return s.Agents[0]
}

func (s *ConsensusScenario) WorkerAgents() []Agent {
	coord := s.Coordinator()
	var out []Agent
	for _, a := range s.Agents {
		if a.ID != coord.ID {
			out = append(out, a)
		}
	}
	return out
}

func (s *ConsensusScenario) AgentIDs() []string {
	ids := make([]string, len(s.Agents))
	for i, a := range s.Agents {
		ids[i] = a.ID
	}
	return ids
}

func (s *ConsensusScenario) SlimNames() []string {
	names := make([]string, len(s.Agents))
	for i, a := range s.Agents {
		names[i] = a.SlimName
	}
	return names
}

type GenerateOptions struct {
	Family      string
	Agents      int
	ThinkTimeMs int64
	Seed        int64
}

func Generate(opts GenerateOptions) (*ConsensusScenario, error) {
	if opts.Agents < 2 {
		return nil, fmt.Errorf("agents must be >= 2")
	}
	if opts.ThinkTimeMs <= 0 {
		opts.ThinkTimeMs = 10
	}
	if opts.Family == "" {
		opts.Family = "hypothesis-convergence"
	}
	if opts.Seed == 0 {
		opts.Seed = 42
	}

	name := fmt.Sprintf("%s-%dagents-%dms", opts.Family, opts.Agents, opts.ThinkTimeMs)
	agents := make([]Agent, opts.Agents)
	for i := 0; i < opts.Agents; i++ {
		id := fmt.Sprintf("agent-%d", i)
		role := "worker"
		if i == 0 {
			role = "coordinator"
		}
		agents[i] = Agent{
			ID:       id,
			SlimName: fmt.Sprintf("agntcy/bench-v2/%s", id),
			A2APort:  9700 + i*11,
			Role:     role,
		}
	}

	s := &ConsensusScenario{
		APIVersion: "bench.agntcy.io/v2",
		Kind:       "ConsensusScenario",
		Metadata: Metadata{
			Name:        name,
			Domain:      opts.Family,
			Description: "Generated consensus scenario for transport sweeps",
		},
		Spec: Spec{
			Agents:             opts.Agents,
			ThinkTimeMs:        opts.ThinkTimeMs,
			FindingEmitDelayMs: maxInt64(1, opts.ThinkTimeMs/10),
			MaxRounds:          200,
			TargetMode:         TargetModeMajority,
			Seed:               opts.Seed,
			ValueSpace:         3,
		},
		Agents: agents,
	}
	return s, s.Validate()
}

func Marshal(s *ConsensusScenario) ([]byte, error) {
	return yaml.Marshal(s)
}

func WriteFile(path string, s *ConsensusScenario) error {
	data, err := Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func NormalizeFamily(input string) string {
	return strings.TrimSpace(input)
}
