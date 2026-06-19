// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Command runner is the A2A benchmark driver and relay hub. Because A2A has no
// peer multicast, the runner is the explicit relay: it subscribes to every
// agent's findings stream (OpStreamFindings) and fans each finding out to all
// other agents over the relay stream (OpStreamRelay). It also drives start and
// polls snapshots to detect global consensus.
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

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/a2a/internal/client"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/a2a/internal/relay"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/consensus"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/metrics"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/scenario"
)

func main() {
	scenarioPath := flag.String("scenario", "", "path to consensus scenario yaml")
	agentBin := flag.String("agent-bin", "", "path to a2a-agent binary")
	outputJSON := flag.String("output-json", "", "write run metrics json")
	outputTSV := flag.String("output-tsv", "", "append run metrics tsv")
	waitReady := flag.Duration("wait-ready", 3*time.Second, "wait for agents to start")
	relayGRPCPort := flag.Int("relay-grpc-port", 9600, "relay hub gRPC port")
	relayCardPort := flag.Int("relay-card-port", 9601, "relay hub agent card port")
	quiet := flag.Bool("quiet", false, "disable benchmark logs")
	flag.Parse()

	if *quiet {
		benchlog.SetEnabled(false)
	}
	if *scenarioPath == "" {
		log.Fatal("--scenario is required")
	}

	s, err := scenario.LoadFile(*scenarioPath)
	if err != nil {
		log.Fatalf("load scenario: %v", err)
	}

	agentPath := *agentBin
	if agentPath == "" {
		agentPath = os.Getenv("A2A_AGENT_BIN")
	}
	if agentPath == "" {
		log.Fatal("set --agent-bin or A2A_AGENT_BIN")
	}

	// Start the relay hub first so agents can subscribe as they come up.
	hub := relay.NewHub(*relayGRPCPort, *relayCardPort)
	if err := hub.Serve(); err != nil {
		log.Fatalf("relay serve: %v", err)
	}

	procs := startAgents(s, agentPath, *scenarioPath, hub.CardURL())
	defer stopAgents(procs)
	time.Sleep(*waitReady)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cli, err := newClientWithRetry(ctx, s)
	if err != nil {
		stopAgents(procs)
		log.Fatalf("client: %v", err)
	}

	agentIDs := s.AgentIDs()
	agentIndexByID := map[string]int{}
	for i, a := range s.Agents {
		agentIndexByID[a.ID] = i
	}

	// Runner subscribes to every agent's findings stream and relays each finding
	// to all other agents. Retries keep the subscription alive across restarts.
	for _, id := range agentIDs {
		go func(agentID string) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				err := cli.SubscribeFindings(ctx, agentID, func(f consensus.Finding) {
					hub.Broadcast(f)
				})
				if err == nil || ctx.Err() != nil {
					return
				}
				time.Sleep(300 * time.Millisecond)
			}
		}(id)
	}

	// Let finding subscriptions establish before agents start producing.
	time.Sleep(500 * time.Millisecond)

	runStart := time.Now()
	benchlog.SetRunStart(runStart)
	if err := cli.StartAll(ctx, agentIDs); err != nil {
		stopAgents(procs)
		log.Fatalf("start agents: %v", err)
	}

	result := metrics.RunResult{
		ScenarioName:   s.Metadata.Name,
		Domain:         s.Metadata.Domain,
		Implementation: benchlog.ImplA2A,
		Agents:         len(s.Agents),
		ThinkTimeMs:    s.Spec.ThinkTimeMs,
	}

	var snapshots []consensus.AgentSnapshot
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		snapshots = snapshots[:0]
		var pollErr error
		for _, id := range agentIDs {
			snap, err := cli.Snapshot(ctx, id)
			if err != nil {
				pollErr = err
				break
			}
			snapshots = append(snapshots, snap)
		}
		if pollErr == nil {
			if ok, _ := consensus.GlobalConsensus(snapshots); ok {
				result.Success = true
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !result.Success && len(snapshots) > 0 {
		if ok, _ := consensus.GlobalConsensus(snapshots); ok {
			result.Success = true
		}
	}

	result = aggregateResult(result, runStart, snapshots, hub)

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
	if !result.Success {
		os.Exit(1)
	}
}

func newClientWithRetry(ctx context.Context, s *scenario.ConsensusScenario) (*client.Client, error) {
	var lastErr error
	for attempt := 0; attempt < 50; attempt++ {
		cli, err := client.New(ctx, s)
		if err == nil {
			return cli, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return nil, lastErr
}

func aggregateResult(result metrics.RunResult, runStart time.Time, snapshots []consensus.AgentSnapshot, hub *relay.Hub) metrics.RunResult {
	if len(snapshots) == 0 {
		result.Error = "no agent snapshots"
		result.ConsensusWallMS = time.Since(runStart).Milliseconds()
		return result
	}

	var (
		totalEmitted      int
		totalApplied      int
		lastConvergeMS    int64
		propDurations     []int64
		maxConsensusRound int
		maxRound          int
	)

	for _, snap := range snapshots {
		totalEmitted += snap.FindingsEmitted
		totalApplied += snap.FindingsApplied
		if snap.ConsensusRound > maxConsensusRound {
			maxConsensusRound = snap.ConsensusRound
		}
		if snap.Round > maxRound {
			maxRound = snap.Round
		}
		if snap.ConvergedAtNs > 0 {
			ms := (snap.ConvergedAtNs - runStart.UnixNano()) / int64(time.Millisecond)
			if ms > lastConvergeMS {
				lastConvergeMS = ms
			}
		}
		if snap.AvgPropagationMs > 0 {
			propDurations = append(propDurations, snap.AvgPropagationMs)
		}
	}

	if result.Success {
		result.ConsensusWallMS = time.Since(runStart).Milliseconds()
		if lastConvergeMS > 0 {
			result.ConsensusWallMS = lastConvergeMS
		}
	} else {
		result.ConsensusWallMS = time.Since(runStart).Milliseconds()
		result.Error = "consensus not reached"
	}

	result.ConsensusRound = maxConsensusRound
	if result.ConsensusRound == 0 {
		result.ConsensusRound = maxRound
	}
	result.FindingsEmitted = totalEmitted
	result.FindingsReceivedTotal = totalApplied
	result.LastAgentConvergeMS = lastConvergeMS
	result.AvgPropagationMS, result.P95PropagationMS = metrics.AggregatePropagation(propDurations)
	// Relay hub did all the fan-out work; report its real counters.
	streamCount, fanoutMS := hub.Stats()
	result.StreamRPCCount = streamCount
	result.CoordFanoutMS = fanoutMS
	return result
}

func startAgents(s *scenario.ConsensusScenario, agentBin, scenarioFile, relayCardURL string) []*exec.Cmd {
	var procs []*exec.Cmd
	for i, agent := range s.Agents {
		cmd := exec.Command(
			agentBin,
			"--agent-id", agent.ID,
			"--grpc-port", fmt.Sprintf("%d", agent.A2APort),
			"--card-port", fmt.Sprintf("%d", s.CardPort(agent)),
			"--scenario-file", scenarioFile,
			"--agent-index", fmt.Sprintf("%d", i),
			"--relay-card-url", relayCardURL,
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
