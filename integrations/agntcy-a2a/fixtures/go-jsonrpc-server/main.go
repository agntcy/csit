// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"iter"
	"log"
	"net"
	"net/http"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

type interopExecutor struct{}

func (*interopExecutor) Execute(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	response := a2a.NewMessage(
		a2a.MessageRoleAgent,
		a2a.NewTextPart(fmt.Sprintf("go server received: %s", firstText(execCtx.Message))),
	)

	return func(yield func(a2a.Event, error) bool) {
		yield(response, nil)
	}
}

func (*interopExecutor) Cancel(_ context.Context, _ *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {}
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

func agentCard(port int) *a2a.AgentCard {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	return &a2a.AgentCard{
		Name:        "CSIT Go JSON-RPC Agent",
		Description: "Go interoperability fixture for CSIT",
		Version:     "1.0.0",
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(baseURL+"/rpc", a2a.TransportProtocolJSONRPC),
		},
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
	}
}

func main() {
	port := flag.Int("port", 19091, "port for the JSON-RPC fixture server")
	flag.Parse()

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatalf("failed to bind listener: %v", err)
	}

	handler := a2asrv.NewHandler(&interopExecutor{})
	mux := http.NewServeMux()
	mux.Handle("/rpc", a2asrv.NewJSONRPCHandler(handler))
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard(*port)))

	log.Printf("go jsonrpc fixture listening on http://127.0.0.1:%d", *port)
	if err := http.Serve(listener, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
