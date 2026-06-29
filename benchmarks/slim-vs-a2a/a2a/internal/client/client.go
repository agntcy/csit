// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	a2agrpc "github.com/a2aproject/a2a-go/v2/a2agrpc/v1"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/a2a/internal/protocol"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/consensus"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/scenario"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	clients map[string]*a2aclient.Client
}

func New(ctx context.Context, s *scenario.ConsensusScenario) (*Client, error) {
	clients := map[string]*a2aclient.Client{}
	for _, agent := range s.Agents {
		baseURL := s.CardBaseURL(agent)
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
	return &Client{clients: clients}, nil
}

func (c *Client) StartAll(ctx context.Context, agentIDs []string) error {
	for _, id := range agentIDs {
		if err := c.send(ctx, id, protocol.Request{Op: protocol.OpStart}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Snapshot(ctx context.Context, agentID string) (consensus.AgentSnapshot, error) {
	resp, err := c.sendWithResponse(ctx, agentID, protocol.Request{Op: protocol.OpSnapshot})
	if err != nil {
		return consensus.AgentSnapshot{}, err
	}
	var snap consensus.AgentSnapshot
	if err := json.Unmarshal([]byte(resp.Body), &snap); err != nil {
		return consensus.AgentSnapshot{}, err
	}
	return snap, nil
}

// SubscribeFindings opens the OpStreamFindings server stream to an agent and
// invokes onFinding for every finding the agent produces. It blocks until the
// stream ends or ctx is cancelled, so callers run it in a goroutine.
func (c *Client) SubscribeFindings(ctx context.Context, agentID string, onFinding func(consensus.Finding)) error {
	cli, ok := c.clients[agentID]
	if !ok {
		return fmt.Errorf("unknown agent %q", agentID)
	}
	reqText, err := json.Marshal(protocol.Request{Op: protocol.OpStreamFindings})
	if err != nil {
		return err
	}
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(string(reqText)))
	for ev, streamErr := range cli.SendStreamingMessage(ctx, &a2a.SendMessageRequest{Message: msg}) {
		if streamErr != nil {
			return streamErr
		}
		f, ok := findingFromEvent(ev)
		if !ok {
			continue
		}
		onFinding(f)
	}
	return nil
}

func (c *Client) send(ctx context.Context, agentID string, req protocol.Request) error {
	_, err := c.sendWithResponse(ctx, agentID, req)
	return err
}

func (c *Client) sendWithResponse(ctx context.Context, agentID string, req protocol.Request) (protocol.Response, error) {
	cli, ok := c.clients[agentID]
	if !ok {
		return protocol.Response{}, fmt.Errorf("unknown agent %q", agentID)
	}
	text, err := json.Marshal(req)
	if err != nil {
		return protocol.Response{}, err
	}
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(string(text)))
	result, err := cli.SendMessage(ctx, &a2a.SendMessageRequest{Message: msg})
	if err != nil {
		return protocol.Response{}, err
	}
	respText, err := responseText(result)
	if err != nil {
		return protocol.Response{}, err
	}
	return protocol.DecodeResponse(respText)
}

func findingFromEvent(ev a2a.Event) (consensus.Finding, bool) {
	su, ok := ev.(*a2a.TaskStatusUpdateEvent)
	if !ok || su.Status.Message == nil {
		return consensus.Finding{}, false
	}
	text := firstText(su.Status.Message)
	if text == "" {
		return consensus.Finding{}, false
	}
	f, err := consensus.DecodeFinding([]byte(text))
	if err != nil {
		return consensus.Finding{}, false
	}
	return f, true
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
