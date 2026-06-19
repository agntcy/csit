// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Thin A2A probe adapter: executes a single A2A operation and writes the raw
// JSON result to stdout.  On A2A error, writes {"code":…,"message":"…"} to
// stderr and exits 1.  No assertions or orchestration live here — all test
// logic is in the Go/Ginkgo spec tree.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ── error helpers ─────────────────────────────────────────────────────────────

func probeErrorCode(err error) int {
	switch {
	case errors.Is(err, a2a.ErrTaskNotFound):
		return -32001
	case errors.Is(err, a2a.ErrTaskNotCancelable):
		return -32002
	case errors.Is(err, a2a.ErrPushNotificationNotSupported):
		return -32003
	case errors.Is(err, a2a.ErrUnsupportedOperation):
		return -32004
	default:
		return -32000
	}
}

func fail(code int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	data, _ := json.Marshal(map[string]any{"code": code, "message": msg})
	fmt.Fprintln(os.Stderr, string(data))
	os.Exit(1)
}

func failErr(err error) {
	fail(probeErrorCode(err), "%s", err.Error())
}

// ── output helpers ────────────────────────────────────────────────────────────

func printJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		fail(-32000, "marshal response: %v", err)
	}
	fmt.Println(string(data))
}

func envelope(typ string, data any) map[string]any {
	return map[string]any{"type": typ, "data": data}
}

// ── argument helpers ──────────────────────────────────────────────────────────

func flagValue(args []string, name string) string {
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	fail(-32000, "missing %s", name)
	return ""
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	cardURL := flag.String("card-url", "", "agent card URL")
	flag.Parse()

	if *cardURL == "" {
		fail(-32000, "missing --card-url")
	}

	rest := flag.Args()
	if len(rest) == 0 {
		fail(-32000, "missing subcommand")
	}
	subcommand := rest[0]
	args := rest[1:]

	ctx := context.Background()

	card, err := agentcard.DefaultResolver.Resolve(ctx, *cardURL)
	if err != nil {
		fail(-32000, "resolve agent card: %v", err)
	}

	client, err := a2aclient.NewFromCard(
		ctx,
		card,
		a2agrpc.WithGRPCTransport(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		fail(-32000, "create client: %v", err)
	}

	switch subcommand {

	// ── send-message ──────────────────────────────────────────────────────────
	case "send-message":
		var req a2a.SendMessageRequest
		if err := json.Unmarshal([]byte(flagValue(args, "--message-json")), &req); err != nil {
			fail(-32000, "parse send-message request: %v", err)
		}
		result, err := client.SendMessage(ctx, &req)
		if err != nil {
			failErr(err)
		}
		switch v := result.(type) {
		case *a2a.Task:
			printJSON(envelope("task", v))
		case *a2a.Message:
			printJSON(envelope("message", v))
		default:
			fail(-32000, "unexpected send-message result type")
		}

	// ── send-streaming-message ────────────────────────────────────────────────
	case "send-streaming-message":
		var req a2a.SendMessageRequest
		if err := json.Unmarshal([]byte(flagValue(args, "--message-json")), &req); err != nil {
			fail(-32000, "parse send-streaming-message request: %v", err)
		}
		for event, err := range client.SendStreamingMessage(ctx, &req) {
			if err != nil {
				failErr(err)
			}
			switch v := event.(type) {
			case *a2a.Task:
				printJSON(envelope("task", v))
			case *a2a.Message:
				printJSON(envelope("message", v))
			case *a2a.TaskStatusUpdateEvent:
				printJSON(envelope("task-status-update", v))
			case *a2a.TaskArtifactUpdateEvent:
				printJSON(envelope("task-artifact-update", v))
			default:
				fail(-32000, "unexpected streaming event type")
			}
		}

	// ── get-task ──────────────────────────────────────────────────────────────
	case "get-task":
		task, err := client.GetTask(ctx, &a2a.GetTaskRequest{
			ID: a2a.TaskID(flagValue(args, "--task-id")),
		})
		if err != nil {
			failErr(err)
		}
		printJSON(task)

	// ── cancel-task ───────────────────────────────────────────────────────────
	case "cancel-task":
		task, err := client.CancelTask(ctx, &a2a.CancelTaskRequest{
			ID: a2a.TaskID(flagValue(args, "--task-id")),
		})
		if err != nil {
			failErr(err)
		}
		printJSON(task)

	// ── list-tasks ────────────────────────────────────────────────────────────
	case "list-tasks":
		resp, err := client.ListTasks(ctx, &a2a.ListTasksRequest{
			ContextID: flagValue(args, "--context-id"),
		})
		if err != nil {
			failErr(err)
		}
		printJSON(resp)

	// ── create-push-config ────────────────────────────────────────────────────
	case "create-push-config":
		var cfg a2a.PushConfig
		if err := json.Unmarshal([]byte(flagValue(args, "--config-json")), &cfg); err != nil {
			fail(-32000, "parse push config: %v", err)
		}
		result, err := client.CreateTaskPushConfig(ctx, &cfg)
		if err != nil {
			failErr(err)
		}
		printJSON(result)

	// ── get-push-config ───────────────────────────────────────────────────────
	case "get-push-config":
		result, err := client.GetTaskPushConfig(ctx, &a2a.GetTaskPushConfigRequest{
			TaskID: a2a.TaskID(flagValue(args, "--task-id")),
			ID:     flagValue(args, "--config-id"),
		})
		if err != nil {
			failErr(err)
		}
		printJSON(result)

	// ── list-push-configs ─────────────────────────────────────────────────────
	case "list-push-configs":
		configs, err := client.ListTaskPushConfigs(ctx, &a2a.ListTaskPushConfigRequest{
			TaskID: a2a.TaskID(flagValue(args, "--task-id")),
		})
		if err != nil {
			failErr(err)
		}
		printJSON(a2a.ListTaskPushConfigResponse{Configs: configs})

	// ── delete-push-config ────────────────────────────────────────────────────
	case "delete-push-config":
		if err := client.DeleteTaskPushConfig(ctx, &a2a.DeleteTaskPushConfigRequest{
			TaskID: a2a.TaskID(flagValue(args, "--task-id")),
			ID:     flagValue(args, "--config-id"),
		}); err != nil {
			failErr(err)
		}

	// ── get-extended-card ─────────────────────────────────────────────────────
	case "get-extended-card":
		card, err := client.GetExtendedAgentCard(ctx, &a2a.GetExtendedAgentCardRequest{})
		if err != nil {
			failErr(err)
		}
		printJSON(card)

	default:
		fail(-32000, "unknown subcommand: %s", subcommand)
	}
}
