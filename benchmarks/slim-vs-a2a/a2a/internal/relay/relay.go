// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package relay implements the runner-side relay hub for the A2A topology.
//
// The hub is the central point every finding must pass through, because A2A has
// no peer multicast. It hosts an A2A server exposing OpStreamRelay: each agent
// subscribes and receives a server stream of every relayed finding. The runner
// separately subscribes to each agent's OpStreamFindings stream and calls
// Broadcast for every finding, which fans it out to all other agents' relay
// streams. This relay fan-out is exactly the work native SLIM avoids.
package relay

import (
	"context"
	"fmt"
	"iter"
	"net"
	"net/http"
	"sync"
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
}

type Hub struct {
	grpcPort int
	cardPort int

	mu             sync.Mutex
	subs           map[*subscriber]struct{}
	streamRPCCount int
	fanoutMS       int64
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

// Broadcast fans a finding out to every subscriber except its producer.
func (h *Hub) Broadcast(f consensus.Finding) {
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

	delivered := 0
	for _, s := range targets {
		select {
		case s.ch <- f:
			delivered++
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
	s := &subscriber{agentIndex: agentIndex, ch: make(chan consensus.Finding, 1024)}
	h.mu.Lock()
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
			case f := <-sub.ch:
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
