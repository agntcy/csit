// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"iter"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/a2aproject/a2a-go/v2/a2asrv/push"
	"github.com/a2aproject/a2a-go/v2/a2asrv/taskstore"
	"google.golang.org/grpc"
)

type transportProtocol string

const (
	pendingRequestText                         = "pending"
	transportProtocolJSONRPC transportProtocol = "jsonrpc"
	transportProtocolREST    transportProtocol = "rest"
	transportProtocolGRPC    transportProtocol = "grpc"
)

type interopExecutor struct{}

func responseText(message *a2a.Message) string {
	return fmt.Sprintf("go server received: %s", firstText(message))
}

func buildTask(execCtx *a2asrv.ExecutorContext, state a2a.TaskState, text string) *a2a.Task {
	task := a2a.NewSubmittedTask(execCtx, execCtx.Message)
	task.Status = a2a.TaskStatus{
		State: state,
		Message: a2a.NewMessageForTask(
			a2a.MessageRoleAgent,
			task,
			a2a.NewTextPart(text),
		),
	}
	return task
}

func (*interopExecutor) Execute(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	state := a2a.TaskStateCompleted
	if firstText(execCtx.Message) == pendingRequestText {
		state = a2a.TaskStateWorking
	}
	response := buildTask(execCtx, state, responseText(execCtx.Message))

	return func(yield func(a2a.Event, error) bool) {
		yield(response, nil)
	}
}

func (*interopExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(
			a2a.NewStatusUpdateEvent(
				execCtx,
				a2a.TaskStateCanceled,
				a2a.NewMessageForTask(
					a2a.MessageRoleAgent,
					execCtx,
					a2a.NewTextPart("go server canceled task"),
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

func parseTransportProtocol(raw string) (transportProtocol, error) {
	switch transportProtocol(raw) {
	case transportProtocolJSONRPC:
		return transportProtocolJSONRPC, nil
	case transportProtocolREST:
		return transportProtocolREST, nil
	case transportProtocolGRPC:
		return transportProtocolGRPC, nil
	default:
		return "", fmt.Errorf("unsupported protocol %q", raw)
	}
}

func agentCard(cardPort int, protocol transportProtocol, grpcPort int) *a2a.AgentCard {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", cardPort)
	name := "CSIT Go JSON-RPC Agent"
	iface := a2a.NewAgentInterface(baseURL+"/rpc", a2a.TransportProtocolJSONRPC)

	if protocol == transportProtocolREST {
		name = "CSIT Go REST Agent"
		iface = a2a.NewAgentInterface(baseURL, a2a.TransportProtocolHTTPJSON)
	} else if protocol == transportProtocolGRPC {
		name = "CSIT Go gRPC Agent"
		iface = a2a.NewAgentInterface(
			fmt.Sprintf("127.0.0.1:%d", grpcPort),
			a2a.TransportProtocolGRPC,
		)
	}

	return &a2a.AgentCard{
		Name:        name,
		Description: "Go interoperability fixture for CSIT",
		Version:     "1.0.0",
		SupportedInterfaces: []*a2a.AgentInterface{
			iface,
		},
		Capabilities: a2a.AgentCapabilities{
			Streaming:         true,
			PushNotifications: true,
		},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
	}
}

func serveAgentCard(listener net.Listener, card *a2a.AgentCard) error {
	mux := http.NewServeMux()
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(card))
	return http.Serve(listener, mux)
}

func serveGRPC(listener net.Listener, handler a2asrv.RequestHandler) error {
	grpcHandler := a2agrpc.NewHandler(handler)
	server := grpc.NewServer()
	grpcHandler.RegisterWith(server)
	return server.Serve(listener)
}

func wrapRESTPushCreateCompat(base http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/pushNotificationConfigs") {
			base.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()

		var nested struct {
			TaskID string          `json:"taskId"`
			Config json.RawMessage `json:"config"`
		}

		if err := json.Unmarshal(body, &nested); err == nil && len(nested.Config) > 0 {
			body = nested.Config
		}

		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
		r.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

		base.ServeHTTP(w, r)
	})
}

func main() {
	port := flag.Int("port", 19091, "port for the fixture agent card server")
	grpcPort := flag.Int("grpc-port", 19092, "port for the gRPC fixture transport server")
	protocolFlag := flag.String("protocol", string(transportProtocolJSONRPC), "transport protocol to serve: jsonrpc, rest, or grpc")
	flag.Parse()

	protocol, err := parseTransportProtocol(*protocolFlag)
	if err != nil {
		log.Fatalf("failed to parse protocol: %v", err)
	}

	handler := a2asrv.NewHandler(
		&interopExecutor{},
		a2asrv.WithTaskStore(taskstore.NewInMemory(&taskstore.InMemoryStoreConfig{
			Authenticator: func(context.Context) (string, error) {
				return "csit-user", nil
			},
		})),
		a2asrv.WithPushNotifications(push.NewInMemoryStore(), push.NewHTTPPushSender(nil)),
	)

	if protocol == transportProtocolGRPC {
		cardListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
		if err != nil {
			log.Fatalf("failed to bind agent card listener: %v", err)
		}

		transportListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *grpcPort))
		if err != nil {
			log.Fatalf("failed to bind gRPC listener: %v", err)
		}

		card := agentCard(*port, protocol, *grpcPort)
		errCh := make(chan error, 2)

		go func() {
			errCh <- serveGRPC(transportListener, handler)
		}()
		go func() {
			errCh <- serveAgentCard(cardListener, card)
		}()

		log.Printf(
			"go grpc fixture listening on http://127.0.0.1:%d with card http://127.0.0.1:%d",
			*grpcPort,
			*port,
		)

		if err := <-errCh; err != nil {
			log.Fatalf("server stopped: %v", err)
		}
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatalf("failed to bind listener: %v", err)
	}

	mux := http.NewServeMux()
	switch protocol {
	case transportProtocolJSONRPC:
		mux.Handle("/rpc", a2asrv.NewJSONRPCHandler(handler))
		log.Printf("go jsonrpc fixture listening on http://127.0.0.1:%d", *port)
	case transportProtocolREST:
		mux.Handle("/", wrapRESTPushCreateCompat(a2asrv.NewRESTHandler(handler)))
		log.Printf("go rest fixture listening on http://127.0.0.1:%d", *port)
	}
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard(*port, protocol, *grpcPort)))

	if err := http.Serve(listener, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
