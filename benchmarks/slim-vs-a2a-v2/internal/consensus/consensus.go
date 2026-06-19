// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package consensus

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/scenario"
)

const consensusConfidence = 0.55

type Finding struct {
	FindingID  int64   `json:"findingId"`
	AgentIndex int     `json:"agentIndex"`
	Round      int     `json:"round"`
	Value      int     `json:"value"`
	Confidence float64 `json:"confidence"`
	EmittedAt  int64   `json:"emittedAtNs"`
}

type Config struct {
	AgentIndex int
	AgentCount int
	ValueSpace int
	TargetMode string
	Seed       int64
}

type Engine struct {
	cfg Config

	mu sync.Mutex

	round            int
	value            int
	confidence       float64
	targetValue      int
	lastEmitValue    int
	lastEmitConf     float64
	lastEmitRound    int
	nextFindingID    int64
	distinctSupports map[int]map[int]struct{}
	convergedAt      int64
	consensusRound   int

	propMu          sync.Mutex
	propDurations   []int64
	findingsEmitted int
	findingsApplied int
}

func NewEngine(spec scenario.Spec, agentIndex int) *Engine {
	k := spec.ValueSpace
	if k <= 0 {
		k = 3
	}
	target := targetValue(spec, agentIndex)
	return &Engine{
		cfg: Config{
			AgentIndex: agentIndex,
			AgentCount: spec.Agents,
			ValueSpace: k,
			TargetMode: spec.TargetMode,
			Seed:       spec.Seed,
		},
		value:            agentIndex % k,
		confidence:       0.1,
		targetValue:      target,
		lastEmitValue:    -1,
		distinctSupports: map[int]map[int]struct{}{},
	}
}

func targetValue(spec scenario.Spec, agentIndex int) int {
	k := spec.ValueSpace
	if k <= 0 {
		k = 3
	}
	switch spec.TargetMode {
	case scenario.TargetModeMajority:
		return (spec.Agents / 2) % k
	default:
		return agentIndex % k
	}
}

func (e *Engine) TargetValue() int {
	return e.targetValue
}

func (e *Engine) Think() (finding *Finding, emit bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.round++
	e.refreshConfidenceLocked()

	if e.value == e.targetValue && e.confidence >= consensusConfidence {
		if e.convergedAt == 0 {
			e.convergedAt = time.Now().UnixNano()
			e.consensusRound = e.round
		}
	}

	if e.lastEmitValue == e.value && e.lastEmitConf == e.confidence && e.lastEmitRound == e.round {
		return nil, false
	}

	e.lastEmitValue = e.value
	e.lastEmitConf = e.confidence
	e.lastEmitRound = e.round
	e.nextFindingID++
	e.findingsEmitted++

	f := Finding{
		FindingID:  e.nextFindingID,
		AgentIndex: e.cfg.AgentIndex,
		Round:      e.round,
		Value:      e.value,
		Confidence: e.confidence,
		EmittedAt:  time.Now().UnixNano(),
	}
	return &f, true
}

func (e *Engine) refreshConfidenceLocked() {
	sources := e.distinctSupports[e.value]
	if len(sources)+1 >= (e.cfg.AgentCount+1)/2 {
		e.confidence += 0.12
		if e.confidence > 1 {
			e.confidence = 1
		}
	}
	if e.value != e.targetValue && e.round > 2 {
		e.value = e.targetValue
		e.confidence = maxFloat(e.confidence, 0.35)
	}
}

func (e *Engine) ApplyFinding(f Finding) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.findingsApplied++
	if _, ok := e.distinctSupports[f.Value]; !ok {
		e.distinctSupports[f.Value] = map[int]struct{}{}
	}
	e.distinctSupports[f.Value][f.AgentIndex] = struct{}{}

	if f.Confidence >= e.confidence || f.Value == e.targetValue {
		e.value = f.Value
		e.confidence = maxFloat(e.confidence, f.Confidence)
	}
	e.refreshConfidenceLocked()

	if e.value == e.targetValue && e.confidence >= consensusConfidence && e.convergedAt == 0 {
		e.convergedAt = time.Now().UnixNano()
		e.consensusRound = e.round
	}
}

func (e *Engine) RecordPropagation(emitNs int64, recvNs int64) {
	if emitNs <= 0 || recvNs <= emitNs {
		return
	}
	e.propMu.Lock()
	e.propDurations = append(e.propDurations, (recvNs-emitNs)/int64(time.Millisecond))
	e.propMu.Unlock()
}

func (e *Engine) HasLocalConsensus() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.value == e.targetValue && e.confidence >= consensusConfidence
}

func (e *Engine) LocalState() (value int, confidence float64, round int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.value, e.confidence, e.round
}

func (e *Engine) ConvergedAtNs() int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.convergedAt
}

func (e *Engine) ConsensusRound() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.consensusRound
}

func (e *Engine) FindingsEmitted() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.findingsEmitted
}

func (e *Engine) FindingsApplied() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.findingsApplied
}

func (e *Engine) PropagationStats() (avg, p95 int64) {
	e.propMu.Lock()
	defer e.propMu.Unlock()
	if len(e.propDurations) == 0 {
		return 0, 0
	}
	var sum int64
	for _, d := range e.propDurations {
		sum += d
	}
	avg = sum / int64(len(e.propDurations))
	p95 = percentile(e.propDurations, 0.95)
	return avg, p95
}

func EncodeFinding(f Finding) ([]byte, error) {
	return json.Marshal(f)
}

func DecodeFinding(data []byte) (Finding, error) {
	var f Finding
	err := json.Unmarshal(data, &f)
	return f, err
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func percentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

type AgentSnapshot struct {
	AgentIndex       int     `json:"agentIndex"`
	Value            int     `json:"value"`
	Confidence       float64 `json:"confidence"`
	Round            int     `json:"round"`
	HasConsensus     bool    `json:"hasConsensus"`
	ConvergedAtNs    int64   `json:"convergedAtNs"`
	ConsensusRound   int     `json:"consensusRound"`
	FindingsEmitted  int     `json:"findingsEmitted"`
	FindingsApplied  int     `json:"findingsApplied"`
	AvgPropagationMs int64   `json:"avgPropagationMs"`
	P95PropagationMs int64   `json:"p95PropagationMs"`
}

func (e *Engine) Snapshot() AgentSnapshot {
	value, conf, round := e.LocalState()
	avg, p95 := e.PropagationStats()
	return AgentSnapshot{
		AgentIndex:       e.cfg.AgentIndex,
		Value:            value,
		Confidence:       conf,
		Round:            round,
		HasConsensus:     e.HasLocalConsensus(),
		ConvergedAtNs:    e.ConvergedAtNs(),
		ConsensusRound:   e.ConsensusRound(),
		FindingsEmitted:  e.FindingsEmitted(),
		FindingsApplied:  e.FindingsApplied(),
		AvgPropagationMs: avg,
		P95PropagationMs: p95,
	}
}

func GlobalConsensus(snapshots []AgentSnapshot) (ok bool, value int) {
	if len(snapshots) == 0 {
		return false, 0
	}
	if !snapshots[0].HasConsensus {
		return false, 0
	}
	v := snapshots[0].Value
	for _, s := range snapshots[1:] {
		if !s.HasConsensus || s.Value != v {
			return false, 0
		}
	}
	return true, v
}

func ValidateSnapshots(snapshots []AgentSnapshot, expected int) error {
	if len(snapshots) != expected {
		return fmt.Errorf("expected %d snapshots, got %d", expected, len(snapshots))
	}
	return nil
}
