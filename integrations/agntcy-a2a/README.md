# A2A Interoperability CSIT

This component hosts cross-SDK A2A interoperability checks.

The suite is structured around shared Ginkgo behaviors and per-SDK harnesses. The behavior assertions are written once, expanded across client/server transport matrices, and the language-specific differences are isolated behind Go, Rust, .NET, and Python launchers or probes.

The current coverage includes a Rust/Go suite across JSON-RPC, HTTP+JSON, and gRPC; .NET-backed suites for Rust, Go, and Python across JSON-RPC and HTTP+JSON; and Python v1.0 suites for both Go and Rust across JSON-RPC, HTTP+JSON, and gRPC. Each client/server leg is split into shared behavior slices for unary and streaming requests, task lifecycle APIs, push-config semantics, and scenario parity.

All 12 Rust/Go client-server legs are green across JSON-RPC, HTTP+JSON, and gRPC. The Go and Rust fixtures expose push-config CRUD, and CSIT validates that path from both clients against both server targets across all three transports.

The Rust/.NET suite reuses the existing Rust fixture and Rust probe, adds CSIT-owned .NET fixture and probe binaries, and covers 8 legs: 4 JSON-RPC legs (`dotnet-dotnet`, `dotnet-rust`, `rust-dotnet`, `rust-rust-dotnet`) plus the same 4 legs over HTTP+JSON. This slice does not currently cover gRPC.

The Go/.NET suite reuses the existing Go fixture with the same CSIT-owned .NET fixture and probe binaries, and covers 8 legs: 4 JSON-RPC legs (`go-go`, `go-dotnet`, `dotnet-go`, `dotnet-dotnet`) plus the same 4 legs over HTTP+JSON. This slice does not currently cover gRPC.

The Python/Go suite adds a CSIT-owned Python server and probe built against the `1.0-dev` branch of `a2aproject/a2a-python`. It now covers 12 legs: 4 JSON-RPC legs (`go-go`, `go-python`, `python-go`, `python-python`), the same 4 legs over HTTP+JSON, and the same 4 legs over gRPC.

The Rust/Python suite reuses the existing Rust fixture and probe together with the same CSIT-owned Python server and probe. It now covers 12 legs: 4 JSON-RPC legs (`rust-rust`, `rust-python`, `python-rust`, `python-python`), the same 4 legs over HTTP+JSON, and the same 4 legs over gRPC.

The Python/.NET suite reuses the same CSIT-owned Python server and probe together with the same .NET fixture and probe binaries. It currently covers 8 legs: 4 JSON-RPC legs (`python-python`, `python-dotnet`, `dotnet-python`, `dotnet-dotnet`) plus the same 4 legs over HTTP+JSON. This slice does not currently cover gRPC.

Push-config is covered in the .NET-backed suites as well, but only the non-.NET server legs are expected to support push-config CRUD; the .NET-server legs are expected to return the current unsupported error. The default `task test` and `task integrations:a2a:test` entrypoints run the Rust/Go, Rust/.NET, Go/.NET, Python/Go, Rust/Python, and Python/.NET suites.

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
- successful push-config CRUD on supported server paths across the covered transports
- preservation of a mixed text plus structured-data request payload and message metadata through task history

The fixtures are intentionally small and deterministic so the suite can run the same way locally and in CI without depending on sibling SDK checkouts.

Each leg keeps its existing suite, transport, and pair labels for Task targets, and the shared behavior slices add dedicated Ginkgo labels (`behavior-core`, `behavior-unary-streaming`, `behavior-lifecycle`, `behavior-push-config`, and `behavior-parity`) so full legs and cross-leg behavior groups can be filtered independently.

The gRPC legs follow the same agent-card discovery path as the other transports: each fixture serves `/.well-known/agent-card.json` over HTTP and advertises a separate gRPC transport endpoint from that card.

The .NET-backed slices currently require a local `dotnet` CLI for the .NET 8 SDK because the CSIT harness builds the fixture and probe from published `A2A` and `A2A.AspNetCore` NuGet packages at test time.

The Python/Go, Rust/Python, and Python/.NET slices require Python 3.10+ and install the Python SDK fixture environment into a temporary virtualenv on first use from `fixtures/python/requirements.txt`, which pins the SDK to the `1.0-dev` branch.

## Matrix

Legend: ✅ covered by passing automated CSIT, ❌ not currently covered by this suite.

### SDK Pair Coverage

| SDK pair | JSON-RPC | HTTP+JSON | gRPC | Component task |
| --- | --- | --- | --- | --- |
| Rust/Go | ✅ | ✅ | ✅ | `task test:rust-go` |
| Rust/.NET | ✅ | ✅ | ❌ | `task test:rust-dotnet` |
| Go/.NET | ✅ | ✅ | ❌ | `task test:go-dotnet` |
| Python/Go | ✅ | ✅ | ✅ | `task test:python-go` |
| Rust/Python | ✅ | ✅ | ✅ | `task test:rust-python` |
| Python/.NET | ✅ | ✅ | ❌ | `task test:python-dotnet` |

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

### Go/.NET Leg Coverage

| Client -> Server | JSON-RPC | HTTP+JSON | gRPC |
| --- | --- | --- | --- |
| Go -> Go (Go/.NET slice) | ✅ | ✅ | ❌ |
| Go -> .NET | ✅ | ✅ | ❌ |
| .NET -> Go | ✅ | ✅ | ❌ |
| .NET -> .NET (Go/.NET slice) | ✅ | ✅ | ❌ |

### Python/Go Leg Coverage

| Client -> Server | JSON-RPC | HTTP+JSON | gRPC |
| --- | --- | --- | --- |
| Go -> Go (Python/Go slice) | ✅ | ✅ | ✅ |
| Go -> Python | ✅ | ✅ | ✅ |
| Python -> Go | ✅ | ✅ | ✅ |
| Python -> Python | ✅ | ✅ | ✅ |

### Rust/Python Leg Coverage

| Client -> Server | JSON-RPC | HTTP+JSON | gRPC |
| --- | --- | --- | --- |
| Rust -> Rust (Rust/Python slice) | ✅ | ✅ | ✅ |
| Rust -> Python | ✅ | ✅ | ✅ |
| Python -> Rust | ✅ | ✅ | ✅ |
| Python -> Python (Rust/Python slice) | ✅ | ✅ | ✅ |

### Python/.NET Leg Coverage

| Client -> Server | JSON-RPC | HTTP+JSON | gRPC |
| --- | --- | --- | --- |
| Python -> Python (Python/.NET slice) | ✅ | ✅ | ❌ |
| Python -> .NET | ✅ | ✅ | ❌ |
| .NET -> Python | ✅ | ✅ | ❌ |
| .NET -> .NET (Python/.NET slice) | ✅ | ✅ | ❌ |

### Behavior Coverage

| Behavior | Rust/Go | Rust/.NET | Go/.NET | Python/Go | Rust/Python | Python/.NET |
| --- | --- | --- | --- | --- | --- | --- |
| Unary `SendMessage` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Streaming `SendMessage` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Message-only response path | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `GetTask`, `ListTasks`, and `CancelTask` lifecycle | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Failed-task response path | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Multi-turn input-required continuation | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Long-running completion after non-blocking send | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Artifact-rich streaming updates | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Structured text + data + URL artifact payloads | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Extended-agent-card discovery | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Extended-card security-scheme metadata where present | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Missing-task and non-cancelable-task errors | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Mixed text + structured-data payload preserved through task history | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Push-config CRUD against Rust-server targets where covered | ✅ | ✅ | n/a | n/a | ✅ | n/a |
| Unsupported push-config response against .NET-server targets | n/a | ✅ | ✅ | n/a | n/a | ✅ |
| Push-config CRUD against Go- and Python-server targets where covered | n/a | n/a | ✅ | ✅ | ✅ | ✅ |

For the .NET-backed slices, the uncovered cells are the current gRPC gap. For push-config, a ✅ means the suite verifies the expected behavior for that leg: CRUD succeeds on supported Go, Rust, or Python server targets, and the current unsupported error is returned on .NET-server targets.

For the .NET-owned fixture specifically, the public and extended cards currently omit `securitySchemes` because the released Rust resolver used by this CSIT slice does not yet accept the .NET SDK's null-filled union encoding for that field. The overall suites still validate bearer-token security metadata against the Rust, Go, and Python fixtures where that field is present.

## Running the Suite

From `integrations/agntcy-a2a/`:

```sh
task test
task test:rust-go
task test:rust-dotnet
task test:go-dotnet
task test:python-go
task test:rust-python
task test:python-dotnet
```

`task test` now runs the full `task test:rust-go` transport matrix plus `task test:rust-dotnet`, `task test:go-dotnet`, `task test:python-go`, `task test:rust-python`, and `task test:python-dotnet`.

The user-facing task triggers are organized by scope:

- Full suites:

```sh
task test
task test:rust-go
task test:rust-dotnet
task test:go-dotnet
task test:python-go
task test:rust-python
task test:python-dotnet
```

- Cross-suite behavior slices:

```sh
task test:behavior:core
task test:behavior:unary-streaming
task test:behavior:lifecycle
task test:behavior:push-config
task test:behavior:parity
```

- Rust/Go suite filters:

```sh
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
task test:rust-go:behavior:core
task test:rust-go:behavior:unary-streaming
task test:rust-go:behavior:lifecycle
task test:rust-go:behavior:push-config
task test:rust-go:behavior:parity
```

- Rust/.NET suite filters:

```sh
task test:rust-dotnet:jsonrpc
task test:rust-dotnet:rest
task test:rust-dotnet:jsonrpc:dotnet-dotnet
task test:rust-dotnet:jsonrpc:dotnet-rust
task test:rust-dotnet:jsonrpc:rust-dotnet
task test:rust-dotnet:jsonrpc:rust-rust-dotnet
task test:rust-dotnet:rest:dotnet-dotnet
task test:rust-dotnet:rest:dotnet-rust
task test:rust-dotnet:rest:rust-dotnet
task test:rust-dotnet:rest:rust-rust-dotnet
task test:rust-dotnet:behavior:core
task test:rust-dotnet:behavior:unary-streaming
task test:rust-dotnet:behavior:lifecycle
task test:rust-dotnet:behavior:push-config
task test:rust-dotnet:behavior:parity
```

- Go/.NET suite filters:

```sh
task test:go-dotnet:jsonrpc
task test:go-dotnet:rest
task test:go-dotnet:jsonrpc:go-go
task test:go-dotnet:jsonrpc:go-dotnet
task test:go-dotnet:jsonrpc:dotnet-go
task test:go-dotnet:jsonrpc:dotnet-dotnet
task test:go-dotnet:rest:go-go
task test:go-dotnet:rest:go-dotnet
task test:go-dotnet:rest:dotnet-go
task test:go-dotnet:rest:dotnet-dotnet
task test:go-dotnet:behavior:core
task test:go-dotnet:behavior:unary-streaming
task test:go-dotnet:behavior:lifecycle
task test:go-dotnet:behavior:push-config
task test:go-dotnet:behavior:parity
```

- Python/Go suite filters:

```sh
task test:python-go:jsonrpc
task test:python-go:rest
task test:python-go:grpc
task test:python-go:jsonrpc:go-go
task test:python-go:jsonrpc:go-python
task test:python-go:jsonrpc:python-go
task test:python-go:jsonrpc:python-python
task test:python-go:rest:go-go
task test:python-go:rest:go-python
task test:python-go:rest:python-go
task test:python-go:rest:python-python
task test:python-go:grpc:go-go
task test:python-go:grpc:go-python
task test:python-go:grpc:python-go
task test:python-go:grpc:python-python
task test:python-go:behavior:core
task test:python-go:behavior:unary-streaming
task test:python-go:behavior:lifecycle
task test:python-go:behavior:push-config
task test:python-go:behavior:parity
```

- Rust/Python suite filters:

```sh
task test:rust-python:jsonrpc
task test:rust-python:rest
task test:rust-python:grpc
task test:rust-python:jsonrpc:rust-rust
task test:rust-python:jsonrpc:rust-python
task test:rust-python:jsonrpc:python-rust
task test:rust-python:jsonrpc:python-python
task test:rust-python:rest:rust-rust
task test:rust-python:rest:rust-python
task test:rust-python:rest:python-rust
task test:rust-python:rest:python-python
task test:rust-python:grpc:rust-rust
task test:rust-python:grpc:rust-python
task test:rust-python:grpc:python-rust
task test:rust-python:grpc:python-python
task test:rust-python:behavior:core
task test:rust-python:behavior:unary-streaming
task test:rust-python:behavior:lifecycle
task test:rust-python:behavior:push-config
task test:rust-python:behavior:parity
```

- Python/.NET suite filters:

```sh
task test:python-dotnet:jsonrpc
task test:python-dotnet:rest
task test:python-dotnet:jsonrpc:python-python
task test:python-dotnet:jsonrpc:python-dotnet
task test:python-dotnet:jsonrpc:dotnet-python
task test:python-dotnet:jsonrpc:dotnet-dotnet
task test:python-dotnet:rest:python-python
task test:python-dotnet:rest:python-dotnet
task test:python-dotnet:rest:dotnet-python
task test:python-dotnet:rest:dotnet-dotnet
task test:python-dotnet:behavior:core
task test:python-dotnet:behavior:unary-streaming
task test:python-dotnet:behavior:lifecycle
task test:python-dotnet:behavior:push-config
task test:python-dotnet:behavior:parity
```

The canonical Rust/.NET pair trigger is `rust-rust-dotnet`. The shorter `rust-rust` name is still kept as a compatibility alias in the Taskfile, but new usage should prefer `rust-rust-dotnet`.

From the repository root, prepend `integrations:a2a:` to any component-level task trigger:

```sh
task integrations:a2a:test:behavior:core
task integrations:a2a:test:rust-go:grpc:rust-rust
task integrations:a2a:test:rust-dotnet:jsonrpc:rust-rust-dotnet
task integrations:a2a:test:rust-go:behavior:lifecycle
task integrations:a2a:test:rust-dotnet:behavior:parity
task integrations:a2a:test:go-dotnet:rest:dotnet-go
task integrations:a2a:test:python-go:rest:python-python
task integrations:a2a:test:python-go:grpc:python-python
task integrations:a2a:test:rust-python:jsonrpc:python-rust
task integrations:a2a:test:rust-python:grpc:python-python
task integrations:a2a:test:python-dotnet:jsonrpc:dotnet-python
```

Each run writes Ginkgo JSON and JUnit reports under `integrations/agntcy-a2a/reports/`. The combined Rust/Go suite emits `report-agntcy-a2a.{json,xml}`, the combined Rust/.NET suite emits `report-agntcy-a2a-rust-dotnet.{json,xml}`, the combined Go/.NET suite emits `report-agntcy-a2a-go-dotnet.{json,xml}`, the combined Python/Go suite emits `report-agntcy-a2a-python-go.{json,xml}`, the combined Rust/Python suite emits `report-agntcy-a2a-rust-python.{json,xml}`, and the combined Python/.NET suite emits `report-agntcy-a2a-python-dotnet.{json,xml}`. The transport-scoped tasks emit `report-agntcy-a2a-jsonrpc.{json,xml}`, `report-agntcy-a2a-rest.{json,xml}`, `report-agntcy-a2a-grpc.{json,xml}`, `report-agntcy-a2a-rust-dotnet-jsonrpc.{json,xml}`, `report-agntcy-a2a-rust-dotnet-rest.{json,xml}`, `report-agntcy-a2a-go-dotnet-jsonrpc.{json,xml}`, `report-agntcy-a2a-go-dotnet-rest.{json,xml}`, `report-agntcy-a2a-python-go-jsonrpc.{json,xml}`, `report-agntcy-a2a-python-go-rest.{json,xml}`, `report-agntcy-a2a-python-go-grpc.{json,xml}`, `report-agntcy-a2a-rust-python-jsonrpc.{json,xml}`, `report-agntcy-a2a-rust-python-rest.{json,xml}`, `report-agntcy-a2a-rust-python-grpc.{json,xml}`, `report-agntcy-a2a-python-dotnet-jsonrpc.{json,xml}`, and `report-agntcy-a2a-python-dotnet-rest.{json,xml}`, and the per-case tasks emit scenario-specific report names via `-ginkgo.label-filter`.

## How to Add a Test

Most new interop coverage should be added once in the shared behavior layer so the same assertions automatically run across the existing client/server matrix.

### Example: `covers task lifecycle behavior`

The existing `covers task lifecycle behavior` entry in `tests/interop_behaviors_test.go` is the reference pattern for a shared cross-SDK test:

```go
{
	name:   "covers task lifecycle behavior",
	labels: []string{"behavior-core", "behavior-lifecycle"},
	run: func(ctx context.Context, harness interopHarness, target interopTarget) {
		harness.AssertTaskLifecycle(ctx, target)
	},
}
```

That one behavior entry is expanded into multiple specs because the suite wrappers register matrices through `registerInteropTransportMatrix(...)`.

To add a new shared behavior:

1. Add a new method to `interopHarness` in `tests/interop_behaviors_test.go`.
2. Implement that method in each harness that should participate, such as `goSDKHarness`, `rustProbeHarness`, `dotNetProbeHarness`, and `pythonProbeHarness`.
3. Add a new entry to `sharedInteropBehaviorSpecs` with a stable label. If the behavior should be runnable as a first-class filtered target like the current behavior slices, add matching Taskfile targets and document them here.
4. If the Rust, .NET, or Python harnesses need their own focused scenario selection, add a new `probeScenario` in `tests/interop_shared_test.go`, pass it through `tests/interop_launchers_test.go`, and implement the scenario in the matching probe binary:
   `fixtures/rust/src/bin/interop-rust-probe.rs`
   `fixtures/dotnet/InteropProbe/Program.cs`
	`fixtures/python/interop_python_probe.py`
5. Run the narrowest task that exercises the new behavior first, for example `task test:rust-go:behavior:lifecycle` or `task test:behavior:lifecycle`, then run the broader suite once that path is green.

If you are adding a new leg rather than a new behavior, use the Rust/Go suite wrapper in `tests/interop_rust_go_test.go`, the Rust/.NET suite wrapper in `tests/interop_rust_dotnet_test.go`, the Go/.NET suite wrapper in `tests/interop_go_dotnet_test.go`, the Python/Go suite wrapper in `tests/interop_python_go_test.go`, the Rust/Python suite wrapper in `tests/interop_rust_python_test.go`, or the Python/.NET suite wrapper in `tests/interop_python_dotnet_test.go` as the model. Update the `clients` or `servers` tables there and let `registerInteropTransportMatrix(...)` expand the existing shared behaviors over the new matrix entry. Keep wrapper-level customization limited to matrix declarations and pair-specific overrides, as shown in `tests/interop_rust_dotnet_test.go` for the .NET-server push-config expectations.
