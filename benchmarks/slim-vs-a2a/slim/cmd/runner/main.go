// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/plan"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/client"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/scheduler"
)

func main() {
	planPath := flag.String("plan", "", "path to execution plan yaml")
	endpoint := flag.String("endpoint", "http://127.0.0.1:46357", "SLIM dataplane endpoint")
	agentBin := flag.String("agent-bin", "", "path to slim agent binary")
	outputJSON := flag.String("output-json", "", "write run metrics json")
	outputTSV := flag.String("output-tsv", "", "append run metrics tsv")
	waitReady := flag.Duration("wait-ready", 3*time.Second, "wait for agents to start")
	roundBudgetMS := flag.Int64("round-budget-ms", 0, "coordination round budget in ms (overrides plan)")
	coordMode := flag.String("coord-mode", "multicast", "coordination mode: multicast or unicast")
	quiet := flag.Bool("quiet", false, "disable benchmark timing logs")
	flag.Parse()

	if *quiet {
		benchlog.SetEnabled(false)
	}

	if *planPath == "" {
		log.Fatal("--plan is required")
	}

	p, err := plan.LoadFile(*planPath)
	if err != nil {
		log.Fatalf("load plan: %v", err)
	}

	mode := client.CoordModeMulticast
	if *coordMode == "unicast" {
		mode = client.CoordModeUnicast
	}

	agentPath := *agentBin
	if agentPath == "" {
		agentPath = os.Getenv("SLIM_AGENT_BIN")
	}
	if agentPath == "" {
		log.Fatal("set --agent-bin or SLIM_AGENT_BIN")
	}

	procs := startAgents(p, agentPath, *endpoint)
	time.Sleep(*waitReady)

	cli, err := client.New(*endpoint, p, client.Config{
		RoundBudgetMS: p.EffectiveRoundBudgetMS(*roundBudgetMS),
		CoordMode:     mode,
	})
	if err != nil {
		stopAgents(procs)
		log.Fatalf("client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := cli.SetupGroup(ctx, p); err != nil {
		cli.Close()
		stopAgents(procs)
		log.Fatalf("setup group: %v", err)
	}

	result := scheduler.New(p, cli).Run(ctx)

	time.Sleep(200 * time.Millisecond)
	cli.Close()
	stopAgents(procs)

	if *outputJSON != "" {
		if err := metrics.WriteJSON(*outputJSON, result); err != nil {
			log.Fatalf("write json: %v", err)
		}
	}
	if *outputTSV != "" {
		if err := metrics.AppendTSV(*outputTSV, result); err != nil {
			log.Fatalf("write tsv: %v", err)
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func startAgents(p *plan.ExecutionPlan, agentBin, endpoint string) []*exec.Cmd {
	var procs []*exec.Cmd
	for _, agent := range p.Agents {
		cmd := exec.Command(agentBin, "--slim-name", agent.SlimName, "--endpoint", endpoint)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			stopAgents(procs)
			log.Fatalf("start agent %s: %v", agent.ID, err)
		}
		procs = append(procs, cmd)
	}
	return procs
}

func stopAgents(procs []*exec.Cmd) {
	for _, cmd := range procs {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
}
