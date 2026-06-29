// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"iter"
	"log"
	"net"
	"net/http"

	"github.com/a2aproject/a2a-go/v2/a2a"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/a2aproject/a2a-go/v2/a2asrv/taskstore"
	agentrt "github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/agent"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/protocol"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/scenario"
	"google.golang.org/grpc"
)

type consensusExecutor struct {
	runtime *agentrt.Runtime
}

func (e *consensusExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	reqText := firstText(execCtx.Message)
	req, err := agentrt.DecodeRequestText(reqText)
	if err != nil {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, fmt.Errorf("decode request: %w", err))
		}
	}

	if req.Op == protocol.OpStreamFindings {
		return e.runtime.StreamFindings(ctx, execCtx)
	}

	resp := e.runtime.HandleUnary(ctx, req)
	body, err := json.Marshal(resp)
	if err != nil {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, err)
		}
	}
	return func(yield func(a2a.Event, error) bool) {
		task := a2a.NewSubmittedTask(execCtx, execCtx.Message)
		state := a2a.TaskStateCompleted
		if !resp.OK {
			state = a2a.TaskStateFailed
		}
		task.Status = a2a.TaskStatus{
			State: state,
			Message: a2a.NewMessageForTask(
				a2a.MessageRoleAgent,
				task,
				a2a.NewTextPart(string(body)),
			),
		}
		yield(task, nil)
	}
}

func (e *consensusExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, a2a.NewMessageForTask(
			a2a.MessageRoleAgent, execCtx, a2a.NewTextPart(`{"ok":true}`),
		)), nil)
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

func main() {
	agentID := flag.String("agent-id", "", "logical agent id")
	grpcPort := flag.Int("grpc-port", 0, "A2A gRPC port")
	cardPort := flag.Int("card-port", 0, "agent card HTTP port")
	scenarioFile := flag.String("scenario-file", "", "consensus scenario yaml")
	agentIndex := flag.Int("agent-index", 0, "agent index")
	relayCardURL := flag.String("relay-card-url", "", "runner relay agent card base URL")
	flag.Parse()

	if *agentID == "" || *grpcPort == 0 || *scenarioFile == "" || *relayCardURL == "" {
		log.Fatal("agent-id, grpc-port, scenario-file, and relay-card-url are required")
	}
	card := *cardPort
	if card == 0 {
		card = *grpcPort + 1000
	}

	s, err := scenario.LoadFile(*scenarioFile)
	if err != nil {
		log.Fatalf("load scenario: %v", err)
	}

	rt := agentrt.NewRuntime(s, *agentIndex, *agentID, *relayCardURL)
	defer rt.Close()

	handler := a2asrv.NewHandler(
		&consensusExecutor{runtime: rt},
		a2asrv.WithTaskStore(taskstore.NewInMemory(&taskstore.InMemoryStoreConfig{
			Authenticator: func(context.Context) (string, error) { return "slim-vs-a2a", nil },
		})),
	)

	agentCard := &a2a.AgentCard{
		Name:        fmt.Sprintf("consensus-v2-%s", *agentID),
		Description: "SLIM vs A2A v2 consensus agent",
		Version:     "1.0.0",
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(fmt.Sprintf("127.0.0.1:%d", *grpcPort), a2a.TransportProtocolGRPC),
		},
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
	}

	cardListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", card))
	if err != nil {
		log.Fatalf("card listen: %v", err)
	}
	grpcListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *grpcPort))
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	grpcHandler := a2agrpc.NewHandler(handler)
	grpcServer := grpc.NewServer()
	grpcHandler.RegisterWith(grpcServer)

	go func() {
		mux := http.NewServeMux()
		mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard))
		if err := http.Serve(cardListener, mux); err != nil {
			log.Printf("card server stopped: %v", err)
		}
	}()

	// Subscribe to the runner relay stream as soon as we are up; it retries
	// until the relay server is reachable.
	rt.StartRelaySubscription(context.Background())

	fmt.Printf("A2A_AGENT_READY agent=%s grpc=%d card=%d\n", *agentID, *grpcPort, card)
	if err := grpcServer.Serve(grpcListener); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
