// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// CSIT fixture: minimal A2A echo agent over SLIMRPC (v1) for cross-language interop.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	a2aslimrpcv1 "github.com/agntcy/slim-a2a-go/a2aslimrpc/v1"
	slim_bindings "github.com/agntcy/slim-bindings-go"
)

const readyMarker = "CSIT_SLIM_SERVER_READY"

func main() {
	endpoint := flag.String("slim-endpoint", envOr("SLIM_SERVER", "http://127.0.0.1:46357"), "SLIM node endpoint")
	identity := flag.String("identity", "agntcy/a2a_csit_slim/server_go", "Full SLIM name ns/group/name")
	secret := flag.String("secret", envOr("SLIM_SHARED_SECRET", "my_shared_secret_for_testing_purposes_only"), "Shared secret for SLIM app")
	flag.Parse()

	ns, group, name, err := parseIdentity(*identity)
	if err != nil {
		slog.Error("bad --identity", "err", err)
		os.Exit(1)
	}

	slog.Info("starting CSIT slim echo server", "endpoint", *endpoint, "identity", *identity)

	if err := run(*endpoint, *secret, ns, group, name); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseIdentity(s string) (ns, group, name string, err error) {
	p := strings.Split(strings.Trim(s, "/"), "/")
	if len(p) != 3 {
		return "", "", "", fmt.Errorf("identity must be ns/group/name, got %q", s)
	}
	return p[0], p[1], p[2], nil
}

func run(endpoint, secret, ns, group, name string) error {
	slim_bindings.InitializeWithDefaults()
	svc := slim_bindings.GetGlobalService()

	slimName := slim_bindings.NewName(ns, group, name)
	app, err := svc.CreateAppWithSecret(slimName, secret)
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	connID, err := svc.Connect(slim_bindings.NewInsecureClientConfig(endpoint))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	if err := app.Subscribe(slimName, &connID); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	requestHandler := a2asrv.NewHandler(&echoExecutor{})
	server := slim_bindings.ServerNewWithConnection(app, slimName, &connID)
	a2aslimrpcv1.NewHandler(requestHandler).RegisterWith(server)

	if _, err := io.WriteString(os.Stdout, readyMarker+"\n"); err != nil {
		return err
	}
	return server.Serve()
}

type echoExecutor struct{}

var _ a2asrv.AgentExecutor = (*echoExecutor)(nil)

func (e *echoExecutor) Execute(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		if execCtx.StoredTask == nil {
			if !yield(a2a.NewSubmittedTask(execCtx, execCtx.Message), nil) {
				return
			}
		}
		if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil) {
			return
		}
		text := extractText(execCtx.Message)
		if !yield(a2a.NewArtifactEvent(execCtx, a2a.NewTextPart(text)), nil) {
			return
		}
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
	}
}

func (e *echoExecutor) Cancel(_ context.Context, execCtx *a2asrv.ExecutorContext) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCanceled, nil), nil)
	}
}

func extractText(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	for _, part := range msg.Parts {
		if text, ok := part.Content.(a2a.Text); ok {
			return string(text)
		}
	}
	return ""
}
