# A2A Interoperability CSIT

This component hosts cross-SDK A2A interoperability checks.

The initial slice covers a self-contained Rust and Go JSON-RPC smoke matrix:

- Go client -> Go server
- Go client -> Rust server
- Rust client -> Go server
- Rust client -> Rust server

The fixtures are intentionally small and deterministic so the suite can run the same way locally and in CI without depending on sibling SDK checkouts.

Available task targets:

- `task test` or `task test:rust-go:jsonrpc` runs the full matrix.
- `task test:rust-go:jsonrpc:go-go` runs the Go client -> Go server case.
- `task test:rust-go:jsonrpc:go-rust` runs the Go client -> Rust server case.
- `task test:rust-go:jsonrpc:rust-go` runs the Rust client -> Go server case.
- `task test:rust-go:jsonrpc:rust-rust` runs the Rust client -> Rust server case.