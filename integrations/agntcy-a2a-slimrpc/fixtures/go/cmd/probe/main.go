// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// CSIT fixture: A2A client over SLIMRPC v1; prints response text to stdout for the orchestrator.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	a2aslimrpcv1 "github.com/agntcy/slim-a2a-go/a2aslimrpc/v1"
	slim_bindings "github.com/agntcy/slim-bindings-go"
)

func main() {
	endpoint := flag.String("slim-endpoint", envOr("SLIM_SERVER", "http://127.0.0.1:46357"), "SLIM node endpoint")
	local := flag.String("local", "agntcy/a2a_csit_slim/client_go", "Full local SLIM identity ns/group/name")
	remote := flag.String("remote", "agntcy/a2a_csit_slim/server_go", "Full remote server identity")
	secret := flag.String("secret", envOr("SLIM_SHARED_SECRET", "my_shared_secret_for_testing_purposes_only"), "Shared secret")
	text := flag.String("text", "Hello there!", "Outbound text; response must contain this substring")
	flag.Parse()

	if err := run(*endpoint, *secret, *local, *remote, *text); err != nil {
		fmt.Fprintf(os.Stderr, "probe error: %v\n", err)
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

func run(endpoint, secret, localFull, remoteFull, want string) error {
	lns, lgr, lnm, err := parseIdentity(localFull)
	if err != nil {
		return err
	}
	rns, rgr, rnm, err := parseIdentity(remoteFull)
	if err != nil {
		return err
	}

	slim_bindings.InitializeWithDefaults()
	svc := slim_bindings.GetGlobalService()

	localName := slim_bindings.NewName(lns, lgr, lnm)
	app, err := svc.CreateAppWithSecret(localName, secret)
	if err != nil {
		return fmt.Errorf("create app: %w", err)
	}

	connID, err := svc.Connect(slim_bindings.NewInsecureClientConfig(endpoint))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	if err := app.Subscribe(localName, &connID); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	remoteName := slim_bindings.NewName(rns, rgr, rnm)
	channel := slim_bindings.ChannelNewWithConnection(app, remoteName, &connID)
	defer channel.Destroy()

	req := &a2a.SendMessageRequest{
		Message: &a2a.Message{
			ID:    a2a.NewMessageID(),
			Role:  a2a.MessageRoleUser,
			Parts: a2a.ContentParts{a2a.NewTextPart(want)},
		},
	}

	t := a2aslimrpcv1.NewTransport(channel)
	defer func() { _ = t.Destroy() }()

	result, err := t.SendMessage(context.Background(), nil, req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	obs := observe(result)
	emitObservation(obs)
	if !strings.Contains(obs.text, want) {
		return fmt.Errorf("response %q does not contain sent text %q", obs.text, want)
	}
	return nil
}

// observation is the parseable view of a SendMessage result consumed by the
// lifecycle specs (terminal task state + echoed artifact text).
type observation struct {
	kind            string // "task" | "message" | "unknown"
	state           string // a2a.TaskState string (e.g. TASK_STATE_COMPLETED); empty for a bare message
	artifactPresent bool
	text            string
}

func observe(result a2a.SendMessageResult) observation {
	switch r := result.(type) {
	case *a2a.Message:
		return observation{kind: "message", text: extractText(r)}
	case *a2a.Task:
		text, present := taskArtifactText(r)
		return observation{
			kind:            "task",
			state:           string(r.Status.State),
			artifactPresent: present,
			text:            text,
		}
	default:
		return observation{kind: "unknown", text: fmt.Sprintf("(unexpected result type %T)", result)}
	}
}

func taskArtifactText(t *a2a.Task) (string, bool) {
	var b strings.Builder
	present := false
	for _, artifact := range t.Artifacts {
		for _, part := range artifact.Parts {
			if text, ok := part.Content.(a2a.Text); ok {
				b.WriteString(string(text))
				present = true
			}
		}
	}
	return b.String(), present
}

// emitObservation prints the parseable lifecycle block (keys consumed by matrix_test.go)
// followed by the raw echoed text so the echo spec's substring check still holds.
func emitObservation(o observation) {
	fmt.Printf("CSIT_SLIM_RESULT_KIND=%s\n", o.kind)
	fmt.Printf("CSIT_SLIM_TASK_STATE=%s\n", o.state)
	fmt.Printf("CSIT_SLIM_ARTIFACT_PRESENT=%t\n", o.artifactPresent)
	fmt.Printf("CSIT_SLIM_ARTIFACT_TEXT=%s\n", o.text)
	fmt.Println(o.text)
}

func extractText(msg *a2a.Message) string {
	if msg == nil {
		return ""
	}
	var b strings.Builder
	for _, part := range msg.Parts {
		if text, ok := part.Content.(a2a.Text); ok {
			b.WriteString(string(text))
		}
	}
	return b.String()
}
