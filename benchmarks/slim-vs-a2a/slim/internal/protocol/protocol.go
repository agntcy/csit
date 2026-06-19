// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package protocol

import "encoding/json"

const (
	OpExecute = "execute"
	OpCancel  = "cancel"
	OpContext = "context"
	OpSync    = "sync"
)

type Request struct {
	Op                   string   `json:"op"`
	TaskID               string   `json:"taskId,omitempty"`
	CompletionTimeSec    float64  `json:"completionTimeSec,omitempty"`
	MaxCompletionTimeSec float64  `json:"maxCompletionTimeSec,omitempty"`
	Output               string   `json:"output,omitempty"`
	InjectFailure        bool     `json:"injectFailure,omitempty"`
	TaskIDs              []string `json:"taskIds,omitempty"`
	Payload              string   `json:"payload,omitempty"`
	Phase                string   `json:"phase,omitempty"`
	FailedTaskID         string   `json:"failedTaskId,omitempty"`
	// TargetSlimNames lists org/group/app SLIM names that should handle a multicast.
	// Other group members acknowledge and ignore the request.
	TargetSlimNames []string `json:"targetSlimNames,omitempty"`
}

func (r Request) Targets(slimName string) bool {
	if len(r.TargetSlimNames) == 0 {
		return true
	}
	target := NormalizeSlimName(slimName)
	for _, name := range r.TargetSlimNames {
		if NormalizeSlimName(name) == target {
			return true
		}
	}
	return false
}

// NormalizeSlimName strips SlimRPC instance suffixes so agntcy/bench/foo and
// agntcy/bench/foo/<instance> compare equal.
func NormalizeSlimName(name string) string {
	parts := splitSlimName(name)
	if len(parts) >= 3 {
		return parts[0] + "/" + parts[1] + "/" + parts[2]
	}
	return name
}

func splitSlimName(name string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(name); i++ {
		if name[i] == '/' {
			if i > start {
				parts = append(parts, name[start:i])
			}
			start = i + 1
		}
	}
	if start < len(name) {
		parts = append(parts, name[start:])
	}
	return parts
}

type Response struct {
	OK         bool    `json:"ok"`
	TaskID     string  `json:"taskId,omitempty"`
	SlimName   string  `json:"slimName,omitempty"`
	Output     string  `json:"output,omitempty"`
	Error      string  `json:"error,omitempty"`
	ElapsedSec float64 `json:"elapsedSec,omitempty"`
}

func EncodeRequest(req Request) ([]byte, error) {
	return json.Marshal(req)
}

func DecodeRequest(data []byte) (Request, error) {
	var req Request
	err := json.Unmarshal(data, &req)
	return req, err
}

func EncodeResponse(resp Response) ([]byte, error) {
	return json.Marshal(resp)
}

func DecodeResponse(data []byte) (Response, error) {
	var resp Response
	err := json.Unmarshal(data, &resp)
	return resp, err
}
