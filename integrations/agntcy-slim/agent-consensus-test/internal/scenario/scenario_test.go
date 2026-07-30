// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scenario_test

import (
	"testing"

	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/scenario"
)

func TestGenerateAndValidate(t *testing.T) {
	s, err := scenario.Generate(scenario.GenerateOptions{Agents: 5, ThinkTimeMs: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Agents) != 5 {
		t.Fatalf("agents=%d", len(s.Agents))
	}
	if s.Coordinator().ID != "agent-0" {
		t.Fatalf("coordinator=%s", s.Coordinator().ID)
	}
}

func TestMatrixScenarioFilename(t *testing.T) {
	if got := scenario.MatrixScenarioFilename(0, 10, 0); got != "matrix-lat0-10ag" {
		t.Fatalf("name=%q", got)
	}
	if got := scenario.MatrixScenarioFilename(100, 20, 10240); got != "matrix-lat100-20ag-10240b" {
		t.Fatalf("name=%q", got)
	}
	s, err := scenario.Generate(scenario.GenerateOptions{
		Agents:       10,
		ThinkTimeMs:  20,
		PayloadBytes: 4096,
		LatencyMs:    50,
		MatrixNaming: true,
		Family:       "matrix",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Metadata.Name != "matrix-lat50-10ag-4096b" {
		t.Fatalf("metadata.name=%q", s.Metadata.Name)
	}
	if s.RepresentativeLatencyMs() != 50 {
		t.Fatalf("latency=%d", s.RepresentativeLatencyMs())
	}
}
