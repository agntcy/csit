// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package protocol defines the message envelope exchanged over the native SLIM
// group session. There is no relay and no request/response RPC: the runner
// broadcasts a single start, agents broadcast findings to all peers, and agents
// push their snapshots to the runner the instant they converge.
package protocol

import (
	"encoding/json"

	"github.com/agntcy/csit/integrations/agntcy-slim/agent-consensus-test/internal/consensus"
)

const (
	// KindStart is broadcast once by the runner to define t0 and unblock agents.
	KindStart = "start"
	// KindFinding is broadcast by an agent to all group peers.
	KindFinding = "finding"
	// KindSnapshot is pushed by an agent directly to the runner on convergence.
	KindSnapshot = "snapshot"
)

// Envelope is the single JSON message type carried on the group session.
type Envelope struct {
	Kind       string `json:"kind"`
	AgentIndex int    `json:"agentIndex,omitempty"`
	// Epoch identifies which consensus attempt a KindStart begins.
	Epoch    int                      `json:"epoch,omitempty"`
	Finding  *consensus.Finding       `json:"finding,omitempty"`
	Snapshot *consensus.AgentSnapshot `json:"snapshot,omitempty"`
}

func Encode(e Envelope) ([]byte, error) {
	return json.Marshal(e)
}

func Decode(data []byte) (Envelope, error) {
	var e Envelope
	err := json.Unmarshal(data, &e)
	return e, err
}
