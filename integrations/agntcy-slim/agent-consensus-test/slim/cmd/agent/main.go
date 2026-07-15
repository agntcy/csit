// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/benchlog"
	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/scenario"
	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/slim/internal/agent"
)

func main() {
	slimName := flag.String("slim-name", "", "SLIM identity org/group/app")
	endpoint := flag.String("endpoint", "http://127.0.0.1:46357", "SLIM dataplane endpoint")
	scenarioFile := flag.String("scenario-file", "", "path to consensus scenario yaml")
	agentIndex := flag.Int("agent-index", 0, "agent index in scenario")
	quiet := flag.Bool("quiet", false, "disable benchmark logs")
	flag.Parse()

	if *quiet {
		benchlog.SetEnabled(false)
	}

	if *slimName == "" || *scenarioFile == "" {
		log.Fatal("--slim-name and --scenario-file are required")
	}

	s, err := scenario.LoadFile(*scenarioFile)
	if err != nil {
		log.Fatalf("load scenario: %v", err)
	}
	if *agentIndex < 0 || *agentIndex >= len(s.Agents) {
		log.Fatalf("agent-index out of range")
	}

	rt := agent.NewRuntime(s, *agentIndex, *slimName, *endpoint)
	if err := rt.Setup(); err != nil {
		log.Fatalf("setup: %v", err)
	}
	defer rt.Close()

	if !*quiet {
		fmt.Printf("SLIM_AGENT_READY name=%s index=%d scenario=%s\n", *slimName, *agentIndex, s.Metadata.Name)
	}

	// Block until the moderator invites us into the group session.
	if err := rt.Join(60 * time.Second); err != nil {
		log.Fatalf("join group session: %v", err)
	}

	if err := rt.Run(); err != nil {
		log.Printf("receive loop ended: %v", err)
	}
}
