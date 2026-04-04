# A2A Interoperability CSIT

This component hosts cross-SDK A2A interoperability checks.

The initial slice covers a self-contained Rust and Go JSON-RPC smoke matrix:

- Go client -> Go server
- Go client -> Rust server
- Rust client -> Go server
- Rust client -> Rust server

The fixtures are intentionally small and deterministic so the suite can run the same way locally and in CI without depending on sibling SDK checkouts.