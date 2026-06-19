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
}

type Response struct {
	OK        bool    `json:"ok"`
	TaskID    string  `json:"taskId,omitempty"`
	Output    string  `json:"output,omitempty"`
	Error     string  `json:"error,omitempty"`
	ElapsedSec float64 `json:"elapsedSec,omitempty"`
}

func EncodeRequest(req Request) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func DecodeRequest(text string) (Request, error) {
	var req Request
	err := json.Unmarshal([]byte(text), &req)
	return req, err
}

func DecodeResponse(text string) (Response, error) {
	var resp Response
	err := json.Unmarshal([]byte(text), &resp)
	return resp, err
}
