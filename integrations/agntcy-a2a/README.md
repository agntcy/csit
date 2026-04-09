# A2A Interoperability CSIT

This component hosts cross-SDK A2A interoperability checks.

The current coverage includes Rust and Go across the released JSON-RPC, HTTP+JSON, and gRPC bindings plus a Rust/.NET slice across JSON-RPC and HTTP+JSON.

All 12 Rust/Go client-server legs in the current core lifecycle matrix are green across JSON-RPC, HTTP+JSON, and gRPC.

The Go and Rust fixtures now expose push-config CRUD. CSIT validates that path from both clients against both server targets across JSON-RPC, HTTP+JSON, and gRPC.

The Rust/.NET slice now exists alongside the Rust/Go suite. It reuses the existing Rust fixture and Rust probe, adds CSIT-owned .NET fixture and probe binaries, and covers 8 legs: 4 JSON-RPC legs (`dotnet-dotnet`, `dotnet-rust`, `rust-dotnet`, `rust-rust-dotnet`) plus the same 4 legs over HTTP+JSON. Each leg is split into shared behavior specs for unary and streaming requests, task lifecycle APIs, push-config semantics, and scenario parity. Push-config is covered in this slice as well, but only the Rust-server legs are expected to support push-config CRUD; the .NET-server legs are expected to return the current unsupported error. This slice does not cover gRPC. The default `task test` and `task integrations:a2a:test` entrypoints now run both the established Rust/Go matrix and this Rust/.NET slice.

Across the matrix, the scenarios validate the same core interoperability behavior:

- unary and streaming `SendMessage`
- message-only responses without task creation
- lifecycle methods across `GetTask`, `ListTasks`, and `CancelTask`
- failed-task responses
- multi-turn `TASK_STATE_INPUT_REQUIRED` continuations
- long-running completion after non-blocking sends
- artifact-rich streaming updates
- structured text + data + URL artifact payloads
- extended-agent-card discovery and skill metadata
- negative-path error semantics for missing and non-cancelable tasks
- successful push-config CRUD on both server paths across all three transports
- preservation of a mixed text plus structured-data request payload and message metadata through task history

The fixtures are intentionally small and deterministic so the suite can run the same way locally and in CI without depending on sibling SDK checkouts.

Each leg keeps its existing suite, transport, and pair labels for Task targets, and the shared behavior slices add dedicated Ginkgo labels (`behavior-core`, `behavior-unary-streaming`, `behavior-lifecycle`, `behavior-push-config`, and `behavior-parity`) so full legs and cross-leg behavior groups can be filtered independently.

The gRPC legs follow the same agent-card discovery path as the other transports: each fixture serves `/.well-known/agent-card.json` over HTTP and advertises a separate gRPC transport endpoint from that card.

The Rust/.NET slice currently requires a local `dotnet` CLI for the .NET 8 SDK because the CSIT harness builds the fixture and probe from published `A2A` and `A2A.AspNetCore` NuGet packages at test time.

## Matrix

Legend: ✅ covered by passing automated CSIT, ❌ not currently covered by this suite.

### SDK Pair Coverage

| SDK pair | JSON-RPC | HTTP+JSON | gRPC |
| --- | --- | --- | --- |
| Rust/Go | ✅ | ✅ | ✅ |
| Rust/.NET | ✅ | ✅ | ❌ |

### Rust/Go Leg Coverage

| Client -> Server | JSON-RPC | HTTP+JSON | gRPC |
| --- | --- | --- | --- |
| Go -> Go | ✅ | ✅ | ✅ |
| Go -> Rust | ✅ | ✅ | ✅ |
| Rust -> Go | ✅ | ✅ | ✅ |
| Rust -> Rust | ✅ | ✅ | ✅ |

### Rust/.NET Leg Coverage

| Client -> Server | JSON-RPC | HTTP+JSON | gRPC |
| --- | --- | --- | --- |
| .NET -> .NET | ✅ | ✅ | ❌ |
| .NET -> Rust | ✅ | ✅ | ❌ |
| Rust -> .NET | ✅ | ✅ | ❌ |
| Rust -> Rust (Rust/.NET slice) | ✅ | ✅ | ❌ |

### Behavior Coverage

| Behavior | Rust/Go | Rust/.NET |
| --- | --- | --- |
| Unary `SendMessage` | ✅ | ✅ |
| Streaming `SendMessage` | ✅ | ✅ |
| Message-only response path | ✅ | ✅ |
| `GetTask`, `ListTasks`, and `CancelTask` lifecycle | ✅ | ✅ |
| Failed-task response path | ✅ | ✅ |
| Multi-turn input-required continuation | ✅ | ✅ |
| Long-running completion after non-blocking send | ✅ | ✅ |
| Artifact-rich streaming updates | ✅ | ✅ |
| Structured text + data + URL artifact payloads | ✅ | ✅ |
| Extended-agent-card discovery | ✅ | ✅ |
| Extended-card security-scheme metadata | ✅ | ✅ |
| Missing-task and non-cancelable-task errors | ✅ | ✅ |
| Mixed text + structured-data payload preserved through task history | ✅ | ✅ |
| Push-config CRUD against Rust-server targets | ✅ | ✅ |
| Unsupported push-config response against non-Rust server targets | ✅ | ✅ |

For Rust/.NET, the uncovered cells are the current gRPC gap. For push-config, a ✅ means the suite verifies the expected behavior for that leg: CRUD succeeds on Rust-server targets and the current unsupported error is returned on Go-server or .NET-server targets.

For the .NET-owned fixture specifically, the public and extended cards currently omit `securitySchemes` because the released Rust resolver used by this CSIT slice does not yet accept the .NET SDK's null-filled union encoding for that field. The overall suites still validate bearer-token security metadata against the Rust and Go fixtures.

## Running the Suite

From `integrations/agntcy-a2a/`:

```sh
task test
task test:rust-go
task test:rust-go:jsonrpc
task test:rust-go:rest
task test:rust-go:grpc
task test:rust-go:jsonrpc:go-go
task test:rust-go:jsonrpc:go-rust
task test:rust-go:jsonrpc:rust-go
task test:rust-go:jsonrpc:rust-rust
task test:rust-go:rest:go-go
task test:rust-go:rest:go-rust
task test:rust-go:rest:rust-go
task test:rust-go:rest:rust-rust
task test:rust-go:grpc:go-go
task test:rust-go:grpc:go-rust
task test:rust-go:grpc:rust-go
task test:rust-go:grpc:rust-rust
```

`task test` now runs the full `task test:rust-go` transport matrix plus `task test:rust-dotnet`.

The Rust/.NET slice is also exposed separately so it can be iterated independently:

```sh
task test:rust-dotnet
task test:rust-dotnet:jsonrpc
task test:rust-dotnet:jsonrpc:dotnet-dotnet
task test:rust-dotnet:jsonrpc:dotnet-rust
task test:rust-dotnet:jsonrpc:rust-dotnet
task test:rust-dotnet:jsonrpc:rust-rust-dotnet
task test:rust-dotnet:rest
task test:rust-dotnet:rest:dotnet-dotnet
task test:rust-dotnet:rest:dotnet-rust
task test:rust-dotnet:rest:rust-dotnet
task test:rust-dotnet:rest:rust-rust-dotnet
```

Behavior-focused runs are exposed as first-class Task targets as well:

```sh
task test:behavior:core
task test:behavior:unary-streaming
task test:behavior:lifecycle
task test:behavior:push-config
task test:behavior:parity
task test:rust-go:behavior:unary-streaming
task test:rust-go:behavior:lifecycle
task test:rust-go:behavior:push-config
task test:rust-dotnet:behavior:unary-streaming
task test:rust-dotnet:behavior:lifecycle
task test:rust-dotnet:behavior:push-config
```

From the repository root:

```sh
task integrations:a2a:test
task integrations:a2a:test:rust-go
task integrations:a2a:test:rust-go:jsonrpc
task integrations:a2a:test:rust-go:rest
task integrations:a2a:test:rust-go:grpc
task integrations:a2a:test:rust-go:jsonrpc:go-go
task integrations:a2a:test:rust-go:jsonrpc:go-rust
task integrations:a2a:test:rust-go:jsonrpc:rust-go
task integrations:a2a:test:rust-go:jsonrpc:rust-rust
task integrations:a2a:test:rust-go:rest:go-go
task integrations:a2a:test:rust-go:rest:go-rust
task integrations:a2a:test:rust-go:rest:rust-go
task integrations:a2a:test:rust-go:rest:rust-rust
task integrations:a2a:test:rust-go:grpc:go-go
task integrations:a2a:test:rust-go:grpc:go-rust
task integrations:a2a:test:rust-go:grpc:rust-go
task integrations:a2a:test:rust-go:grpc:rust-rust
```

`task integrations:a2a:test` is the repository-level alias for the same combined Rust/Go plus Rust/.NET run.

The repository-level aliases for the new slice remain available for focused runs:

```sh
task integrations:a2a:test:rust-dotnet
task integrations:a2a:test:rust-dotnet:jsonrpc
task integrations:a2a:test:rust-dotnet:jsonrpc:dotnet-dotnet
task integrations:a2a:test:rust-dotnet:jsonrpc:dotnet-rust
task integrations:a2a:test:rust-dotnet:jsonrpc:rust-dotnet
task integrations:a2a:test:rust-dotnet:jsonrpc:rust-rust-dotnet
task integrations:a2a:test:rust-dotnet:rest
task integrations:a2a:test:rust-dotnet:rest:dotnet-dotnet
task integrations:a2a:test:rust-dotnet:rest:dotnet-rust
task integrations:a2a:test:rust-dotnet:rest:rust-dotnet
task integrations:a2a:test:rust-dotnet:rest:rust-rust-dotnet
```

The same behavior-focused targets are available from the repository root:

```sh
task integrations:a2a:test:behavior:core
task integrations:a2a:test:behavior:unary-streaming
task integrations:a2a:test:behavior:lifecycle
task integrations:a2a:test:behavior:push-config
task integrations:a2a:test:behavior:parity
task integrations:a2a:test:rust-go:behavior:unary-streaming
task integrations:a2a:test:rust-go:behavior:lifecycle
task integrations:a2a:test:rust-dotnet:behavior:push-config
```

Each run writes Ginkgo JSON and JUnit reports under `integrations/agntcy-a2a/reports/`. The combined Rust/Go suite emits `report-agntcy-a2a.{json,xml}`, the combined Rust/.NET suite emits `report-agntcy-a2a-rust-dotnet.{json,xml}`, the transport-scoped tasks emit `report-agntcy-a2a-jsonrpc.{json,xml}`, `report-agntcy-a2a-rest.{json,xml}`, `report-agntcy-a2a-grpc.{json,xml}`, `report-agntcy-a2a-rust-dotnet-jsonrpc.{json,xml}`, and `report-agntcy-a2a-rust-dotnet-rest.{json,xml}`, and the per-case tasks emit scenario-specific report names via `-ginkgo.label-filter`.
