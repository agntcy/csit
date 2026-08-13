// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Command runner is the SLIM benchmark driver. It acts as a group-session
// moderator (creates the session, invites every agent, broadcasts a single
// start) and then becomes a passive observer: it never relays findings. Agents
// broadcast findings to each other directly over the SLIM dataplane and push
// their snapshots to the runner the instant they converge. Global consensus is
// detected event-driven from those pushed snapshots.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/benchlog"
	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/consensus"
	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/metrics"
	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/scenario"
	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/slim/internal/protocol"
	slim "github.com/agntcy/slim-bindings-go"
)

const (
	runnerSlimName      = "agntcy/bench-v2/runner"
	groupChannelName    = "agntcy/bench-v2/consensus"
	defaultSharedSecret = "demo-shared-secret-min-32-chars!!"
)

func main() {
	scenarioPath := flag.String("scenario", "", "path to consensus scenario yaml")
	endpoint := flag.String("endpoint", "http://127.0.0.1:46357", "SLIM dataplane endpoint")
	agentBin := flag.String("agent-bin", "", "path to slim-agent binary")
	outputJSON := flag.String("output-json", "", "write run metrics json")
	outputTSV := flag.String("output-tsv", "", "append run metrics tsv")
	runID := flag.Int("run-id", 0, "repetition index for statistical runs")
	waitReady := flag.Duration("wait-ready", 3*time.Second, "wait for agents to start")
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
		agentPath = os.Getenv("SLIM_AGENT_BIN")
	}
	if agentPath == "" {
		log.Fatal("set --agent-bin or SLIM_AGENT_BIN")
	}

	procs := startAgents(s, agentPath, *endpoint, *scenarioPath, *quiet)
	defer stopAgents(procs)

	mod, err := newModerator(*endpoint, s)
	if err != nil {
		log.Fatalf("moderator: %v", err)
	}
	defer mod.Close()

	// Let agents come up and reach ListenForSessionAsync before inviting.
	time.Sleep(*waitReady)
	if err := mod.InviteAll(); err != nil {
		log.Fatalf("invite agents: %v", err)
	}

	runStart := time.Now()
	benchlog.SetRunStart(runStart)

	result := metrics.RunResult{
		RunID:          *runID,
		ScenarioName:   s.Metadata.Name,
		Domain:         s.Metadata.Domain,
		Implementation: benchlog.ImplSLIM,
		Agents:         len(s.Agents),
		ThinkTimeMs:    s.Spec.ThinkTimeMs,
		PayloadBytes:   s.Spec.PayloadBytes,
		LatencyMs:      s.RepresentativeLatencyMs(),
		Epochs:         s.Spec.Epochs,
	}

	n := len(s.Agents)
	maxEpochTime := time.Duration(s.Spec.MaxEpochTimeMs) * time.Millisecond
	var (
		wallSamples  []int64
		allSnapshots []consensus.AgentSnapshot
	)
	for epoch := 0; epoch < s.Spec.Epochs; epoch++ {
		log.Printf("epoch %d/%d started (budget %s)", epoch+1, s.Spec.Epochs, maxEpochTime)
		if err := mod.Start(epoch); err != nil {
			log.Fatalf("broadcast start (epoch %d): %v", epoch, err)
		}
		epochStart := time.Now()
		latest := map[int]consensus.AgentSnapshot{}
		deadline := epochStart.Add(maxEpochTime)
		success := false
		for time.Now().Before(deadline) {
			timeout := time.Second
			msg, err := mod.session.GetMessageAsync(&timeout)
			if err != nil {
				if isTimeout(err) {
					continue
				}
				break
			}
			env, derr := protocol.Decode(msg.Payload)
			if derr != nil || env.Kind != protocol.KindSnapshot || env.Snapshot == nil {
				continue
			}
			// Ignore stragglers from other epochs.
			if env.Snapshot.Epoch != epoch {
				continue
			}
			latest[env.Snapshot.AgentIndex] = *env.Snapshot
			if len(latest) == n {
				if ok, _ := consensus.GlobalConsensus(snapshotSlice(latest)); ok {
					success = true
					break
				}
			}
		}
		allSnapshots = append(allSnapshots, snapshotSlice(latest)...)
		epochWall := time.Since(epochStart)
		if success {
			result.EpochsSucceeded++
			wallSamples = append(wallSamples, epochWall.Milliseconds())
			log.Printf("epoch %d/%d ok wall_ms=%d", epoch+1, s.Spec.Epochs, epochWall.Milliseconds())
		} else {
			result.EpochsFailed++
			log.Printf("epoch %d/%d failed after %s (budget %s exhausted)", epoch+1, s.Spec.Epochs, epochWall.Round(time.Millisecond), maxEpochTime)
		}
	}

	result.Success = result.EpochsFailed == 0
	result = aggregateResult(result, wallSamples, allSnapshots)

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

// moderator owns the group session: it creates it, invites agents, and
// broadcasts the start signal. It does not relay any traffic.
type moderator struct {
	app     *slim.App
	connID  uint64
	session *slim.Session
	agents  []scenario.Agent
}

func newModerator(endpoint string, s *scenario.ConsensusScenario) (*moderator, error) {
	slim.InitializeWithDefaults()
	service := slim.GetGlobalService()

	name, err := slim.NameFromString(runnerSlimName)
	if err != nil {
		return nil, err
	}
	app, err := service.CreateAppWithSecret(name, defaultSharedSecret)
	if err != nil {
		return nil, err
	}
	connID, err := service.Connect(slim.NewInsecureClientConfig(endpoint))
	if err != nil {
		app.Destroy()
		return nil, err
	}
	if err := app.Subscribe(name, &connID); err != nil {
		app.Destroy()
		return nil, err
	}

	channelName, err := slim.NameFromString(groupChannelName)
	if err != nil {
		app.Destroy()
		return nil, err
	}
	interval := 5 * time.Second
	maxRetries := uint32(5)
	config := slim.SessionConfig{
		SessionType: slim.SessionTypeGroup,
		MaxRetries:  &maxRetries,
		Interval:    &interval,
		Metadata:    map[string]string{},
	}
	session, err := app.CreateSessionAndWaitAsync(config, channelName)
	if err != nil {
		app.Destroy()
		return nil, err
	}
	// Give the session a moment to establish before inviting.
	time.Sleep(100 * time.Millisecond)

	return &moderator{app: app, connID: connID, session: session, agents: s.Agents}, nil
}

func (m *moderator) InviteAll() error {
	for _, a := range m.agents {
		name, err := slim.NameFromString(a.SlimName)
		if err != nil {
			return err
		}
		if err := m.app.SetRouteAsync(name, m.connID); err != nil {
			return fmt.Errorf("set route %s: %w", a.SlimName, err)
		}
		if err := m.session.InviteAndWaitAsync(name); err != nil {
			return fmt.Errorf("invite %s: %w", a.SlimName, err)
		}
	}
	return nil
}

func (m *moderator) Start(epoch int) error {
	payload, err := protocol.Encode(protocol.Envelope{Kind: protocol.KindStart, Epoch: epoch})
	if err != nil {
		return err
	}
	return m.session.PublishAndWaitAsync(payload, nil, nil)
}

func (m *moderator) Close() {
	if m.session != nil && m.app != nil {
		_ = m.app.DeleteSessionAndWaitAsync(m.session)
	}
	if m.app != nil {
		m.app.Destroy()
	}
}

// aggregateResult folds the snapshots collected across every epoch into the run
// result. wallSamples holds the wall-clock duration of each successful epoch.
func aggregateResult(result metrics.RunResult, wallSamples []int64, snapshots []consensus.AgentSnapshot) metrics.RunResult {
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
	// Native group session: findings are broadcast peer-to-peer, no relay.
	result.StreamRPCCount = totalEmitted
	result.CoordFanoutMS = 0
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

func snapshotSlice(m map[int]consensus.AgentSnapshot) []consensus.AgentSnapshot {
	out := make([]consensus.AgentSnapshot, 0, len(m))
	for _, snap := range m {
		out = append(out, snap)
	}
	return out
}

func startAgents(s *scenario.ConsensusScenario, agentBin, endpoint, scenarioFile string, quiet bool) []*exec.Cmd {
	var procs []*exec.Cmd
	for i, agent := range s.Agents {
		args := []string{
			"--slim-name", agent.SlimName,
			"--endpoint", endpoint,
			"--scenario-file", scenarioFile,
			"--agent-index", fmt.Sprintf("%d", i),
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

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timed out") || strings.Contains(msg, "timeout")
}
