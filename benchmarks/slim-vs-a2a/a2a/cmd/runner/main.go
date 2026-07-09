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
	"sync"
	"time"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/client"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/relay"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/consensus"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/scenario"
)

func main() {
	scenarioPath := flag.String("scenario", "", "path to consensus scenario yaml")
	agentBin := flag.String("agent-bin", "", "path to a2a-agent binary")
	outputJSON := flag.String("output-json", "", "write run metrics json")
	outputTSV := flag.String("output-tsv", "", "append run metrics tsv")
	waitReady := flag.Duration("wait-ready", 3*time.Second, "wait for agents to start")
	relayGRPCPort := flag.Int("relay-grpc-port", 9600, "relay hub gRPC port")
	relayCardPort := flag.Int("relay-card-port", 9601, "relay hub agent card port")
	snapshotIntervalMs := flag.Int("snapshot-interval-ms", 20, "delay between consensus snapshot polls")
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

	procs := startAgents(s, agentPath, *scenarioPath, hub.CardURL(), *quiet)
	defer stopAgents(procs)
	time.Sleep(*waitReady)

	overall := time.Duration(s.Spec.Epochs)*time.Duration(s.Spec.MaxEpochTimeMs)*time.Millisecond + 5*time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), overall)
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

	runStart := time.Now()
	benchlog.SetRunStart(runStart)

	result := metrics.RunResult{
		ScenarioName:   s.Metadata.Name,
		Domain:         s.Metadata.Domain,
		Implementation: benchlog.ImplA2A,
		Agents:         len(s.Agents),
		ThinkTimeMs:    s.Spec.ThinkTimeMs,
		PayloadBytes:   s.Spec.PayloadBytes,
		Epochs:         s.Spec.Epochs,
	}

	maxEpochTime := time.Duration(s.Spec.MaxEpochTimeMs) * time.Millisecond
	snapshotInterval := time.Duration(*snapshotIntervalMs) * time.Millisecond
	var (
		wallSamples  []int64
		allSnapshots []consensus.AgentSnapshot
	)
	for epoch := 0; epoch < s.Spec.Epochs; epoch++ {
		log.Printf("epoch %d/%d started (budget %s)", epoch+1, s.Spec.Epochs, maxEpochTime)
		hub.SetCurrentEpoch(epoch)

		// Open a fresh OpStreamFindings subscription to every agent for this
		// epoch, so a2a-go task history on the findings leg does not accumulate
		// across epochs. The subscriptions are torn down at the end of the
		// epoch (mirrors the per-epoch OpStreamRelay reset on the agent side).
		epochCtx, epochCancel := context.WithCancel(ctx)
		var subWG sync.WaitGroup
		for _, id := range agentIDs {
			subWG.Add(1)
			go func(agentID string) {
				defer subWG.Done()
				for {
					select {
					case <-epochCtx.Done():
						return
					default:
					}
					err := cli.SubscribeFindings(epochCtx, agentID, func(f consensus.Finding) {
						hub.Broadcast(f)
					})
					if err == nil || epochCtx.Err() != nil {
						return
					}
					time.Sleep(50 * time.Millisecond)
				}
			}(id)
		}
		// Let this epoch's finding subscriptions establish before agents start
		// producing (findings emitted before then buffer in the agent outbound).
		time.Sleep(300 * time.Millisecond)

		if err := cli.StartAll(ctx, agentIDs, epoch); err != nil {
			epochCancel()
			subWG.Wait()
			stopAgents(procs)
			log.Fatalf("start agents (epoch %d): %v", epoch, err)
		}
		epochStart := time.Now()
		deadline := epochStart.Add(maxEpochTime)
		var snapshots []consensus.AgentSnapshot
		success := false
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
			// Only accept consensus once every agent has reset into this epoch.
			if pollErr == nil && allForEpoch(snapshots, epoch) {
				if ok, _ := consensus.GlobalConsensus(snapshots); ok {
					success = true
					break
				}
			}
			time.Sleep(snapshotInterval)
		}
		allSnapshots = append(allSnapshots, snapshots...)
		epochWall := time.Since(epochStart)
		if success {
			result.EpochsSucceeded++
			wallSamples = append(wallSamples, epochWall.Milliseconds())
			log.Printf("epoch %d/%d ok wall_ms=%d", epoch+1, s.Spec.Epochs, epochWall.Milliseconds())
		} else {
			result.EpochsFailed++
			log.Printf("epoch %d/%d failed after %s (budget %s exhausted)", epoch+1, s.Spec.Epochs, epochWall.Round(time.Millisecond), maxEpochTime)
		}

		// Close this epoch's findings streams so the next epoch opens fresh
		// tasks instead of growing one findings task across the whole run.
		epochCancel()
		subWG.Wait()
	}

	result.Success = result.EpochsFailed == 0
	result = aggregateResult(result, wallSamples, allSnapshots, hub)

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

// allForEpoch reports whether every polled snapshot belongs to the given epoch,
// i.e. all agents have reset into it. Empty input is treated as not-yet-ready.
func allForEpoch(snapshots []consensus.AgentSnapshot, epoch int) bool {
	if len(snapshots) == 0 {
		return false
	}
	for _, snap := range snapshots {
		if snap.Epoch != epoch {
			return false
		}
	}
	return true
}

// aggregateResult folds the snapshots collected across every epoch into the run
// result. wallSamples holds the wall-clock duration of each successful epoch.
func aggregateResult(result metrics.RunResult, wallSamples []int64, snapshots []consensus.AgentSnapshot, hub *relay.Hub) metrics.RunResult {
	// Relay hub did all the fan-out work; report its real counters (cumulative
	// across every epoch).
	streamCount, fanoutMS := hub.Stats()
	result.StreamRPCCount = streamCount
	result.CoordFanoutMS = fanoutMS

	if len(snapshots) == 0 {
		result.Error = "no agent snapshots"
		return result
	}

	var (
		totalEmitted      int
		totalApplied      int
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
		if snap.AvgPropagationMs > 0 {
			propDurations = append(propDurations, snap.AvgPropagationMs)
		}
	}

	meanWall, maxWall := summarizeWall(wallSamples)
	result.ConsensusWallMS = meanWall
	result.LastAgentConvergeMS = maxWall
	if result.EpochsSucceeded == 0 {
		result.Error = "consensus not reached in any epoch"
	}

	result.ConsensusRound = maxConsensusRound
	if result.ConsensusRound == 0 {
		result.ConsensusRound = maxRound
	}
	result.FindingsEmitted = totalEmitted
	result.FindingsReceivedTotal = totalApplied
	result.AvgPropagationMS, result.P95PropagationMS = metrics.AggregatePropagation(propDurations)
	return result
}

// summarizeWall returns the mean and max of the per-epoch wall-clock samples.
func summarizeWall(wallSamples []int64) (mean, max int64) {
	if len(wallSamples) == 0 {
		return 0, 0
	}
	var sum int64
	for _, w := range wallSamples {
		sum += w
		if w > max {
			max = w
		}
	}
	return sum / int64(len(wallSamples)), max
}

func startAgents(s *scenario.ConsensusScenario, agentBin, scenarioFile, relayCardURL string, quiet bool) []*exec.Cmd {
	var procs []*exec.Cmd
	for i, agent := range s.Agents {
		args := []string{
			"--agent-id", agent.ID,
			"--grpc-port", fmt.Sprintf("%d", agent.A2APort),
			"--card-port", fmt.Sprintf("%d", s.CardPort(agent)),
			"--scenario-file", scenarioFile,
			"--agent-index", fmt.Sprintf("%d", i),
			"--relay-card-url", relayCardURL,
		}
		if quiet {
			args = append(args, "--quiet")
		}
		cmd := exec.Command(agentBin, args...)
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
