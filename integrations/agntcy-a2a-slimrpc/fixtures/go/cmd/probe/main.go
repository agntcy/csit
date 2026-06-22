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

// Scenario sentinels: outbound request text that drives a non-echo server response.
// Must match the sentinels in cmd/server/main.go and the Python fixtures byte-for-byte.
const (
	scenarioEcho              = "echo"
	sentinelMessageOnly       = "csit-scenario:message-only"
	sentinelTaskFailure       = "csit-scenario:task-failure"
	sentinelInputRequired     = "csit-scenario:input-required"
	sentinelStreaming         = "csit-scenario:streaming"
	sentinelCancel            = "csit-scenario:cancel"
	sentinelMultiTurn         = "csit-scenario:multi-turn"
	sentinelMultiTurnContinue = "csit-scenario:multi-turn-continue"
)

// probe transport modes selected per scenario.
const (
	modeUnary     = "unary"
	modeStreaming = "streaming"
	modeCancel    = "cancel"
	modeMultiTurn = "multi-turn"
)

func main() {
	endpoint := flag.String("slim-endpoint", envOr("SLIM_SERVER", "http://127.0.0.1:46357"), "SLIM node endpoint")
	local := flag.String("local", "agntcy/a2a_csit_slim/client_go", "Full local SLIM identity ns/group/name")
	remote := flag.String("remote", "agntcy/a2a_csit_slim/server_go", "Full remote server identity")
	secret := flag.String("secret", envOr("SLIM_SHARED_SECRET", "my_shared_secret_for_testing_purposes_only"), "Shared secret")
	text := flag.String("text", "Hello there!", "Outbound text for the echo scenario; response must contain this substring")
	scenario := flag.String("scenario", scenarioEcho, "Behavior to drive: echo, message-only, task-failure, input-required, streaming, task-cancel")
	flag.Parse()

	want, enforceEcho, mode, err := scenarioRequest(*scenario, *text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "probe error: %v\n", err)
		os.Exit(1)
	}

	if err := run(*endpoint, *secret, *local, *remote, want, enforceEcho, mode); err != nil {
		fmt.Fprintf(os.Stderr, "probe error: %v\n", err)
		os.Exit(1)
	}
}

// scenarioRequest maps a scenario selector to the outbound request text, whether the
// response must echo it, and the transport mode to drive. Non-echo scenarios send a
// fixed sentinel and only emit the observation block (asserted by the harness).
func scenarioRequest(scenario, text string) (want string, enforceEcho bool, mode string, err error) {
	switch scenario {
	case scenarioEcho, "":
		return text, true, modeUnary, nil
	case "message-only":
		return sentinelMessageOnly, false, modeUnary, nil
	case "task-failure":
		return sentinelTaskFailure, false, modeUnary, nil
	case "input-required":
		return sentinelInputRequired, false, modeUnary, nil
	case "streaming":
		return sentinelStreaming, false, modeStreaming, nil
	case "task-cancel":
		return sentinelCancel, false, modeCancel, nil
	case "multi-turn":
		// The probe drives both turns; the start turn sends sentinelMultiTurn.
		return sentinelMultiTurn, false, modeMultiTurn, nil
	default:
		return "", false, "", fmt.Errorf("unknown scenario %q", scenario)
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

func run(endpoint, secret, localFull, remoteFull, want string, enforceEcho bool, mode string) error {
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

	ctx := context.Background()
	var obs observation
	switch mode {
	case modeStreaming:
		obs, err = runStreaming(ctx, t, req)
	case modeCancel:
		obs, err = runCancel(ctx, t, req)
	case modeMultiTurn:
		obs, err = runMultiTurn(ctx, t, req)
	default:
		obs, err = runUnary(ctx, t, req)
	}
	if err != nil {
		return err
	}

	emitObservation(obs)
	if enforceEcho && !strings.Contains(obs.text, want) {
		return fmt.Errorf("response %q does not contain sent text %q", obs.text, want)
	}
	return nil
}

// runUnary performs a single SendMessage and observes the aggregated result.
func runUnary(ctx context.Context, t *a2aslimrpcv1.Transport, req *a2a.SendMessageRequest) (observation, error) {
	result, err := t.SendMessage(ctx, nil, req)
	if err != nil {
		return observation{}, fmt.Errorf("send message: %w", err)
	}
	return observe(result), nil
}

// runStreaming drives SendStreamingMessage, aggregating streamed events into one
// observation and counting how many events arrived (proves the stream was multi-event).
func runStreaming(ctx context.Context, t *a2aslimrpcv1.Transport, req *a2a.SendMessageRequest) (observation, error) {
	obs := observation{kind: "task"}
	var b strings.Builder
	events := 0
	for event, err := range t.SendStreamingMessage(ctx, nil, req) {
		if err != nil {
			return obs, fmt.Errorf("send streaming message: %w", err)
		}
		events++
		switch e := event.(type) {
		case *a2a.Task:
			obs.state = string(e.Status.State)
			if text, present := taskArtifactText(e); present {
				b.WriteString(text)
				obs.artifactPresent = true
			}
		case *a2a.TaskStatusUpdateEvent:
			obs.state = string(e.Status.State)
		case *a2a.TaskArtifactUpdateEvent:
			for _, part := range e.Artifact.Parts {
				if text, ok := part.Content.(a2a.Text); ok {
					b.WriteString(string(text))
					obs.artifactPresent = true
				}
			}
		case *a2a.Message:
			obs.kind = "message"
			b.WriteString(extractText(e))
		}
	}
	obs.streamEvents = events
	obs.text = b.String()
	return obs, nil
}

// runCancel creates a working task via the streaming RPC (a non-terminal task would
// block unary SendMessage, which only returns on a final/auth-required event or queue
// close), captures its task ID, then cancels it via CancelTask and observes the
// terminal (canceled) task.
func runCancel(ctx context.Context, t *a2aslimrpcv1.Transport, req *a2a.SendMessageRequest) (observation, error) {
	// Read just enough of the stream to learn the task ID, then stop: the task stays
	// working (non-terminal) so its subscription never ends on its own. Canceling the
	// stream context closes that RPC; CancelTask then drives the task to canceled.
	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()

	var taskID a2a.TaskID
	for event, err := range t.SendStreamingMessage(streamCtx, nil, req) {
		if err != nil {
			return observation{}, fmt.Errorf("send streaming message: %w", err)
		}
		switch e := event.(type) {
		case *a2a.Task:
			taskID = e.ID
		case *a2a.TaskStatusUpdateEvent:
			taskID = e.TaskID
		case *a2a.TaskArtifactUpdateEvent:
			taskID = e.TaskID
		}
		if taskID != "" {
			break
		}
	}
	cancelStream()
	if taskID == "" {
		return observation{kind: "unknown"}, fmt.Errorf("task-cancel scenario: no task id observed")
	}
	canceled, err := t.CancelTask(ctx, nil, &a2a.CancelTaskRequest{ID: taskID})
	if err != nil {
		return observation{}, fmt.Errorf("cancel task: %w", err)
	}
	text, present := taskArtifactText(canceled)
	return observation{
		kind:            "task",
		state:           string(canceled.Status.State),
		artifactPresent: present,
		text:            text,
	}, nil
}

// runMultiTurn drives a two-turn conversation on a single task: turn 1 sends the
// start sentinel and must reach input-required (capturing the server-assigned task
// and context IDs); turn 2 references those IDs to continue the same task, which the
// server completes with the multi-turn artifact. Only the final (completed)
// observation is returned, so the harness asserts it like the other task scenarios.
func runMultiTurn(ctx context.Context, t *a2aslimrpcv1.Transport, startReq *a2a.SendMessageRequest) (observation, error) {
	res1, err := t.SendMessage(ctx, nil, startReq)
	if err != nil {
		return observation{}, fmt.Errorf("multi-turn start: %w", err)
	}
	task1, ok := res1.(*a2a.Task)
	if !ok {
		return observation{kind: "unknown"}, fmt.Errorf("multi-turn start: expected a task, got %T", res1)
	}
	if task1.Status.State != a2a.TaskStateInputRequired {
		return observe(task1), fmt.Errorf("multi-turn start: expected input-required, got %s", task1.Status.State)
	}

	continueReq := &a2a.SendMessageRequest{
		Message: &a2a.Message{
			ID:        a2a.NewMessageID(),
			Role:      a2a.MessageRoleUser,
			Parts:     a2a.ContentParts{a2a.NewTextPart(sentinelMultiTurnContinue)},
			TaskID:    task1.ID,
			ContextID: task1.ContextID,
		},
	}
	res2, err := t.SendMessage(ctx, nil, continueReq)
	if err != nil {
		return observation{}, fmt.Errorf("multi-turn continue: %w", err)
	}
	return observe(res2), nil
}

// observation is the parseable view of a SendMessage result consumed by the
// lifecycle specs (terminal task state + echoed artifact text).
type observation struct {
	kind            string // "task" | "message" | "unknown"
	state           string // a2a.TaskState string (e.g. TASK_STATE_COMPLETED); empty for a bare message
	artifactPresent bool
	text            string
	streamEvents    int // number of events received over a streaming call (0 for unary)
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
	fmt.Printf("CSIT_SLIM_STREAM_EVENTS=%d\n", o.streamEvents)
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
