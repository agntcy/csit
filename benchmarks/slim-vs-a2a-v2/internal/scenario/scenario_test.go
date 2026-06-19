// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scenario_test

import (
	"testing"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/scenario"
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
