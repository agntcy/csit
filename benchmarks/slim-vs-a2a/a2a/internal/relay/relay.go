// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package relay implements the runner-side relay hub for the A2A topology.
//
// The hub is the central point every finding must pass through, because A2A has
// no peer multicast. It hosts an A2A server exposing OpStreamRelay: each agent
// subscribes and receives a server stream of every relayed finding. The runner
// separately subscribes to each agent's OpStreamFindings stream and calls
// Broadcast for every finding, which fans it out to all other agents' relay
// streams. Broadcast drops findings whose epoch does not match the hub's
// current epoch so stragglers from a prior attempt are not relayed. This
// relay fan-out is exactly the work native SLIM avoids.
package relay

import (
	"context"
	"fmt"
	"iter"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/a2aproject/a2a-go/v2/a2asrv/taskstore"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/protocol"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/consensus"
	"google.golang.org/grpc"
)

type subscriber struct {
	agentIndex int
	ch         chan consensus.Finding
	// done is closed to terminate the relay execution serving this subscriber.
	// a2a-go runs executors on a detached context, so a client disconnect never
	// cancels the server-side stream; this is the only way to end it.
	done     chan struct{}
	stopOnce sync.Once
}

func (s *subscriber) stop() {
	s.stopOnce.Do(func() { close(s.done) })
}

type Hub struct {
	grpcPort int
	cardPort int

	mu             sync.Mutex
	subs           map[*subscriber]struct{}
	streamRPCCount int
	fanoutMS       int64
	// currentEpoch is the consensus attempt the hub will relay. Findings tagged
	// with a different epoch are dropped so stragglers from a prior attempt do
	// not fan out or grow relay-stream task history.
	currentEpoch atomic.Int32

	// relayLatency, when set, makes the hub account for per-agent network delay
	// at the relay hop: the sender's delay on the agent->relay leg (once) plus
	// each distant target's delay on the relay->agent leg during the sequential
	// fan-out. This models the two-hop cost a central relay pays for
	// geo-distributed members (native SLIM multicast pays it once, in parallel).
	relayLatency []time.Duration
}

func NewHub(grpcPort, cardPort int) *Hub {
	return &Hub{
		grpcPort: grpcPort,
		cardPort: cardPort,
		subs:     map[*subscriber]struct{}{},
	}
}

func (h *Hub) CardURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", h.cardPort)
}

// SetRelayLatency enables the relay-hop latency model with per-agent one-way
// delays indexed by agent index. Passing nil (or all-zero) leaves the hub
// latency-free (the boundary model applies delay at the agents instead).
func (h *Hub) SetRelayLatency(latencies []time.Duration) {
	h.relayLatency = latencies
}

func (h *Hub) latencyFor(agentIndex int) time.Duration {
	if agentIndex >= 0 && agentIndex < len(h.relayLatency) {
		return h.relayLatency[agentIndex]
	}
	return 0
}

// Serve binds the relay's gRPC and card listeners and serves them in the
// background. It returns once the listeners are bound so callers can hand the
// card URL to agents.
func (h *Hub) Serve() error {
	handler := a2asrv.NewHandler(
		&relayExecutor{hub: h},
		a2asrv.WithTaskStore(taskstore.NewInMemory(&taskstore.InMemoryStoreConfig{
			Authenticator: func(context.Context) (string, error) { return "slim-vs-a2a", nil },
		})),
	)

	card := &a2a.AgentCard{
		Name:        "consensus-v2-relay",
		Description: "SLIM vs A2A v2 relay hub",
		Version:     "1.0.0",
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(fmt.Sprintf("127.0.0.1:%d", h.grpcPort), a2a.TransportProtocolGRPC),
		},
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
	}

	cardListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", h.cardPort))
	if err != nil {
		return fmt.Errorf("relay card listen: %w", err)
	}
	grpcListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", h.grpcPort))
	if err != nil {
		_ = cardListener.Close()
		return fmt.Errorf("relay grpc listen: %w", err)
	}

	grpcHandler := a2agrpc.NewHandler(handler)
	grpcServer := grpc.NewServer()
	grpcHandler.RegisterWith(grpcServer)

	go func() {
		mux := http.NewServeMux()
		mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(card))
		_ = http.Serve(cardListener, mux)
	}()
	go func() {
		_ = grpcServer.Serve(grpcListener)
	}()
	return nil
}

// ResetStreams terminates every active relay subscription so agents open fresh
// OpStreamRelay tasks on the next epoch instead of growing one task forever.
func (h *Hub) ResetStreams() {
	h.mu.Lock()
	subs := make([]*subscriber, 0, len(h.subs))
	for s := range h.subs {
		subs = append(subs, s)
	}
	h.subs = map[*subscriber]struct{}{}
	h.mu.Unlock()
	for _, s := range subs {
		s.stop()
	}
}

// SetCurrentEpoch advances the relay filter to the given epoch. Only findings
// stamped with this epoch are broadcast; older (or future) epochs are ignored.
func (h *Hub) SetCurrentEpoch(epoch int) {
	h.currentEpoch.Store(int32(epoch))
}

// Broadcast fans a finding out to every subscriber except its producer.
func (h *Hub) Broadcast(f consensus.Finding) {
	if int(h.currentEpoch.Load()) != f.Epoch {
		return
	}
	start := time.Now()
	h.mu.Lock()
	targets := make([]*subscriber, 0, len(h.subs))
	for s := range h.subs {
		if s.agentIndex == f.AgentIndex {
			continue
		}
		targets = append(targets, s)
	}
	h.mu.Unlock()

	// Relay-hop latency model: pay the sender's uplink delay once (agent->relay)
	// before fanning out.
	if senderLat := h.latencyFor(f.AgentIndex); senderLat > 0 {
		time.Sleep(senderLat)
	}

	delivered := 0
	for _, s := range targets {
		// Relay-hop latency model: the hub pays each distant target's delay on
		// the relay->agent leg, sequentially, because a central relay fans out
		// through one point. This is exactly the head-of-line cost native SLIM
		// multicast avoids.
		if targetLat := h.latencyFor(s.agentIndex); targetLat > 0 {
			time.Sleep(targetLat)
		}
		select {
		case s.ch <- f:
			delivered++
		case <-s.done:
		case <-time.After(5 * time.Second):
		}
	}

	h.mu.Lock()
	h.streamRPCCount += delivered
	h.fanoutMS += time.Since(start).Milliseconds()
	h.mu.Unlock()
}

func (h *Hub) Stats() (streamRPCCount int, fanoutMS int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.streamRPCCount, h.fanoutMS
}

func (h *Hub) register(agentIndex int) *subscriber {
	s := &subscriber{
		agentIndex: agentIndex,
		ch:         make(chan consensus.Finding, 1024),
		done:       make(chan struct{}),
	}
	h.mu.Lock()
	// A new subscription for an agent supersedes its previous one: terminate any
	// existing subscriber for this agentIndex so the prior epoch's relay
	// execution ends instead of blocking forever on a detached context.
	for existing := range h.subs {
		if existing.agentIndex == agentIndex {
			existing.stop()
			delete(h.subs, existing)
		}
	}
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s
}

func (h *Hub) unregister(s *subscriber) {
	h.mu.Lock()
	delete(h.subs, s)
	h.mu.Unlock()
}

type relayExecutor struct {
	hub *Hub
}

func (e *relayExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	req, err := protocol.DecodeRequest(firstText(execCtx.Message))
	if err != nil {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, fmt.Errorf("decode request: %w", err))
		}
	}
	if req.Op != protocol.OpStreamRelay {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, fmt.Errorf("relay only serves stream_relay, got %q", req.Op))
		}
	}

	sub := e.hub.register(req.AgentIndex)
	return func(yield func(a2a.Event, error) bool) {
		defer e.hub.unregister(sub)
		task := a2a.NewSubmittedTask(execCtx, execCtx.Message)
		if !yield(task, nil) {
			return
		}
		for {
			select {
			case <-ctx.Done():
				yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
				return
			case <-sub.done:
				yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
				return
			case f, ok := <-sub.ch:
				if !ok {
					yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
					return
				}
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

func (e *relayExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
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
