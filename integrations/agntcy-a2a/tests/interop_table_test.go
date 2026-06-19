// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// Three nested DescribeTableSubtrees produce the full (client, server, transport)
// interop matrix with minimal repetition:
//
//	A2A interoperability
//	  go (client)                              [CLIENT:go]
//	    server go                              [SERVER:go]
//	      transport JSON-RPC                   [TRANSPORT:jsonrpc]
//	      transport REST                       [TRANSPORT:rest]
//	      transport gRPC                       [TRANSPORT:grpc]
//	    server rust                            [SERVER:rust]
//	      ...
//	    server dotnet                          [SERVER:dotnet]
//	      transport JSON-RPC
//	      transport REST
//	      transport gRPC                       ← skipped: dotnet does not support gRPC
//	    ...
//	  rust (client)                            [CLIENT:rust]
//	    ...
//	  dotnet (client)                          [CLIENT:dotnet]
//	    ...
//	      transport gRPC                       ← skipped: dotnet does not support gRPC
//
// Each spec inherits all three category labels from its ancestor entries, enabling
// set-based label filtering:
//
//	--ginkgo.label-filter='CLIENT: consistsOf {go}'
//	--ginkgo.label-filter='CLIENT: containsAny {rust, go} && SERVER: consistsOf {dotnet}'
//	--ginkgo.label-filter='TRANSPORT: containsAny {grpc}'
//
// When either the client or the server SDK does not support gRPC, the gRPC
// BeforeAll calls Skip() so the specs appear as skipped rather than absent.
//
// All fixtures use their toolchain's run command (go run / cargo run / uv run /
// dotnet run) so no pre-build step or asset cache is needed.

import (
	. "github.com/onsi/ginkgo/v2"
)

// ── interoperability matrix ───────────────────────────────────────────────────

var _ = DescribeTableSubtree(
	"A2A interoperability",
	func(newClient newClientFn, clientGRPC bool) {
		DescribeTableSubtree("server",
			func(startServer serverStarter, serverPrefix string, serverGRPC bool) {
				// ContinueOnFailure is not set here: transport entries are nested
				// inside client entries (which are the outermost Ordered containers)
				// and inherit ContinueOnFailure behaviour from them.
				DescribeTableSubtree("transport",
					func(protocol transportProtocol) {
						registerBehaviors(newClient, startServer, protocol, serverPrefix, clientGRPC, serverGRPC)
					},
					Entry("JSON-RPC", Ordered, Label("TRANSPORT:jsonrpc"), transportJSONRPC),
					Entry("REST",     Ordered, Label("TRANSPORT:rest"),    transportREST),
					Entry("gRPC",     Ordered, Label("TRANSPORT:grpc"),    transportGRPC),
				)
			},
			// ── server entries ────────────────────────────────────────────────
			// ContinueOnFailure is inherited from the client entries (outermost Ordered).
			Entry("go",     Ordered, Label("SERVER:go"),     startGoFixture,     "go",     true),
			Entry("rust",   Ordered, Label("SERVER:rust"),   startRustFixture,   "rust",   true),
			Entry("python", Ordered, Label("SERVER:python"), startPythonFixture, "python", true),
			Entry("dotnet", Ordered, Label("SERVER:dotnet"), startDotNetFixture, "dotnet", false),
		)
	},
	// ── client entries ────────────────────────────────────────────────────────
	Entry("go",     Ordered, ContinueOnFailure, Label("CLIENT:go"),     newGoProbeClient,     true),
	Entry("rust",   Ordered, ContinueOnFailure, Label("CLIENT:rust"),   newRustProbeClient,   true),
	Entry("python", Ordered, ContinueOnFailure, Label("CLIENT:python"), newPythonProbeClient, true),
	Entry("dotnet", Ordered, ContinueOnFailure, Label("CLIENT:dotnet"), newDotNetProbeClient, false),
)
