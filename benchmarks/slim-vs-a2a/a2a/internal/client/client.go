// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/protocol"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/plan"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	RoundBudgetMS int64
}

type Client struct {
	clients    map[string]*a2aclient.Client
	stats      *Stats
	coordStats *metrics.CoordStats
}

type Stats struct {
	ExecuteRPCCount    int
	SequentialRPCCount int
	MulticastRPCCount  int
}

func New(ctx context.Context, p *plan.ExecutionPlan, cfg Config) (*Client, error) {
	clients := map[string]*a2aclient.Client{}
	for _, agent := range p.Agents {
		baseURL := p.CardBaseURL(agent)
		card, err := agentcard.DefaultResolver.Resolve(ctx, baseURL)
		if err != nil {
			return nil, fmt.Errorf("resolve card for %s: %w", agent.ID, err)
		}
		cli, err := a2aclient.NewFromCard(
			ctx,
			card,
			a2agrpc.WithGRPCTransport(grpc.WithTransportCredentials(insecure.NewCredentials())),
		)
		if err != nil {
			return nil, fmt.Errorf("client for %s: %w", agent.ID, err)
		}
		clients[agent.ID] = cli
	}
	return &Client{
		clients:    clients,
		stats:      &Stats{},
		coordStats: metrics.NewCoordStats(cfg.RoundBudgetMS),
	}, nil
}

func (c *Client) CoordStats() *metrics.CoordStats { return c.coordStats }

func (c *Client) Stats() Stats {
	return *c.stats
}

func (c *Client) recordCoord(duration time.Duration, targets, responded, payloadBytes, messagesSent int) {
	if c.coordStats != nil {
		c.coordStats.RecordCoordOp(duration, targets, responded, payloadBytes, messagesSent)
	}
}

func (c *Client) ExecuteTask(ctx context.Context, agentID string, task plan.Task) (protocol.Response, error) {
	req := protocol.Request{
		Op:                   protocol.OpExecute,
		TaskID:               task.ID,
		CompletionTimeSec:    task.CompletionTimeSec,
		MaxCompletionTimeSec: task.MaxCompletionTimeSec,
		Output:               task.Output,
		InjectFailure:        task.InjectFailure,
	}
	return c.send(ctx, agentID, req, true)
}

func (c *Client) CancelTasks(ctx context.Context, agentIDs []string, taskIDs []string) error {
	start := time.Now()
	var perAgent []string
	responded := 0
	for i, agentID := range agentIDs {
		req := protocol.Request{Op: protocol.OpCancel, TaskIDs: taskIDs}
		callStart := time.Now()
		_, err := c.send(ctx, agentID, req, false)
		perAgent = append(perAgent, fmt.Sprintf("%s:%d", agentID, time.Since(callStart).Milliseconds()))
		if err != nil {
			c.recordCoord(time.Since(start), len(agentIDs), responded, len(taskIDs)*16, i)
			benchlog.RPC(benchlog.ImplA2A, protocol.OpCancel, "sequential", time.Since(start), false,
				fmt.Sprintf("agents=%d", len(agentIDs)),
				fmt.Sprintf("idx=%d", i+1),
				fmt.Sprintf("per_agent_ms=%s", strings.Join(perAgent, ",")),
				fmt.Sprintf("err=%v", err),
			)
			return err
		}
		responded++
		c.stats.SequentialRPCCount++
	}
	duration := time.Since(start)
	c.recordCoord(duration, len(agentIDs), responded, len(taskIDs)*16, len(agentIDs))
	benchlog.RPC(benchlog.ImplA2A, protocol.OpCancel, "sequential", duration, true,
		fmt.Sprintf("agents=%d", len(agentIDs)),
		fmt.Sprintf("per_agent_ms=%s", strings.Join(perAgent, ",")),
	)
	return nil
}

func (c *Client) PushContext(ctx context.Context, agentIDs []string, payload string) (time.Duration, error) {
	start := time.Now()
	req := protocol.Request{Op: protocol.OpContext, Payload: payload}
	var perAgent []string
	responded := 0
	for i, agentID := range agentIDs {
		callStart := time.Now()
		if _, err := c.send(ctx, agentID, req, false); err != nil {
			perAgent = append(perAgent, fmt.Sprintf("%s:%d", agentID, time.Since(callStart).Milliseconds()))
			c.recordCoord(time.Since(start), len(agentIDs), responded, len(payload), i)
			benchlog.RPC(benchlog.ImplA2A, protocol.OpContext, "sequential", time.Since(start), false,
				fmt.Sprintf("agents=%d", len(agentIDs)),
				fmt.Sprintf("idx=%d", i+1),
				fmt.Sprintf("per_agent_ms=%s", strings.Join(perAgent, ",")),
				fmt.Sprintf("err=%v", err),
			)
			return 0, err
		}
		perAgent = append(perAgent, fmt.Sprintf("%s:%d", agentID, time.Since(callStart).Milliseconds()))
		responded++
		c.stats.SequentialRPCCount++
	}
	duration := time.Since(start)
	c.recordCoord(duration, len(agentIDs), responded, len(payload), len(agentIDs))
	benchlog.RPC(benchlog.ImplA2A, protocol.OpContext, "sequential", duration, true,
		fmt.Sprintf("agents=%d", len(agentIDs)),
		fmt.Sprintf("per_agent_ms=%s", strings.Join(perAgent, ",")),
	)
	return duration, nil
}

func (c *Client) SyncPhase(ctx context.Context, agentIDs []string, phase string) (time.Duration, error) {
	start := time.Now()
	req := protocol.Request{Op: protocol.OpSync, Phase: phase}
	var perAgent []string
	responded := 0
	for i, agentID := range agentIDs {
		callStart := time.Now()
		if _, err := c.send(ctx, agentID, req, false); err != nil {
			perAgent = append(perAgent, fmt.Sprintf("%s:%d", agentID, time.Since(callStart).Milliseconds()))
			c.recordCoord(time.Since(start), len(agentIDs), responded, len(phase), i)
			benchlog.RPC(benchlog.ImplA2A, protocol.OpSync, "sequential", time.Since(start), false,
				fmt.Sprintf("agents=%d", len(agentIDs)),
				fmt.Sprintf("idx=%d", i+1),
				fmt.Sprintf("per_agent_ms=%s", strings.Join(perAgent, ",")),
				fmt.Sprintf("err=%v", err),
			)
			return 0, err
		}
		perAgent = append(perAgent, fmt.Sprintf("%s:%d", agentID, time.Since(callStart).Milliseconds()))
		responded++
		c.stats.SequentialRPCCount++
	}
	duration := time.Since(start)
	c.recordCoord(duration, len(agentIDs), responded, len(phase), len(agentIDs))
	benchlog.RPC(benchlog.ImplA2A, protocol.OpSync, "sequential", duration, true,
		fmt.Sprintf("agents=%d", len(agentIDs)),
		fmt.Sprintf("per_agent_ms=%s", strings.Join(perAgent, ",")),
	)
	return duration, nil
}

func (c *Client) NotifyFailure(ctx context.Context, agentIDs []string, failedTaskID string) error {
	start := time.Now()
	payload := "failure=" + failedTaskID
	req := protocol.Request{Op: protocol.OpContext, Payload: payload}
	var perAgent []string
	responded := 0
	for i, agentID := range agentIDs {
		callStart := time.Now()
		if _, err := c.send(ctx, agentID, req, false); err != nil {
			perAgent = append(perAgent, fmt.Sprintf("%s:%d", agentID, time.Since(callStart).Milliseconds()))
			c.recordCoord(time.Since(start), len(agentIDs), responded, len(payload), i)
			benchlog.RPC(benchlog.ImplA2A, protocol.OpContext, "sequential", time.Since(start), false,
				fmt.Sprintf("agents=%d", len(agentIDs)),
				fmt.Sprintf("failed_task=%s", failedTaskID),
				fmt.Sprintf("idx=%d", i+1),
				fmt.Sprintf("per_agent_ms=%s", strings.Join(perAgent, ",")),
				fmt.Sprintf("err=%v", err),
			)
			return err
		}
		perAgent = append(perAgent, fmt.Sprintf("%s:%d", agentID, time.Since(callStart).Milliseconds()))
		responded++
		c.stats.SequentialRPCCount++
	}
	duration := time.Since(start)
	c.recordCoord(duration, len(agentIDs), responded, len(payload), len(agentIDs))
	benchlog.RPC(benchlog.ImplA2A, protocol.OpContext, "sequential", duration, true,
		fmt.Sprintf("agents=%d", len(agentIDs)),
		fmt.Sprintf("failed_task=%s", failedTaskID),
		fmt.Sprintf("per_agent_ms=%s", strings.Join(perAgent, ",")),
	)
	return nil
}

func (c *Client) send(ctx context.Context, agentID string, req protocol.Request, execute bool) (protocol.Response, error) {
	cli, ok := c.clients[agentID]
	if !ok {
		return protocol.Response{}, fmt.Errorf("unknown agent %q", agentID)
	}
	text, err := protocol.EncodeRequest(req)
	if err != nil {
		return protocol.Response{}, err
	}
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(text))
	callStart := time.Now()
	result, err := cli.SendMessage(ctx, &a2a.SendMessageRequest{Message: msg})
	duration := time.Since(callStart)
	if err != nil {
		if execute {
			benchlog.RPC(benchlog.ImplA2A, req.Op, "p2p", duration, false,
				fmt.Sprintf("agent=%s", agentID),
				fmt.Sprintf("task=%s", req.TaskID),
				fmt.Sprintf("err=%v", err),
			)
		}
		return protocol.Response{}, err
	}
	if execute {
		c.stats.ExecuteRPCCount++
	}
	respText, err := responseText(result)
	if err != nil {
		if execute {
			benchlog.RPC(benchlog.ImplA2A, req.Op, "p2p", duration, false,
				fmt.Sprintf("agent=%s", agentID),
				fmt.Sprintf("task=%s", req.TaskID),
				fmt.Sprintf("err=%v", err),
			)
		}
		return protocol.Response{}, err
	}
	resp, decErr := protocol.DecodeResponse(respText)
	if execute {
		benchlog.RPC(benchlog.ImplA2A, req.Op, "p2p", duration, decErr == nil && resp.OK,
			fmt.Sprintf("agent=%s", agentID),
			fmt.Sprintf("task=%s", req.TaskID),
			fmt.Sprintf("resp_ok=%t", resp.OK),
		)
	}
	if decErr != nil {
		return protocol.Response{}, decErr
	}
	return resp, nil
}

func responseText(result a2a.SendMessageResult) (string, error) {
	switch v := result.(type) {
	case *a2a.Message:
		return firstText(v), nil
	case *a2a.Task:
		if v.Status.Message != nil {
			return firstText(v.Status.Message), nil
		}
		return "", fmt.Errorf("task response missing message")
	default:
		return "", fmt.Errorf("unexpected result type %T", result)
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
