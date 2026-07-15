// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package agent implements an A2A consensus agent for the relay topology.
//
// There is no peer-to-peer multicast in A2A, so every finding flows through the
// runner relay over two server-streaming legs:
//   - OpStreamFindings: this agent streams the findings it produces to the
//     runner (the runner is the stream client, this agent the stream server).
//   - OpStreamRelay: this agent subscribes to the runner and receives every
//     relayed finding (this agent is the stream client, the runner the server).
//
// OpStart / OpSnapshot remain unary calls the runner drives.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"sync"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/protocol"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/consensus"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/scenario"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Runtime struct {
	scenario     *scenario.ConsensusScenario
	agentIndex   int
	agentID      string
	relayCardURL string

	// latency is the extra one-way delay applied to this agent's finding sends
	// and receives, simulating an agent in a distant region.
	latency time.Duration

	engine *consensus.Engine

	outbound  chan consensus.Finding
	startedAt time.Time

	relayMu     sync.Mutex
	relayCancel context.CancelFunc
	// relayClient is dialed once and reused across epochs. Re-dialing every
	// epoch leaked a gRPC connection (and its server-side transport goroutines)
	// per attempt, which was the dominant source of cross-epoch slowdown.
	relayClient *a2aclient.Client

	// findingsDone terminates the previous epoch's OpStreamFindings execution
	// when a new one starts. a2a-go runs executors on a detached context, so a
	// runner disconnect never cancels the server stream; superseding is the only
	// way to keep the findings-leg executions from accumulating.
	findingsMu   sync.Mutex
	findingsDone chan struct{}

	mu sync.Mutex
	// lastStartedEpoch dedupes repeated OpStart for the same epoch;
	// currentEpoch is the attempt currently running (used to drop stale relayed
	// findings and to stop a superseded epoch loop).
	lastStartedEpoch int
	currentEpoch     int
}

func NewRuntime(s *scenario.ConsensusScenario, agentIndex int, agentID, relayCardURL string) *Runtime {
	var latency time.Duration
	// The agent boundary only applies latency in the "boundary" model. In the
	// "relay" model the central hub accounts for the two relay legs instead, so
	// the agent itself stays fast (see internal/relay).
	if scenario.LatencyModel() != scenario.LatencyModelRelay && agentIndex >= 0 && agentIndex < len(s.Agents) {
		latency = time.Duration(s.Agents[agentIndex].LatencyMs) * time.Millisecond
	}
	return &Runtime{
		scenario:         s,
		agentIndex:       agentIndex,
		agentID:          agentID,
		relayCardURL:     relayCardURL,
		latency:          latency,
		engine:           consensus.NewEngine(s.Spec, agentIndex),
		outbound:         make(chan consensus.Finding, 1024),
		lastStartedEpoch: -1,
		currentEpoch:     -1,
	}
}

// stopRelay ends the current OpStreamRelay subscription, if any.
func (r *Runtime) stopRelay() {
	r.relayMu.Lock()
	if r.relayCancel != nil {
		r.relayCancel()
		r.relayCancel = nil
	}
	r.relayMu.Unlock()
}

// startRelay dials the runner relay and subscribes to OpStreamRelay for the
// current epoch. Each epoch gets a fresh stream so a2a-go task history does
// not accumulate across attempts.
func (r *Runtime) startRelay() {
	r.stopRelay()
	relayCtx, cancel := context.WithCancel(context.Background())
	r.relayMu.Lock()
	r.relayCancel = cancel
	r.relayMu.Unlock()
	go r.relayLoop(relayCtx)
}

func (r *Runtime) drainOutbound() {
	for {
		select {
		case <-r.outbound:
		default:
			return
		}
	}
}

func (r *Runtime) relayLoop(ctx context.Context) {
	cli := r.relayConn(ctx)
	if cli == nil {
		return
	}
	req := protocol.Request{Op: protocol.OpStreamRelay, AgentIndex: r.agentIndex}
	text, err := json.Marshal(req)
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(string(text)))
		for ev, streamErr := range cli.SendStreamingMessage(ctx, &a2a.SendMessageRequest{Message: msg}) {
			if streamErr != nil {
				break
			}
			f, ok := findingFromEvent(ev)
			if !ok || f.AgentIndex == r.agentIndex {
				continue
			}
			// Drop stragglers from a different (usually previous) epoch.
			r.mu.Lock()
			cur := r.currentEpoch
			r.mu.Unlock()
			if f.Epoch != cur {
				continue
			}
			// Apply the receive-side (downlink) delay off the relay-stream loop
			// so a distant agent does not block draining its stream. The engine
			// is mutex-guarded, so concurrent async applies are safe.
			if r.latency > 0 {
				go func(f consensus.Finding) {
					time.Sleep(r.latency)
					r.applyFinding(f)
				}(f)
			} else {
				r.applyFinding(f)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// relayConn returns the shared relay client, dialing it once on first use. The
// connection outlives individual epochs, so it is created with a background
// context rather than the per-epoch relay context (which is cancelled at every
// epoch boundary).
func (r *Runtime) relayConn(ctx context.Context) *a2aclient.Client {
	r.relayMu.Lock()
	if r.relayClient != nil {
		cli := r.relayClient
		r.relayMu.Unlock()
		return cli
	}
	r.relayMu.Unlock()

	cli := r.dialRelay(context.Background())
	if cli == nil {
		return nil
	}
	r.relayMu.Lock()
	if r.relayClient == nil {
		r.relayClient = cli
	} else {
		// Lost a race; keep the existing connection and discard this one.
		_ = cli.Destroy()
	}
	shared := r.relayClient
	r.relayMu.Unlock()
	return shared
}

func (r *Runtime) dialRelay(ctx context.Context) *a2aclient.Client {
	for attempt := 0; attempt < 100; attempt++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		card, err := agentcard.DefaultResolver.Resolve(ctx, r.relayCardURL)
		if err == nil {
			cli, cerr := a2aclient.NewFromCard(ctx, card,
				a2agrpc.WithGRPCTransport(grpc.WithTransportCredentials(insecure.NewCredentials())))
			if cerr == nil {
				return cli
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

// HandleUnary serves the unary control ops (start, snapshot).
func (r *Runtime) HandleUnary(_ context.Context, req protocol.Request) protocol.Response {
	switch req.Op {
	case protocol.OpStart:
		r.startEpoch(req.Epoch)
		return protocol.Response{OK: true}
	case protocol.OpSnapshot:
		body, err := json.Marshal(r.engine.Snapshot())
		if err != nil {
			return protocol.Response{OK: false, Error: err.Error()}
		}
		return protocol.Response{OK: true, Body: string(body)}
	default:
		return protocol.Response{OK: false, Error: "unknown op"}
	}
}

// StreamFindings serves OpStreamFindings: a server stream that emits every
// finding this agent produces. The runner opens a fresh stream (hence a fresh
// task) each epoch, so findings-leg task history does not accumulate across
// epochs. The first event establishes the task; each finding is delivered as a
// non-terminal Working status update so the stream stays open for the epoch.
func (r *Runtime) StreamFindings(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	// A new findings stream supersedes the previous one so its execution ends
	// (a2a-go never cancels the executor's detached context on disconnect).
	done := make(chan struct{})
	r.findingsMu.Lock()
	if r.findingsDone != nil {
		close(r.findingsDone)
	}
	r.findingsDone = done
	r.findingsMu.Unlock()

	return func(yield func(a2a.Event, error) bool) {
		task := a2a.NewSubmittedTask(execCtx, execCtx.Message)
		if !yield(task, nil) {
			return
		}
		for {
			select {
			case <-ctx.Done():
				yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
				return
			case <-done:
				yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
				return
			case f := <-r.outbound:
				body, err := consensus.EncodeFinding(f)
				if err != nil {
					continue
				}
				msg := a2a.NewMessageForTask(a2a.MessageRoleAgent, execCtx, a2a.NewTextPart(string(body)))
				ev := a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, msg)
				if !yield(ev, nil) {
					return
				}
			}
		}
	}
}

// startEpoch resets the engine for a fresh attempt and launches the epoch loop.
// Repeated starts for the same (or older) epoch are ignored so a duplicated
// call cannot double-run an attempt.
func (r *Runtime) startEpoch(epoch int) {
	r.mu.Lock()
	if epoch <= r.lastStartedEpoch {
		r.mu.Unlock()
		return
	}
	r.lastStartedEpoch = epoch
	r.currentEpoch = epoch
	r.mu.Unlock()

	r.stopRelay()
	r.drainOutbound()
	r.engine.Reset(epoch)
	r.startedAt = time.Now()
	benchlog.SetRunStart(r.startedAt)
	r.startRelay()
	go r.runEpoch(epoch)
}

func (r *Runtime) runEpoch(epoch int) {
	spec := r.scenario.Spec
	think := time.Duration(spec.ThinkTimeMs) * time.Millisecond
	emitGap := time.Duration(spec.FindingEmitDelayMs) * time.Millisecond

	for round := 0; round < spec.MaxRounds; round++ {
		// Stop early if a newer epoch has superseded this one.
		r.mu.Lock()
		superseded := r.currentEpoch != epoch
		r.mu.Unlock()
		if superseded {
			return
		}

		time.Sleep(think)

		finding, emit := r.engine.Think()
		if emit && finding != nil {
			// Simulate this agent's uplink delay before its finding leaves.
			if r.latency > 0 {
				time.Sleep(r.latency)
			}
			select {
			case r.outbound <- *finding:
				benchlog.Finding(benchlog.ImplA2A, "published", r.agentIndex, finding.FindingID)
			case <-time.After(5 * time.Second):
				benchlog.Finding(benchlog.ImplA2A, "publish_timeout", r.agentIndex, finding.FindingID)
			}
			time.Sleep(emitGap)
		}

		if r.engine.HasLocalConsensus() {
			return
		}
	}
}

func (r *Runtime) applyFinding(f consensus.Finding) {
	recvNs := time.Now().UnixNano()
	r.engine.ApplyFinding(f)
	if f.EmittedAt > 0 {
		r.engine.RecordPropagation(f.EmittedAt, recvNs)
	}
	benchlog.Finding(benchlog.ImplA2A, "received", r.agentIndex, f.FindingID,
		fmt.Sprintf("from=%d", f.AgentIndex))
}

func (r *Runtime) Snapshot() consensus.AgentSnapshot {
	return r.engine.Snapshot()
}

func (r *Runtime) Close() {
	r.stopRelay()
	r.relayMu.Lock()
	if r.relayClient != nil {
		_ = r.relayClient.Destroy()
		r.relayClient = nil
	}
	r.relayMu.Unlock()

	r.findingsMu.Lock()
	if r.findingsDone != nil {
		close(r.findingsDone)
		r.findingsDone = nil
	}
	r.findingsMu.Unlock()
}

// findingFromEvent extracts a finding from a streamed event, ignoring the
// initial submitted-task event and any event without a message payload.
func findingFromEvent(ev a2a.Event) (consensus.Finding, bool) {
	su, ok := ev.(*a2a.TaskStatusUpdateEvent)
	if !ok || su.Status.Message == nil {
		return consensus.Finding{}, false
	}
	text := firstText(su.Status.Message)
	if text == "" {
		return consensus.Finding{}, false
	}
	f, err := consensus.DecodeFinding([]byte(text))
	if err != nil {
		return consensus.Finding{}, false
	}
	return f, true
}

func firstText(message *a2a.Message) string {
	if message == nil {
		return ""
	}
	for _, part := range message.Parts {
		if text := part.Text(); text != "" {
			return text
		}
	}
	return ""
}

// DecodeRequestText is used by the agent server executor.
func DecodeRequestText(text string) (protocol.Request, error) {
	return protocol.DecodeRequest(text)
}
