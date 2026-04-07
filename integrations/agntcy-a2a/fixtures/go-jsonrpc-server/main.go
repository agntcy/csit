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
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/a2aproject/a2a-go/v2/a2asrv/push"
	"github.com/a2aproject/a2a-go/v2/a2asrv/taskstore"
	"google.golang.org/grpc"
)

type transportProtocol string

const (
	pendingRequestText                             = "pending"
	messageOnlyRequestText                         = "message-only"
	taskFailureRequestText                         = "task-failure"
	multiTurnStartRequestText                      = "multi-turn start"
	multiTurnContinueRequestText                   = "multi-turn continue"
	streamingRequestText                           = "streaming"
	longRunningRequestText                         = "long-running"
	dataTypesRequestText                           = "data-types"
	extendedCardSchemeID                           = a2a.SecuritySchemeName("bearer_token")
	transportProtocolJSONRPC     transportProtocol = "jsonrpc"
	transportProtocolREST        transportProtocol = "rest"
	transportProtocolGRPC        transportProtocol = "grpc"
)

type interopExecutor struct{}

func responseText(message *a2a.Message) string {
	return fmt.Sprintf("go server received: %s", firstText(message))
}

func buildTask(execCtx *a2asrv.ExecutorContext, state a2a.TaskState, text string) *a2a.Task {
	task := a2a.NewSubmittedTask(execCtx, execCtx.Message)
	if execCtx.StoredTask != nil {
		history := append([]*a2a.Message{}, execCtx.StoredTask.History...)
		if execCtx.Message != nil && (len(history) == 0 || history[len(history)-1].ID != execCtx.Message.ID) {
			history = append(history, execCtx.Message)
		}
		task.History = history
	}
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

func buildDataTypesArtifact() *a2a.Artifact {
	filePart := a2a.NewFileURLPart("https://example.invalid/diagram.svg", "image/svg+xml")
	filePart.Filename = "diagram.svg"

	return &a2a.Artifact{
		ID:          a2a.NewArtifactID(),
		Name:        "data-types-artifact",
		Description: "Mixed content artifact for scenario parity",
		Parts: a2a.ContentParts{
			a2a.NewTextPart("structured summary"),
			a2a.NewDataPart(map[string]any{"kind": "report", "items": 2}),
			filePart,
		},
	}
}

func buildSkill(id string, description string) a2a.AgentSkill {
	return a2a.AgentSkill{
		ID:          id,
		Name:        id,
		Description: description,
		Tags:        []string{"csit", "scenario-parity"},
	}
}

func scenarioSkills() []a2a.AgentSkill {
	return []a2a.AgentSkill{
		buildSkill("message-only", "Returns a message response without creating a task."),
		buildSkill("task-lifecycle", "Creates, lists, fetches, and cancels tasks."),
		buildSkill("task-failure", "Returns a failed task response."),
		buildSkill("task-cancel", "Creates a cancelable working task."),
		buildSkill("multi-turn", "Requests more input before completing the task."),
		buildSkill("streaming", "Streams task and artifact updates."),
		buildSkill("long-running", "Returns early and completes asynchronously."),
		buildSkill("data-types", "Produces text, structured data, and URL parts."),
	}
}

func (*interopExecutor) Execute(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	requestText := firstText(execCtx.Message)

	return func(yield func(a2a.Event, error) bool) {
		switch requestText {
		case messageOnlyRequestText:
			yield(a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart("go server message-only response")), nil)
		case taskFailureRequestText:
			yield(buildTask(execCtx, a2a.TaskStateFailed, "go server failed task"), nil)
		case multiTurnStartRequestText:
			yield(buildTask(execCtx, a2a.TaskStateInputRequired, "go server needs more input"), nil)
		case multiTurnContinueRequestText:
			yield(buildTask(execCtx, a2a.TaskStateCompleted, "go server multi-turn completed"), nil)
		case streamingRequestText:
			working := buildTask(execCtx, a2a.TaskStateWorking, "go server streaming started")
			if !yield(working, nil) {
				return
			}
			artifactID := a2a.NewArtifactID()
			if !yield(&a2a.TaskArtifactUpdateEvent{
				TaskID:    execCtx.TaskID,
				ContextID: execCtx.ContextID,
				Artifact: &a2a.Artifact{
					ID:    artifactID,
					Parts: a2a.ContentParts{a2a.NewTextPart("streaming chunk 1")},
				},
			}, nil) {
				return
			}
			if !yield(a2a.NewArtifactUpdateEvent(execCtx, artifactID, a2a.NewTextPart("streaming chunk 2")), nil) {
				return
			}
			if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, a2a.NewMessageForTask(a2a.MessageRoleAgent, execCtx, a2a.NewTextPart("go server streaming complete"))), nil) {
				return
			}
		case longRunningRequestText:
			if !yield(buildTask(execCtx, a2a.TaskStateWorking, "go server long-running started"), nil) {
				return
			}
			time.Sleep(150 * time.Millisecond)
			if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, a2a.NewMessageForTask(a2a.MessageRoleAgent, execCtx, a2a.NewTextPart("go server long-running progress"))), nil) {
				return
			}
			time.Sleep(150 * time.Millisecond)
			yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, a2a.NewMessageForTask(a2a.MessageRoleAgent, execCtx, a2a.NewTextPart("go server long-running complete"))), nil)
		case dataTypesRequestText:
			task := buildTask(execCtx, a2a.TaskStateCompleted, "go server data-types ready")
			task.Artifacts = []*a2a.Artifact{buildDataTypesArtifact()}
			yield(task, nil)
		default:
			state := a2a.TaskStateCompleted
			if requestText == pendingRequestText {
				state = a2a.TaskStateWorking
			}
			yield(buildTask(execCtx, state, responseText(execCtx.Message)), nil)
		}
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

func agentCard(cardPort int, protocol transportProtocol, grpcPort int, extended bool) *a2a.AgentCard {
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
		Name: name,
		Description: func() string {
			if extended {
				return "Go interoperability fixture for CSIT (extended)"
			}
			return "Go interoperability fixture for CSIT"
		}(),
		Version: "1.0.0",
		SupportedInterfaces: []*a2a.AgentInterface{
			iface,
		},
		Capabilities: a2a.AgentCapabilities{
			Streaming:         true,
			PushNotifications: true,
			ExtendedAgentCard: true,
		},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		SecuritySchemes: a2a.NamedSecuritySchemes{
			extendedCardSchemeID: a2a.HTTPAuthSecurityScheme{
				Scheme:       "Bearer",
				BearerFormat: "JWT",
				Description:  "Bearer token authentication",
			},
		},
		Skills: scenarioSkills(),
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
		a2asrv.WithExtendedAgentCard(agentCard(*port, protocol, *grpcPort, true)),
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

		card := agentCard(*port, protocol, *grpcPort, false)
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
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard(*port, protocol, *grpcPort, false)))

	if err := http.Serve(listener, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
