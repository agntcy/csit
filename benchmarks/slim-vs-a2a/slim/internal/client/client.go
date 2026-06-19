// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	slim "github.com/agntcy/slim-bindings-go"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/plan"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/protocol"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/slimrpc"
)

const defaultSharedSecret = "demo-shared-secret-min-32-chars!!"

type CoordMode string

const (
	CoordModeMulticast CoordMode = "multicast"
	CoordModeUnicast   CoordMode = "unicast"
)

type Config struct {
	RoundBudgetMS int64
	CoordMode     CoordMode
}

type Client struct {
	mu            sync.Mutex
	app           *slim.App
	connID        uint64
	agentNames    map[string]*slim.Name
	slimNameByID  map[string]string
	p2pChannels   map[string]*slim.Channel
	groupChannels map[string]*slim.Channel
	groupLocks    map[string]*sync.Mutex
	coordMode     CoordMode
	coordStats    *metrics.CoordStats
	stats         Stats
	implLabel     string
}

type Stats struct {
	ExecuteRPCCount    int
	SequentialRPCCount int
	MulticastRPCCount  int
}

func New(endpoint string, p *plan.ExecutionPlan, cfg Config) (*Client, error) {
	if cfg.CoordMode == "" {
		cfg.CoordMode = CoordModeMulticast
	}
	impl := benchlog.ImplSLIM
	if cfg.CoordMode == CoordModeUnicast {
		impl = "slim-unicast"
	}

	slim.InitializeWithDefaults()
	service := slim.GetGlobalService()

	runnerName := slim.NewName("agntcy", "bench", "runner")
	app, err := service.CreateAppWithSecret(runnerName, defaultSharedSecret)
	if err != nil {
		return nil, err
	}

	connID, err := service.Connect(slim.NewInsecureClientConfig(endpoint))
	if err != nil {
		app.Destroy()
		return nil, err
	}
	if err := app.Subscribe(runnerName, &connID); err != nil {
		app.Destroy()
		return nil, err
	}

	agentNames := map[string]*slim.Name{}
	slimNameByID := map[string]string{}
	for _, agent := range p.Agents {
		name, err := nameFromString(agent.SlimName)
		if err != nil {
			app.Destroy()
			return nil, err
		}
		agentNames[agent.ID] = name
		slimNameByID[agent.ID] = agent.SlimName
		if err := app.SetRoute(name, connID); err != nil {
			app.Destroy()
			return nil, fmt.Errorf("set route %s: %w", agent.SlimName, err)
		}
	}

	return &Client{
		app:           app,
		connID:        connID,
		agentNames:    agentNames,
		slimNameByID:  slimNameByID,
		p2pChannels:   make(map[string]*slim.Channel),
		groupChannels: make(map[string]*slim.Channel),
		groupLocks:    make(map[string]*sync.Mutex),
		coordMode:     cfg.CoordMode,
		coordStats:    metrics.NewCoordStats(cfg.RoundBudgetMS),
		implLabel:     impl,
	}, nil
}

func (c *Client) CoordStats() *metrics.CoordStats { return c.coordStats }

func (c *Client) Implementation() string { return c.implLabel }

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, ch := range c.groupChannels {
		_ = ch.Close(nil)
		delete(c.groupChannels, key)
	}
	for id, ch := range c.p2pChannels {
		_ = ch.Close(nil)
		delete(c.p2pChannels, id)
	}
	if c.app != nil {
		c.app.Destroy()
		c.app = nil
	}
}

func (c *Client) Stats() Stats { return c.stats }

func (c *Client) SetupGroup(context.Context, *plan.ExecutionPlan) error { return nil }

func (c *Client) ExecuteTask(ctx context.Context, agentID string, task plan.Task) (protocol.Response, error) {
	req := protocol.Request{
		Op:                   protocol.OpExecute,
		TaskID:               task.ID,
		CompletionTimeSec:    task.CompletionTimeSec,
		MaxCompletionTimeSec: task.MaxCompletionTimeSec,
		Output:               task.Output,
		InjectFailure:        task.InjectFailure,
	}
	payload, err := protocol.EncodeRequest(req)
	if err != nil {
		return protocol.Response{}, err
	}
	start := time.Now()
	reply, err := c.callUnary(ctx, agentID, payload, 30*time.Second)
	duration := time.Since(start)
	if err != nil {
		benchlog.RPC(c.implLabel, protocol.OpExecute, "p2p", duration, false,
			fmt.Sprintf("agent=%s", agentID),
			fmt.Sprintf("task=%s", task.ID),
			fmt.Sprintf("err=%v", err),
		)
		return protocol.Response{}, err
	}
	c.stats.ExecuteRPCCount++
	resp, decErr := protocol.DecodeResponse(reply)
	benchlog.RPC(c.implLabel, protocol.OpExecute, "p2p", duration, decErr == nil && resp.OK,
		fmt.Sprintf("agent=%s", agentID),
		fmt.Sprintf("task=%s", task.ID),
		fmt.Sprintf("resp_ok=%t", resp.OK),
	)
	if decErr != nil {
		return protocol.Response{}, decErr
	}
	return resp, nil
}

func (c *Client) CancelTasks(ctx context.Context, agentIDs []string, taskIDs []string) error {
	return c.coordOp(ctx, agentIDs, protocol.Request{
		Op:              protocol.OpCancel,
		TaskIDs:         taskIDs,
		TargetSlimNames: c.slimNamesFor(agentIDs),
	})
}

func (c *Client) PushContext(ctx context.Context, agentIDs []string, payload string) (time.Duration, error) {
	start := time.Now()
	err := c.coordOp(ctx, agentIDs, protocol.Request{
		Op:              protocol.OpContext,
		Payload:         payload,
		TargetSlimNames: c.slimNamesFor(agentIDs),
	})
	return time.Since(start), err
}

func (c *Client) SyncPhase(ctx context.Context, agentIDs []string, phase string) (time.Duration, error) {
	start := time.Now()
	err := c.coordOp(ctx, agentIDs, protocol.Request{
		Op:              protocol.OpSync,
		Phase:           phase,
		TargetSlimNames: c.slimNamesFor(agentIDs),
	})
	return time.Since(start), err
}

func (c *Client) NotifyFailure(ctx context.Context, agentIDs []string, failedTaskID string) error {
	return c.coordOp(ctx, agentIDs, protocol.Request{
		Op:              protocol.OpContext,
		Payload:         "failure=" + failedTaskID,
		TargetSlimNames: c.slimNamesFor(agentIDs),
	})
}

func (c *Client) coordOp(ctx context.Context, agentIDs []string, req protocol.Request) error {
	if len(agentIDs) == 0 {
		return nil
	}
	payload, err := protocol.EncodeRequest(req)
	if err != nil {
		return err
	}
	if c.coordMode == CoordModeUnicast {
		return c.unicastFanOut(ctx, agentIDs, req, payload)
	}
	return c.multicast(ctx, agentIDs, req, payload)
}

func (c *Client) unicastFanOut(ctx context.Context, agentIDs []string, req protocol.Request, payload []byte) error {
	start := time.Now()
	var firstErr error
	responded := 0
	for i, agentID := range agentIDs {
		callStart := time.Now()
		_, err := c.callUnary(ctx, agentID, payload, 10*time.Second)
		if err != nil && firstErr == nil {
			firstErr = err
		} else if err == nil {
			responded++
		}
		c.stats.SequentialRPCCount++
		if err != nil {
			benchlog.RPC(c.implLabel, req.Op, "sequential", time.Since(start), false,
				fmt.Sprintf("targets=%d", len(agentIDs)),
				fmt.Sprintf("idx=%d", i+1),
				fmt.Sprintf("err=%v", err),
			)
			c.recordCoord(time.Since(start), len(agentIDs), responded, len(payload), len(agentIDs))
			return err
		}
		_ = callStart
	}
	duration := time.Since(start)
	c.recordCoord(duration, len(agentIDs), responded, len(payload), len(agentIDs))
	benchlog.RPC(c.implLabel, req.Op, "sequential", duration, true,
		fmt.Sprintf("targets=%d", len(agentIDs)),
		fmt.Sprintf("responded=%d", responded),
	)
	return firstErr
}

func (c *Client) recordCoord(duration time.Duration, targets, responded, payloadBytes, messagesSent int) {
	if c.coordStats != nil {
		c.coordStats.RecordCoordOp(duration, targets, responded, payloadBytes, messagesSent)
	}
}

func (c *Client) slimNamesFor(agentIDs []string) []string {
	names := make([]string, 0, len(agentIDs))
	for _, id := range agentIDs {
		if name, ok := c.slimNameByID[id]; ok {
			names = append(names, name)
		}
	}
	return names
}

func subsetGroupKey(agentIDs []string) string {
	ids := append([]string(nil), agentIDs...)
	slices.Sort(ids)
	return strings.Join(ids, "|")
}

func (c *Client) groupLock(key string) *sync.Mutex {
	c.mu.Lock()
	defer c.mu.Unlock()
	if l, ok := c.groupLocks[key]; ok {
		return l
	}
	l := &sync.Mutex{}
	c.groupLocks[key] = l
	return l
}

func (c *Client) subsetGroup(agentIDs []string) (*slim.Channel, error) {
	key := subsetGroupKey(agentIDs)

	c.mu.Lock()
	if ch, ok := c.groupChannels[key]; ok {
		c.mu.Unlock()
		return ch, nil
	}
	c.mu.Unlock()

	members := make([]*slim.Name, 0, len(agentIDs))
	for _, id := range agentIDs {
		name, ok := c.agentNames[id]
		if !ok {
			return nil, fmt.Errorf("unknown agent %q", id)
		}
		members = append(members, name)
	}

	channel, err := slim.ChannelNewGroupWithConnection(c.app, members, &c.connID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if existing, ok := c.groupChannels[key]; ok {
		c.mu.Unlock()
		_ = channel.Close(nil)
		return existing, nil
	}
	c.groupChannels[key] = channel
	c.mu.Unlock()
	return channel, nil
}

func (c *Client) multicast(ctx context.Context, agentIDs []string, req protocol.Request, payload []byte) error {
	key := subsetGroupKey(agentIDs)
	lock := c.groupLock(key)
	lock.Lock()
	defer lock.Unlock()

	group, err := c.subsetGroup(agentIDs)
	if err != nil {
		return err
	}

	timeout := 10 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			return context.DeadlineExceeded
		}
	}

	start := time.Now()
	targets := c.slimNamesFor(agentIDs)
	reader, err := group.CallMulticastUnary(slimrpc.ServiceName, slimrpc.MethodHandle, payload, &timeout, nil)
	if err != nil {
		benchlog.RPC(c.implLabel, req.Op, "multicast", time.Since(start), false,
			fmt.Sprintf("targets=%d", len(targets)),
			fmt.Sprintf("target_names=%s", strings.Join(targets, ",")),
			fmt.Sprintf("err=%v", err),
		)
		c.recordCoord(time.Since(start), len(targets), 0, len(payload), 1)
		return err
	}
	defer reader.Destroy()

	c.stats.MulticastRPCCount++
	responded, drainErr := drainMulticast(reader, targets)
	duration := time.Since(start)
	c.recordCoord(duration, len(targets), responded, len(payload), 1)
	kv := []string{
		fmt.Sprintf("targets=%d", len(targets)),
		fmt.Sprintf("responded=%d", responded),
		fmt.Sprintf("target_names=%s", strings.Join(targets, ",")),
	}
	if drainErr != nil {
		kv = append(kv, fmt.Sprintf("err=%v", drainErr))
	}
	benchlog.RPC(c.implLabel, req.Op, "multicast", duration, drainErr == nil, kv...)
	return drainErr
}

func drainMulticast(reader *slim.MulticastResponseReader, targetedSlimNames []string) (int, error) {
	targetSet := make(map[string]struct{}, len(targetedSlimNames))
	for _, name := range targetedSlimNames {
		targetSet[protocol.NormalizeSlimName(name)] = struct{}{}
	}

	var firstErr error
	responded := map[string]struct{}{}

	for {
		switch msg := reader.Next().(type) {
		case slim.MulticastStreamMessageEnd:
			for name := range targetSet {
				if _, ok := responded[name]; !ok && firstErr == nil {
					firstErr = fmt.Errorf("missing multicast response from %q", name)
				}
			}
			return len(responded), firstErr
		case slim.MulticastStreamMessageError:
			if firstErr == nil && msg.Error != nil {
				firstErr = msg.Error.AsError()
			}
		case slim.MulticastStreamMessageData:
			resp, err := protocol.DecodeResponse(msg.Item.Message)
			if err != nil && firstErr == nil {
				firstErr = err
				continue
			}
			slimName := resp.SlimName
			if slimName == "" && msg.Item.Context.Source != nil {
				slimName = msg.Item.Context.Source.String()
			}
			slimName = protocol.NormalizeSlimName(slimName)
			if slimName == "" {
				continue
			}
			responded[slimName] = struct{}{}
			if _, wanted := targetSet[slimName]; wanted && !resp.OK && firstErr == nil {
				firstErr = fmt.Errorf("%s: %s", slimName, resp.Error)
			}
		}
	}
}

func (c *Client) callUnary(ctx context.Context, agentID string, payload []byte, timeout time.Duration) ([]byte, error) {
	channel, err := c.p2pChannel(agentID)
	if err != nil {
		return nil, err
	}

	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	return channel.CallUnary(slimrpc.ServiceName, slimrpc.MethodHandle, payload, &timeout, nil)
}

func (c *Client) p2pChannel(agentID string) (*slim.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ch, ok := c.p2pChannels[agentID]; ok {
		return ch, nil
	}

	dest, ok := c.agentNames[agentID]
	if !ok {
		return nil, fmt.Errorf("unknown agent %q", agentID)
	}

	ch := slim.ChannelNewWithConnection(c.app, dest, &c.connID)
	c.p2pChannels[agentID] = ch
	return ch, nil
}

func nameFromString(value string) (*slim.Name, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid name format: %s", value)
	}
	return slim.NewName(parts[0], parts[1], parts[2]), nil
}
