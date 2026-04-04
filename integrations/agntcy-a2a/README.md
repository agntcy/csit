# A2A Interoperability CSIT

This component hosts cross-SDK A2A interoperability checks.

The initial slice covers a self-contained Rust and Go JSON-RPC smoke matrix:

- Go client -> Go server
- Go client -> Rust server
- Rust client -> Go server
- Rust client -> Rust server

The fixtures are intentionally small and deterministic so the suite can run the same way locally and in CI without depending on sibling SDK checkouts.

## Matrix

| Label | Scenario | Component task | Repository task |
| --- | --- | --- | --- |
| `go-go` | Go client -> Go server | `task test:rust-go:jsonrpc:go-go` | `task integrations:a2a:test:rust-go:jsonrpc:go-go` |
| `go-rust` | Go client -> Rust server | `task test:rust-go:jsonrpc:go-rust` | `task integrations:a2a:test:rust-go:jsonrpc:go-rust` |
| `rust-go` | Rust client -> Go server | `task test:rust-go:jsonrpc:rust-go` | `task integrations:a2a:test:rust-go:jsonrpc:rust-go` |
| `rust-rust` | Rust client -> Rust server | `task test:rust-go:jsonrpc:rust-rust` | `task integrations:a2a:test:rust-go:jsonrpc:rust-rust` |

## Running The Suite

From `integrations/agntcy-a2a/`:

```sh
task test
task test:rust-go:jsonrpc
task test:rust-go:jsonrpc:go-go
task test:rust-go:jsonrpc:go-rust
task test:rust-go:jsonrpc:rust-go
task test:rust-go:jsonrpc:rust-rust
```

From the repository root:

```sh
task integrations:a2a:test
task integrations:a2a:test:rust-go:jsonrpc
task integrations:a2a:test:rust-go:jsonrpc:go-go
task integrations:a2a:test:rust-go:jsonrpc:go-rust
task integrations:a2a:test:rust-go:jsonrpc:rust-go
task integrations:a2a:test:rust-go:jsonrpc:rust-rust
```

Each run writes Ginkgo JSON and JUnit reports under `integrations/agntcy-a2a/reports/`. The full-matrix task emits `report-agntcy-a2a.{json,xml}`, and the per-case tasks emit scenario-specific report names.
