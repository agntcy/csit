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
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/executor"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/protocol"
	"google.golang.org/grpc"
)

type dagExecutor struct {
	engine *executor.Engine
}

func (d *dagExecutor) Execute(ctx context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	reqText := firstText(execCtx.Message)
	req, err := protocol.DecodeRequest(reqText)
	if err != nil {
		return func(yield func(a2a.Event, error) bool) {
			yield(nil, fmt.Errorf("decode request: %w", err))
		}
	}

	resp := d.engine.Handle(ctx, req)
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

func (d *dagExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(
			a2a.NewStatusUpdateEvent(
				execCtx,
				a2a.TaskStateCanceled,
				a2a.NewMessageForTask(
					a2a.MessageRoleAgent,
					execCtx,
					a2a.NewTextPart(`{"ok":true}`),
				),
			),
			nil,
		)
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
	flag.Parse()

	if *agentID == "" || *grpcPort == 0 || *cardPort == 0 {
		log.Fatal("agent-id, grpc-port, and card-port are required")
	}

	handler := a2asrv.NewHandler(
		&dagExecutor{engine: executor.New()},
		a2asrv.WithTaskStore(taskstore.NewInMemory(&taskstore.InMemoryStoreConfig{
			Authenticator: func(context.Context) (string, error) { return "slim-vs-a2a", nil },
		})),
	)

	card := &a2a.AgentCard{
		Name:        fmt.Sprintf("slim-vs-a2a-%s", *agentID),
		Description: "SLIM vs A2A comparison agent",
		Version:     "1.0.0",
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(
				fmt.Sprintf("127.0.0.1:%d", *grpcPort),
				a2a.TransportProtocolGRPC,
			),
		},
		Capabilities:       a2a.AgentCapabilities{Streaming: false},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Skills: []a2a.AgentSkill{
			{ID: "dag-task", Name: "DAG task execution", Description: "Mock DAG task worker"},
		},
	}

	cardListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *cardPort))
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
		mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(card))
		if err := http.Serve(cardListener, mux); err != nil {
			log.Printf("card server stopped: %v", err)
		}
	}()

	fmt.Printf("A2A_AGENT_READY agent=%s grpc=%d card=%d\n", *agentID, *grpcPort, *cardPort)
	if err := grpcServer.Serve(grpcListener); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
