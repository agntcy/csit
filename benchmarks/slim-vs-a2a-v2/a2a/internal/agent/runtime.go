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
	"sync/atomic"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/a2a/internal/protocol"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/consensus"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a-v2/internal/scenario"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Runtime struct {
	scenario     *scenario.ConsensusScenario
	agentIndex   int
	agentID      string
	relayCardURL string

	engine *consensus.Engine

	outbound  chan consensus.Finding
	running   atomic.Bool
	done      chan struct{}
	startedAt time.Time
	relayOnce sync.Once
}

func NewRuntime(s *scenario.ConsensusScenario, agentIndex int, agentID, relayCardURL string) *Runtime {
	return &Runtime{
		scenario:     s,
		agentIndex:   agentIndex,
		agentID:      agentID,
		relayCardURL: relayCardURL,
		engine:       consensus.NewEngine(s.Spec, agentIndex),
		outbound:     make(chan consensus.Finding, 1024),
		done:         make(chan struct{}),
	}
}

// StartRelaySubscription dials the runner relay and subscribes to the relayed
// finding stream (OpStreamRelay). It runs once and retries until the relay is up.
func (r *Runtime) StartRelaySubscription(ctx context.Context) {
	r.relayOnce.Do(func() {
		go r.relayLoop(ctx)
	})
}

func (r *Runtime) relayLoop(ctx context.Context) {
	cli := r.dialRelay(ctx)
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
			r.applyFinding(f)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
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
		r.StartRun()
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

// StreamFindings serves OpStreamFindings: a long-lived server stream that emits
// every finding this agent produces. The first event establishes the task; each
// finding is delivered as a non-terminal Working status update so the stream
// stays open.
func (r *Runtime) StreamFindings(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
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

func (r *Runtime) StartRun() {
	if r.running.Swap(true) {
		return
	}
	r.startedAt = time.Now()
	benchlog.SetRunStart(r.startedAt)
	go r.runLoop()
}

func (r *Runtime) runLoop() {
	defer close(r.done)
	spec := r.scenario.Spec
	think := time.Duration(spec.ThinkTimeMs) * time.Millisecond
	emitGap := time.Duration(spec.FindingEmitDelayMs) * time.Millisecond

	for round := 0; round < spec.MaxRounds; round++ {
		time.Sleep(think)

		finding, emit := r.engine.Think()
		if emit && finding != nil {
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

func (r *Runtime) Close() {}

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
