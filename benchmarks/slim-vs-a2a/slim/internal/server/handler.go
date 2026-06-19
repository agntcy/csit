// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"

	slim "github.com/agntcy/slim-bindings-go"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/executor"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/slim/internal/protocol"
)

type Handler struct {
	Engine *executor.Engine
}

func (h *Handler) Handle(request []byte, _ *slim.Context) ([]byte, error) {
	req, err := protocol.DecodeRequest(request)
	if err != nil {
		return nil, err
	}
	resp := h.Engine.Handle(context.Background(), req)
	return protocol.EncodeResponse(resp)
}
