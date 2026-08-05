// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scenario

import (
	"fmt"
	"math"
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
	// Order is an optional display/run rank used to sort scenarios in the
	// dashboard and driver (lower runs/renders first). Zero means unordered.
	Order int `yaml:"order,omitempty"`
}

type Spec struct {
	Agents             int    `yaml:"agents"`
	ThinkTimeMs        int64  `yaml:"thinkTimeMs"`
	FindingEmitDelayMs int64  `yaml:"findingEmitDelayMs"`
	MaxRounds          int    `yaml:"maxRounds"`
	TargetMode         string `yaml:"targetMode"`
	Seed               int64  `yaml:"seed"`
	ValueSpace         int    `yaml:"valueSpace"`
	// PayloadBytes inflates every finding with fixed-size padding to stress
	// transport bandwidth. 0 (default) keeps findings at their minimal size.
	PayloadBytes int `yaml:"payloadBytes,omitempty"`
	// Epochs is how many times the same consensus attempt is repeated. Each
	// epoch resets the agents and re-runs; the runner counts successful vs
	// failed epochs. Defaults to 10.
	Epochs int `yaml:"epochs,omitempty"`
	// MaxEpochTimeMs is the per-epoch wall-clock budget: if global consensus is
	// not reached within it, the epoch is counted as failed. Defaults to 120000.
	MaxEpochTimeMs int64 `yaml:"maxEpochTimeMs,omitempty"`
}

type Agent struct {
	ID       string `yaml:"id"`
	SlimName string `yaml:"slimName"`
	P2PPort  int    `yaml:"p2pPort"`
	CardPort int    `yaml:"cardPort,omitempty"`
	Role     string `yaml:"role,omitempty"`
	// LatencyMs adds a one-way network delay to this agent's finding sends and
	// receives, simulating an agent in a distant region. 0 (default) = local.
	LatencyMs int64 `yaml:"latencyMs,omitempty"`
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
	if s.Spec.PayloadBytes < 0 {
		s.Spec.PayloadBytes = 0
	}
	if s.Spec.Epochs <= 0 {
		s.Spec.Epochs = 10
	}
	if s.Spec.MaxEpochTimeMs <= 0 {
		s.Spec.MaxEpochTimeMs = 120000
	}
	for i, a := range s.Agents {
		if a.ID == "" {
			return fmt.Errorf("agents[%d].id is required", i)
		}
		if a.SlimName == "" {
			return fmt.Errorf("agents[%d].slimName is required", i)
		}
		if a.P2PPort <= 0 {
			return fmt.Errorf("agents[%d].p2pPort is required", i)
		}
		if a.LatencyMs < 0 {
			s.Agents[i].LatencyMs = 0
		}
	}
	return nil
}

func (s *ConsensusScenario) CardPort(agent Agent) int {
	if agent.CardPort > 0 {
		return agent.CardPort
	}
	return agent.P2PPort + 1000
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
	Family         string
	Agents         int
	ThinkTimeMs    int64
	Seed           int64
	PayloadBytes   int
	Epochs         int
	MaxEpochTimeMs int64
	// LatencyMs, when > 0, is stamped on the last LatencyCount agents to
	// simulate a subset of agents living in a distant region.
	LatencyMs int64
	// LatencyCount is how many (trailing) agents get LatencyMs. When 0 and
	// LatencyMs > 0, it defaults to round(Agents/3).
	LatencyCount int
	// MatrixNaming selects compact matrix filenames: matrix-lat{L}-{N}ag[-{B}b].
	MatrixNaming bool
}

// MatrixScenarioFilename returns the canonical filename stem for an agents×payload
// matrix cell at a fixed latency slice (think time is fixed globally, not encoded).
func MatrixScenarioFilename(latencyMs int64, agents, payloadBytes int) string {
	name := fmt.Sprintf("matrix-lat%d-%dag", latencyMs, agents)
	if payloadBytes > 0 {
		name = fmt.Sprintf("%s-%db", name, payloadBytes)
	}
	return name
}

// RepresentativeLatencyMs returns the configured one-way delay stamped on laggy
// agents (0 when every agent is local).
func (s *ConsensusScenario) RepresentativeLatencyMs() int64 {
	var max int64
	for _, a := range s.Agents {
		if a.LatencyMs > max {
			max = a.LatencyMs
		}
	}
	return max
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
	if opts.PayloadBytes < 0 {
		opts.PayloadBytes = 0
	}
	if opts.Epochs <= 0 {
		opts.Epochs = 10
	}
	if opts.MaxEpochTimeMs <= 0 {
		opts.MaxEpochTimeMs = 120000
	}

	if opts.LatencyMs < 0 {
		opts.LatencyMs = 0
	}
	latencyCount := 0
	if opts.LatencyMs > 0 {
		latencyCount = opts.LatencyCount
		if latencyCount <= 0 {
			latencyCount = int(math.Round(float64(opts.Agents) / 3.0))
		}
		if latencyCount < 1 {
			latencyCount = 1
		}
		if latencyCount > opts.Agents {
			latencyCount = opts.Agents
		}
	}

	var name string
	if opts.MatrixNaming {
		name = MatrixScenarioFilename(opts.LatencyMs, opts.Agents, opts.PayloadBytes)
	} else {
		name = fmt.Sprintf("%s-%dagents-%dms", opts.Family, opts.Agents, opts.ThinkTimeMs)
		if opts.PayloadBytes > 0 {
			name = fmt.Sprintf("%s-%db", name, opts.PayloadBytes)
		}
		if latencyCount > 0 {
			name = fmt.Sprintf("%s-lat%d", name, opts.LatencyMs)
		}
	}
	agents := make([]Agent, opts.Agents)
	// Stamp latency on the trailing latencyCount agents so the coordinator
	// (agent-0) always stays local/fast.
	latencyFrom := opts.Agents - latencyCount
	for i := 0; i < opts.Agents; i++ {
		id := fmt.Sprintf("agent-%d", i)
		role := "worker"
		if i == 0 {
			role = "coordinator"
		}
		a := Agent{
			ID:       id,
			SlimName: fmt.Sprintf("agntcy/bench-v2/%s", id),
			P2PPort:  9700 + i*11,
			Role:     role,
		}
		if latencyCount > 0 && i >= latencyFrom {
			a.LatencyMs = opts.LatencyMs
		}
		agents[i] = a
	}

	s := &ConsensusScenario{
		APIVersion: "bench.agntcy.io/v2",
		Kind:       "ConsensusScenario",
		Metadata: Metadata{
			Name:        name,
			Domain:      opts.Family,
			Description: matrixDescription(opts, latencyCount),
		},
		Spec: Spec{
			Agents:             opts.Agents,
			ThinkTimeMs:        opts.ThinkTimeMs,
			FindingEmitDelayMs: maxInt64(1, opts.ThinkTimeMs/10),
			MaxRounds:          200,
			TargetMode:         TargetModeMajority,
			Seed:               opts.Seed,
			ValueSpace:         3,
			PayloadBytes:       opts.PayloadBytes,
			Epochs:             opts.Epochs,
			MaxEpochTimeMs:     opts.MaxEpochTimeMs,
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

func matrixDescription(opts GenerateOptions, latencyCount int) string {
	if !opts.MatrixNaming {
		return "Generated consensus scenario for transport sweeps"
	}
	desc := fmt.Sprintf("Matrix cell: %d agents, %d B payload, think %d ms",
		opts.Agents, opts.PayloadBytes, opts.ThinkTimeMs)
	if opts.LatencyMs > 0 {
		desc += fmt.Sprintf("; %d ms one-way on %d distant-region agents (~⅓ of group)",
			opts.LatencyMs, latencyCount)
	} else {
		desc += "; all agents local (0 ms latency)"
	}
	return desc
}

func NormalizeFamily(input string) string {
	return strings.TrimSpace(input)
}

// Latency-injection models, selected via the BENCH_LAT_MODEL env var.
const (
	// LatencyModelBoundary applies each agent's latency at that agent's own
	// send/receive boundary (identical for both transports). Differences come
	// only from topology/backpressure.
	LatencyModelBoundary = "boundary"
	// LatencyModelRelay accounts for latency at the network hop: the P2P
	// streaming relay pays each distant agent's delay on both legs (agent->relay
	// and relay->agent) during its sequential fan-out, while native SLIM
	// multicast pays it once per direct delivery. This models the real 1-hop vs
	// 2-hop geo-distribution penalty of a central relay.
	LatencyModelRelay = "relay"
)

// LatencyModel returns the configured latency-injection model, defaulting to
// LatencyModelRelay when BENCH_LAT_MODEL is unset or unrecognized. The relay
// model is the shipped default because it produces the cleanest, deterministic
// picture of a central relay's geo penalty; set BENCH_LAT_MODEL=boundary to use
// the agent-boundary alternative. For zero-latency scenarios both models are
// identical (no delay is injected anywhere).
func LatencyModel() string {
	switch strings.TrimSpace(os.Getenv("BENCH_LAT_MODEL")) {
	case LatencyModelBoundary:
		return LatencyModelBoundary
	default:
		return LatencyModelRelay
	}
}
