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

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/client"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/scheduler"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/plan"
)

func main() {
	planPath := flag.String("plan", "", "path to execution plan yaml")
	agentBin := flag.String("agent-bin", "", "path to a2a agent binary")
	outputJSON := flag.String("output-json", "", "write run metrics json")
	outputTSV := flag.String("output-tsv", "", "append run metrics tsv")
	waitReady := flag.Duration("wait-ready", 3*time.Second, "wait for agents to start")
	roundBudgetMS := flag.Int64("round-budget-ms", 0, "coordination round budget in ms (overrides plan)")
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

	agentPath := *agentBin
	if agentPath == "" {
		agentPath = os.Getenv("A2A_AGENT_BIN")
	}
	if agentPath == "" {
		log.Fatal("set --agent-bin or A2A_AGENT_BIN")
	}

	procs := startAgents(p, agentPath)
	time.Sleep(*waitReady)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cli, err := client.New(ctx, p, client.Config{
		RoundBudgetMS: p.EffectiveRoundBudgetMS(*roundBudgetMS),
	})
	if err != nil {
		stopAgents(procs)
		log.Fatalf("client: %v", err)
	}

	result := scheduler.New(p, cli).Run(ctx)

	time.Sleep(200 * time.Millisecond)
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

func startAgents(p *plan.ExecutionPlan, agentBin string) []*exec.Cmd {
	var procs []*exec.Cmd
	for _, agent := range p.Agents {
		cmd := exec.Command(
			agentBin,
			"--agent-id", agent.ID,
			"--grpc-port", fmt.Sprintf("%d", agent.A2APort),
			"--card-port", fmt.Sprintf("%d", p.CardPort(agent)),
		)
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
