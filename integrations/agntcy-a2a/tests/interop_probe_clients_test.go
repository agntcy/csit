// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// This file defines the probeClient interface, the newClientFn type, and the
// external language probe client implementations for Go, Rust, Python, and .NET.
// Each client stores a basecmd (the command needed to invoke the probe binary) and a
// baseURL. The call() method appends --card-url <url> and the per-operation args, then
// runs the command and returns stdout/stderr. All probe CLIs share the same subcommand
// interface, so no per-language serialization logic is needed beyond what call() does.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

// probeClient is the unified A2A client interface used by all behavior specs.
// Both the native Go SDK and external language probes implement this.
type probeClient interface {
	SendMessage(ctx context.Context, req *a2a.SendMessageRequest) (a2a.SendMessageResult, error)
	SendStreamingMessage(ctx context.Context, req *a2a.SendMessageRequest) ([]a2a.Event, error)
	GetTask(ctx context.Context, req *a2a.GetTaskRequest) (*a2a.Task, error)
	CancelTask(ctx context.Context, req *a2a.CancelTaskRequest) (*a2a.Task, error)
	ListTasks(ctx context.Context, req *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error)
	CreatePushConfig(ctx context.Context, cfg *a2a.PushConfig) (*a2a.PushConfig, error)
	GetPushConfig(ctx context.Context, req *a2a.GetTaskPushConfigRequest) (*a2a.PushConfig, error)
	ListPushConfigs(ctx context.Context, req *a2a.ListTaskPushConfigRequest) ([]*a2a.PushConfig, error)
	DeletePushConfig(ctx context.Context, req *a2a.DeleteTaskPushConfigRequest) error
	GetExtendedCard(ctx context.Context, req *a2a.GetExtendedAgentCardRequest) (*a2a.AgentCard, error)
}

// newClientFn creates a probeClient bound to a specific baseURL.
type newClientFn func(baseURL string) (probeClient, error)

// probeError is returned when an external probe exits with code 1.
type probeError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *probeError) Error() string {
	return e.Message
}

func parseProbeError(stderr []byte) error {
	var pe probeError
	if err := json.Unmarshal(bytes.TrimSpace(stderr), &pe); err == nil && pe.Message != "" {
		return &pe
	}
	if len(stderr) > 0 {
		return fmt.Errorf("%s", strings.TrimSpace(string(stderr)))
	}
	return fmt.Errorf("probe exited with non-zero status")
}

// externalProbeClient dispatches A2A operations to an external language probe.
// basecmd is the full command prefix to invoke the probe binary
// (e.g., ["go", "run", "./fixtures/go-probe"]).
// call() appends "--card-url <baseURL>" and the subcommand args before executing.
type externalProbeClient struct {
	basecmd []string
	baseURL string
	dir     string // working directory; empty means componentRoot()
}

func (c *externalProbeClient) call(ctx context.Context, args ...string) (stdout []byte, stderr []byte, err error) {
	cmdArgs := make([]string, 0, len(c.basecmd)+2+len(args))
	cmdArgs = append(cmdArgs, c.basecmd[1:]...)
	cmdArgs = append(cmdArgs, "--card-url", c.baseURL)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, c.basecmd[0], cmdArgs...)
	dir := c.dir
	if dir == "" {
		dir = componentRoot()
	}
	cmd.Dir = dir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}

// probeMsgEnvelope wraps a send-message result from the probe.
type probeMsgEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// probeEventEnvelope wraps a streaming event from the probe.
type probeEventEnvelope = probeMsgEnvelope

func (c *externalProbeClient) SendMessage(ctx context.Context, req *a2a.SendMessageRequest) (a2a.SendMessageResult, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal send-message request: %w", err)
	}

	stdout, stderr, err := c.call(ctx, "send-message", "--message-json", string(reqJSON))
	if err != nil {
		return nil, parseProbeError(stderr)
	}

	var env probeMsgEnvelope
	if err := json.Unmarshal(stdout, &env); err != nil {
		return nil, fmt.Errorf("parse send-message response: %w\n%s", err, stdout)
	}

	switch env.Type {
	case "task":
		var task a2a.Task
		if err := json.Unmarshal(env.Data, &task); err != nil {
			return nil, fmt.Errorf("parse task result: %w", err)
		}
		return &task, nil
	case "message":
		var msg a2a.Message
		if err := json.Unmarshal(env.Data, &msg); err != nil {
			return nil, fmt.Errorf("parse message result: %w", err)
		}
		return &msg, nil
	default:
		return nil, fmt.Errorf("unknown send-message result type %q", env.Type)
	}
}

func (c *externalProbeClient) SendStreamingMessage(ctx context.Context, req *a2a.SendMessageRequest) ([]a2a.Event, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal send-streaming-message request: %w", err)
	}

	stdout, stderr, err := c.call(ctx, "send-streaming-message", "--message-json", string(reqJSON))
	if err != nil {
		return nil, parseProbeError(stderr)
	}

	var events []a2a.Event
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var env probeEventEnvelope
		if err := json.Unmarshal(line, &env); err != nil {
			return nil, fmt.Errorf("parse streaming event line: %w\n%s", err, line)
		}

		event, err := parseStreamEvent(env)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read streaming events: %w", err)
	}

	return events, nil
}

func parseStreamEvent(env probeEventEnvelope) (a2a.Event, error) {
	switch env.Type {
	case "task":
		var task a2a.Task
		if err := json.Unmarshal(env.Data, &task); err != nil {
			return nil, fmt.Errorf("parse task event: %w", err)
		}
		return &task, nil
	case "task-status-update":
		var event a2a.TaskStatusUpdateEvent
		if err := json.Unmarshal(env.Data, &event); err != nil {
			return nil, fmt.Errorf("parse task-status-update event: %w", err)
		}
		return &event, nil
	case "task-artifact-update":
		var event a2a.TaskArtifactUpdateEvent
		if err := json.Unmarshal(env.Data, &event); err != nil {
			return nil, fmt.Errorf("parse task-artifact-update event: %w", err)
		}
		return &event, nil
	case "message":
		var msg a2a.Message
		if err := json.Unmarshal(env.Data, &msg); err != nil {
			return nil, fmt.Errorf("parse message event: %w", err)
		}
		return &msg, nil
	default:
		return nil, fmt.Errorf("unknown stream event type %q", env.Type)
	}
}

func (c *externalProbeClient) GetTask(ctx context.Context, req *a2a.GetTaskRequest) (*a2a.Task, error) {
	stdout, stderr, err := c.call(ctx, "get-task", "--task-id", string(req.ID))
	if err != nil {
		return nil, parseProbeError(stderr)
	}

	var task a2a.Task
	if err := json.Unmarshal(stdout, &task); err != nil {
		return nil, fmt.Errorf("parse get-task response: %w\n%s", err, stdout)
	}
	return &task, nil
}

func (c *externalProbeClient) CancelTask(ctx context.Context, req *a2a.CancelTaskRequest) (*a2a.Task, error) {
	stdout, stderr, err := c.call(ctx, "cancel-task", "--task-id", string(req.ID))
	if err != nil {
		return nil, parseProbeError(stderr)
	}

	var task a2a.Task
	if err := json.Unmarshal(stdout, &task); err != nil {
		return nil, fmt.Errorf("parse cancel-task response: %w\n%s", err, stdout)
	}
	return &task, nil
}

func (c *externalProbeClient) ListTasks(ctx context.Context, req *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error) {
	stdout, stderr, err := c.call(ctx, "list-tasks", "--context-id", req.ContextID)
	if err != nil {
		return nil, parseProbeError(stderr)
	}

	var resp a2a.ListTasksResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("parse list-tasks response: %w\n%s", err, stdout)
	}
	return &resp, nil
}

func (c *externalProbeClient) CreatePushConfig(ctx context.Context, cfg *a2a.PushConfig) (*a2a.PushConfig, error) {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal push config: %w", err)
	}

	stdout, stderr, err := c.call(ctx, "create-push-config", "--config-json", string(cfgJSON))
	if err != nil {
		return nil, parseProbeError(stderr)
	}

	var result a2a.PushConfig
	if err := json.Unmarshal(stdout, &result); err != nil {
		return nil, fmt.Errorf("parse create-push-config response: %w\n%s", err, stdout)
	}
	return &result, nil
}

func (c *externalProbeClient) GetPushConfig(ctx context.Context, req *a2a.GetTaskPushConfigRequest) (*a2a.PushConfig, error) {
	stdout, stderr, err := c.call(ctx, "get-push-config",
		"--task-id", string(req.TaskID),
		"--config-id", req.ID,
	)
	if err != nil {
		return nil, parseProbeError(stderr)
	}

	var result a2a.PushConfig
	if err := json.Unmarshal(stdout, &result); err != nil {
		return nil, fmt.Errorf("parse get-push-config response: %w\n%s", err, stdout)
	}
	return &result, nil
}

func (c *externalProbeClient) ListPushConfigs(ctx context.Context, req *a2a.ListTaskPushConfigRequest) ([]*a2a.PushConfig, error) {
	stdout, stderr, err := c.call(ctx, "list-push-configs", "--task-id", string(req.TaskID))
	if err != nil {
		return nil, parseProbeError(stderr)
	}

	var resp a2a.ListTaskPushConfigResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("parse list-push-configs response: %w\n%s", err, stdout)
	}
	return resp.Configs, nil
}

func (c *externalProbeClient) DeletePushConfig(ctx context.Context, req *a2a.DeleteTaskPushConfigRequest) error {
	_, stderr, err := c.call(ctx, "delete-push-config",
		"--task-id", string(req.TaskID),
		"--config-id", req.ID,
	)
	if err != nil {
		return parseProbeError(stderr)
	}
	return nil
}

func (c *externalProbeClient) GetExtendedCard(ctx context.Context, req *a2a.GetExtendedAgentCardRequest) (*a2a.AgentCard, error) {
	stdout, stderr, err := c.call(ctx, "get-extended-card")
	if err != nil {
		return nil, parseProbeError(stderr)
	}

	var card a2a.AgentCard
	if err := json.Unmarshal(stdout, &card); err != nil {
		return nil, fmt.Errorf("parse get-extended-card response: %w\n%s", err, stdout)
	}
	return &card, nil
}

func newGoProbeClient(baseURL string) (probeClient, error) {
	return &externalProbeClient{
		basecmd: []string{"go", "run", "./fixtures/go-probe"},
		baseURL: baseURL,
	}, nil
}

func newRustProbeClient(baseURL string) (probeClient, error) {
	return &externalProbeClient{
		basecmd: []string{
			"cargo", "run",
			"--manifest-path", "fixtures/rust/Cargo.toml",
			"--bin", "interop-rust-probe",
			"--",
		},
		baseURL: baseURL,
	}, nil
}

func newPythonProbeClient(baseURL string) (probeClient, error) {
	uvCmd, err := resolveUvCommand()
	if err != nil {
		return nil, err
	}
	return &externalProbeClient{
		basecmd: []string{uvCmd, "run", "interop_python_probe.py"},
		baseURL: baseURL,
		dir:     filepath.Join(componentRoot(), "fixtures", "python"),
	}, nil
}

func newDotNetProbeClient(baseURL string) (probeClient, error) {
	dotnetCmd, err := resolveDotNetCommand()
	if err != nil {
		return nil, err
	}
	return &externalProbeClient{
		basecmd: []string{
			dotnetCmd, "run",
			"--project", "./fixtures/dotnet/InteropProbe",
			"--",
		},
		baseURL: baseURL,
	}, nil
}
