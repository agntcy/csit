// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package agent implements a consensus agent that communicates over a native
// SLIM group session. Unlike the slimrpc-based design, there is NO application
// relay: every agent broadcasts its findings to all peers via PublishAndWait,
// and the SLIM dataplane fans them out. The runner is only a moderator that
// creates the session and a passive observer that consumes snapshots agents
// push to it the moment they converge.
package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"

	slim "github.com/agntcy/slim-bindings-go"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/benchlog"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/consensus"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/scenario"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/protocol"
)

const defaultSharedSecret = "demo-shared-secret-min-32-chars!!"

type Runtime struct {
	scenario   *scenario.ConsensusScenario
	agentIndex int
	slimName   string
	endpoint   string

	engine  *consensus.Engine
	app     *slim.App
	connID  uint64
	session *slim.Session

	mu        sync.Mutex
	runnerCtx *slim.MessageContext
	lastPush  time.Time
	// lastStartedEpoch dedupes repeated KindStart for the same epoch;
	// currentEpoch is the attempt currently running (used to drop stale peer
	// findings and to stop a superseded epoch loop).
	lastStartedEpoch int
	currentEpoch     int

	pubMu sync.Mutex // serializes publishes on the session

	startedAt time.Time
}

func NewRuntime(s *scenario.ConsensusScenario, agentIndex int, slimName, endpoint string) *Runtime {
	return &Runtime{
		scenario:         s,
		agentIndex:       agentIndex,
		slimName:         slimName,
		endpoint:         endpoint,
		engine:           consensus.NewEngine(s.Spec, agentIndex),
		lastStartedEpoch: -1,
		currentEpoch:     -1,
	}
}

// Setup creates the SLIM app, connects to the dataplane, and subscribes to the
// agent's own name so the moderator can route invitations to it.
func (r *Runtime) Setup() error {
	slim.InitializeWithDefaults()
	service := slim.GetGlobalService()

	name, err := slim.NameFromString(r.slimName)
	if err != nil {
		return err
	}

	app, err := service.CreateAppWithSecret(name, defaultSharedSecret)
	if err != nil {
		return err
	}
	r.app = app

	connID, err := service.Connect(slim.NewInsecureClientConfig(r.endpoint))
	if err != nil {
		app.Destroy()
		return err
	}
	r.connID = connID
	if err := app.Subscribe(name, &connID); err != nil {
		app.Destroy()
		return err
	}
	return nil
}

// Join blocks until the moderator invites this agent into the group session.
func (r *Runtime) Join(timeout time.Duration) error {
	session, err := r.app.ListenForSessionAsync(&timeout)
	if err != nil {
		return fmt.Errorf("listen for session: %w", err)
	}
	r.session = session
	return nil
}

// Run is the single receive loop on the group session. It dispatches by
// envelope kind: start launches the think loop, finding is applied locally.
// It blocks until the session ends (the runner kills the process at shutdown).
func (r *Runtime) Run() error {
	for {
		timeout := time.Second
		msg, err := r.session.GetMessageAsync(&timeout)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return err
		}
		env, derr := protocol.Decode(msg.Payload)
		if derr != nil {
			continue
		}
		switch env.Kind {
		case protocol.KindStart:
			ctx := msg.Context
			r.startEpoch(env.Epoch, &ctx)
		case protocol.KindFinding:
			if env.Finding == nil || env.Finding.AgentIndex == r.agentIndex {
				continue
			}
			// Drop stragglers from a different (usually previous) epoch.
			r.mu.Lock()
			cur := r.currentEpoch
			r.mu.Unlock()
			if env.Finding.Epoch != cur {
				continue
			}
			r.applyFinding(*env.Finding)
			// Keep the runner's view fresh if we already converged.
			r.maybePushSnapshot(false)
		}
	}
}

// startEpoch resets the engine for a fresh attempt and launches the epoch loop.
// Repeated starts for the same (or older) epoch are ignored so a duplicated
// broadcast cannot double-run an attempt.
func (r *Runtime) startEpoch(epoch int, ctx *slim.MessageContext) {
	r.mu.Lock()
	if epoch <= r.lastStartedEpoch {
		r.mu.Unlock()
		return
	}
	r.lastStartedEpoch = epoch
	r.currentEpoch = epoch
	r.runnerCtx = ctx
	r.mu.Unlock()

	r.engine.Reset(epoch)
	r.startedAt = time.Now()
	benchlog.SetRunStart(r.startedAt)
	go r.runEpoch(epoch)
}

func (r *Runtime) runEpoch(epoch int) {
	spec := r.scenario.Spec
	think := time.Duration(spec.ThinkTimeMs) * time.Millisecond
	emitGap := time.Duration(spec.FindingEmitDelayMs) * time.Millisecond

	for round := 0; round < spec.MaxRounds; round++ {
		// Stop early if a newer epoch has superseded this one.
		r.mu.Lock()
		superseded := r.currentEpoch != epoch
		r.mu.Unlock()
		if superseded {
			return
		}

		time.Sleep(think)

		finding, emit := r.engine.Think()
		if emit && finding != nil {
			if err := r.publishFinding(*finding); err != nil {
				benchlog.Finding(benchlog.ImplSLIM, "publish_error", r.agentIndex, finding.FindingID,
					fmt.Sprintf("err=%v", err))
			} else {
				benchlog.Finding(benchlog.ImplSLIM, "published", r.agentIndex, finding.FindingID)
			}
			time.Sleep(emitGap)
		}

		if r.engine.HasLocalConsensus() {
			r.maybePushSnapshot(true)
			return
		}
	}
	// Reached max rounds without consensus: still report final state.
	r.maybePushSnapshot(true)
}

// publishFinding broadcasts a finding to every group peer. The SLIM dataplane
// performs the fan-out; there is no relay.
func (r *Runtime) publishFinding(f consensus.Finding) error {
	payload, err := protocol.Encode(protocol.Envelope{
		Kind:       protocol.KindFinding,
		AgentIndex: r.agentIndex,
		Finding:    &f,
	})
	if err != nil {
		return err
	}
	r.pubMu.Lock()
	defer r.pubMu.Unlock()
	return r.session.PublishAndWaitAsync(payload, nil, nil)
}

// maybePushSnapshot sends the current snapshot directly to the runner (targeted,
// off the peer broadcast path). force=true always sends; otherwise it is
// throttled so post-convergence finding updates do not spam the runner.
func (r *Runtime) maybePushSnapshot(force bool) {
	r.mu.Lock()
	ctx := r.runnerCtx
	if ctx == nil {
		r.mu.Unlock()
		return
	}
	if !force && time.Since(r.lastPush) < 50*time.Millisecond {
		r.mu.Unlock()
		return
	}
	r.lastPush = time.Now()
	r.mu.Unlock()

	snap := r.engine.Snapshot()
	payload, err := protocol.Encode(protocol.Envelope{
		Kind:       protocol.KindSnapshot,
		AgentIndex: r.agentIndex,
		Snapshot:   &snap,
	})
	if err != nil {
		return
	}
	r.pubMu.Lock()
	defer r.pubMu.Unlock()
	_ = r.session.PublishToAndWaitAsync(*ctx, payload, nil, nil)
}

func (r *Runtime) applyFinding(f consensus.Finding) {
	recvNs := time.Now().UnixNano()
	r.engine.ApplyFinding(f)
	if f.EmittedAt > 0 {
		r.engine.RecordPropagation(f.EmittedAt, recvNs)
	}
	benchlog.Finding(benchlog.ImplSLIM, "received", r.agentIndex, f.FindingID,
		fmt.Sprintf("from=%d", f.AgentIndex))
}

func (r *Runtime) Close() {
	if r.session != nil && r.app != nil {
		_ = r.app.DeleteSessionAndWaitAsync(r.session)
	}
	if r.app != nil {
		r.app.Destroy()
	}
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timed out") || strings.Contains(msg, "timeout")
}
