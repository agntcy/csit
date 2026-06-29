// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package protocol defines the A2A control/stream ops for the relay topology.
//
// Unlike SLIM's native group session, A2A has no peer multicast: findings must
// pass through the runner, which is the explicit relay hub. Two server-streaming
// legs carry findings:
//   - OpStreamFindings: the runner subscribes to each agent (agent streams its
//     own findings to the runner).
//   - OpStreamRelay: each agent subscribes to the runner (runner streams every
//     relayed finding back out to the agents).
//
// OpStart and OpSnapshot remain unary control calls driven by the runner.
package protocol

import (
	"encoding/json"
)

const (
	OpStart          = "start"
	OpSnapshot       = "snapshot"
	OpStreamFindings = "stream_findings"
	OpStreamRelay    = "stream_relay"
)

type Request struct {
	Op string `json:"op"`
	// AgentIndex identifies the subscriber on OpStreamRelay so the relay can
	// avoid echoing an agent's own findings back to it.
	AgentIndex int `json:"agentIndex,omitempty"`
}

type Response struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Body  string `json:"body,omitempty"`
}

func EncodeRequest(req Request) ([]byte, error) {
	return json.Marshal(req)
}

func DecodeRequest(text string) (Request, error) {
	var req Request
	err := json.Unmarshal([]byte(text), &req)
	return req, err
}

func EncodeResponse(resp Response) ([]byte, error) {
	return json.Marshal(resp)
}

func DecodeResponse(text string) (Response, error) {
	var resp Response
	err := json.Unmarshal([]byte(text), &resp)
	return resp, err
}
