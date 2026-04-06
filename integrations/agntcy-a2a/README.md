# A2A Interoperability CSIT

This component hosts cross-SDK A2A interoperability checks.

The current slice covers Rust and Go across the released JSON-RPC, HTTP+JSON, and gRPC bindings.

All 12 Rust/Go client-server legs in the current core lifecycle matrix are green across JSON-RPC, HTTP+JSON, and gRPC.

The Go and Rust fixtures now expose push-config CRUD. CSIT validates that path from both clients against both server targets across JSON-RPC, HTTP+JSON, and gRPC.

Across the matrix, the scenarios validate the same core interoperability behavior:

- unary and streaming `SendMessage`
- lifecycle methods across `GetTask`, `ListTasks`, and `CancelTask`
- negative-path error semantics for missing and non-cancelable tasks
- successful push-config CRUD on both server paths across all three transports
- preservation of a mixed text plus structured-data request payload and message metadata through task history

The fixtures are intentionally small and deterministic so the suite can run the same way locally and in CI without depending on sibling SDK checkouts.

Each scenario is tagged with a dedicated Ginkgo label and exposed through a matching Task target so the full matrix and each individual leg can be run independently.

The gRPC legs follow the same agent-card discovery path as the other transports: each fixture serves `/.well-known/agent-card.json` over HTTP and advertises a separate gRPC transport endpoint from that card.

## Matrix

| Transport | Label | Scenario | Current outcome | Component task | Repository task |
| --- | --- | --- | --- | --- | --- |
| JSON-RPC | `go-go` | Go client -> Go server | Pass | `task test:rust-go:jsonrpc:go-go` | `task integrations:a2a:test:rust-go:jsonrpc:go-go` |
| JSON-RPC | `go-rust` | Go client -> Rust server | Pass | `task test:rust-go:jsonrpc:go-rust` | `task integrations:a2a:test:rust-go:jsonrpc:go-rust` |
| JSON-RPC | `rust-go` | Rust client -> Go server | Pass | `task test:rust-go:jsonrpc:rust-go` | `task integrations:a2a:test:rust-go:jsonrpc:rust-go` |
| JSON-RPC | `rust-rust` | Rust client -> Rust server | Pass | `task test:rust-go:jsonrpc:rust-rust` | `task integrations:a2a:test:rust-go:jsonrpc:rust-rust` |
| HTTP+JSON | `go-go` | Go client -> Go server | Pass | `task test:rust-go:rest:go-go` | `task integrations:a2a:test:rust-go:rest:go-go` |
| HTTP+JSON | `go-rust` | Go client -> Rust server | Pass | `task test:rust-go:rest:go-rust` | `task integrations:a2a:test:rust-go:rest:go-rust` |
| HTTP+JSON | `rust-go` | Rust client -> Go server | Pass | `task test:rust-go:rest:rust-go` | `task integrations:a2a:test:rust-go:rest:rust-go` |
| HTTP+JSON | `rust-rust` | Rust client -> Rust server | Pass | `task test:rust-go:rest:rust-rust` | `task integrations:a2a:test:rust-go:rest:rust-rust` |
| gRPC | `go-go` | Go client -> Go server | Pass | `task test:rust-go:grpc:go-go` | `task integrations:a2a:test:rust-go:grpc:go-go` |
| gRPC | `go-rust` | Go client -> Rust server | Pass | `task test:rust-go:grpc:go-rust` | `task integrations:a2a:test:rust-go:grpc:go-rust` |
| gRPC | `rust-go` | Rust client -> Go server | Pass | `task test:rust-go:grpc:rust-go` | `task integrations:a2a:test:rust-go:grpc:rust-go` |
| gRPC | `rust-rust` | Rust client -> Rust server | Pass | `task test:rust-go:grpc:rust-rust` | `task integrations:a2a:test:rust-go:grpc:rust-rust` |

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

`task test` is an alias for the full `task test:rust-go` transport matrix run.

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

`task integrations:a2a:test` is the repository-level alias for the same full matrix run.

Each run writes Ginkgo JSON and JUnit reports under `integrations/agntcy-a2a/reports/`. The full transport matrix emits `report-agntcy-a2a.{json,xml}`, the transport-scoped tasks emit `report-agntcy-a2a-jsonrpc.{json,xml}`, `report-agntcy-a2a-rest.{json,xml}`, and `report-agntcy-a2a-grpc.{json,xml}`, and the per-case tasks emit scenario-specific report names via `-ginkgo.label-filter`.
