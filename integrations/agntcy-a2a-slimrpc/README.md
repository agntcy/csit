# A2A SLIMRPC interoperability (CSIT)

Cross-language checks that **A2A over SLIMRPC** behaves consistently across **Go** ([slim-a2a-go](https://github.com/agntcy/slim-a2a-go)), **Python** ([slim-a2a-python](https://github.com/agntcy/slim-a2a-python) / PyPI `slima2a`), **.NET** ([slim-a2a-dotnet](https://github.com/agntcy/slim-a2a-dotnet) / NuGet `Agntcy.SlimA2A`), **Java** ([slim-a2a-java](https://github.com/agntcy/slim-a2a-java) / Maven Central `slim-a2a-java`), and **Node** ([slim-a2a-node](https://github.com/agntcy/slim-a2a-node) / npm `@agntcy/slim-a2a`). Every client↔server language pair is exercised — a **5×5 matrix (25 pairs)** — and all of them interop for real: the five published SDKs all consume **`slim-bindings` 2.x** and all generate the `lf.a2a.v1` A2A service, so they share one SLIM dataplane wire.

This is separate from [`integrations/agntcy-a2a`](../agntcy-a2a) (JSON-RPC / REST / gRPC only) and from [`integrations/agntcy-slim/topology`](../agntcy-slim/topology) (Kubernetes SLIM topology).

> **Baseline: a pinned _released_ slim 2.x node (`ghcr.io/agntcy/slim:2.1.1`).** Against that node the full **5×5** matrix works off-the-shelf — the prebuilt `slim-bindings-go` v2 static lib (Go), PyPI `slim-bindings` 2.x (Python), the `Agntcy.SlimA2A` NuGet (.NET), the `slim-a2a-java` Maven artifact (Java), and the `@agntcy/slim-a2a` npm package (Node) all line up with the node's dataplane wire, so **no overrides and no SDK fork** are needed. (Node images and `slim-bindings` version independently within the 2.x line — `2.1.1` is the node, while the SDKs pin bindings in the `2.0.x`–`2.1.x` range; a released 2.1.x node bridges that client range.) Running a node built from `slim` `main` (an unreleased wire) is a **dev-only** mode that requires the override env vars below.

## Prerequisites

1. **SLIM node** reachable from your machine (default `http://127.0.0.1:46357`). **Bindings must match the dataplane wire** of the slim you run.
   - **Recommended (released baseline):** run **`slim:2.1.1`** — pull `ghcr.io/agntcy/slim:2.1.1`, or in a `slim` checkout `git checkout slim-v2.1.1 && cargo run --bin slim -- --config data-plane/config/base/server-config.yaml`. The default `slim-bindings-setup@v2.1.0` prebuilt lib (Go), PyPI `slim-bindings` 2.x (Python), and the published .NET / Java / Node packages all match it, so the **full 5×5 passes with no overrides**.
   - **Dev-only (`slim` `main`):** the released prebuild may be **too old** for a `main` node → protobuf / handshake failures unless you supply a matching native library (see **`CSIT_SLIM_NATIVE_LIB`** and [Go native and `slim` `main`](#go-native-and-slim-main-dev-only)).
2. **Environment**
   - `SLIM_SERVER` — SLIM HTTP endpoint (optional; default `http://127.0.0.1:46357`).
   - `SLIM_SHARED_SECRET` — shared secret for apps (optional; default matches the fixtures and must match the SLIM node configuration).
   - `CSIT_SLIM_STREAM_SERVER_LOGS=1` — stream fixture server stdout/stderr to your terminal (default off; avoids IDE overload when the dataplane logs reconnect storms).
   - **`CSIT_SLIM_NATIVE_LIB`** — absolute path to a locally built `libslim_bindings_*.a`, **or** a directory that contains that file (see [Go native and `slim` `main`](#go-native-and-slim-main-dev-only)). The suite copies it into `$GOPATH/.cgo-cache/slim-bindings/v2.1.0/` before linking Go fixtures (same tier upstream CGO uses).
   - **`CSIT_SLIM_NO_DOTNET=1` / `CSIT_SLIM_NO_JAVA=1` / `CSIT_SLIM_NO_NODE=1`** — drop the .NET / Java / Node axis if that toolchain is unavailable. **`CSIT_SLIM_GO_ONLY=1`** collapses to the single go→go spec (handy when only the Go toolchain is present).
3. **Go SLIM fixtures (native library)**: `slim-bindings-go/v2` links against `$GOPATH/.cgo-cache/slim-bindings/v2.1.0/` (that **`v2.1.0` segment is fixed in upstream `slim_bindings.go`**, not derived from a `go.mod` pseudo-version). The suite runs **`go run …/slim-bindings-go/v2/cmd/slim-bindings-setup@v2.1.0`** so the prebuilt zip exists on [slim releases](https://github.com/agntcy/slim/releases), then optionally **overwrites** that copy when `CSIT_SLIM_NATIVE_LIB` is set. Needs **network** the first time unless you skip setup and provide the override library yourself. To skip the download step: `SKIP_SLIM_BINDINGS_SETUP=1` (native override still runs if `CSIT_SLIM_NATIVE_LIB` is set).

4. **Toolchains**: Go version **≥ the `go` line in [`fixtures/go/go.mod`](fixtures/go/go.mod)** (currently **1.25.2**, driven by [`slim-a2a-go`](https://github.com/agntcy/slim-a2a-go) v0.3.0); the repo root [`integrations/go.mod`](../go.mod) may differ. **Python 3.10+** on `PATH` with `venv` (macOS `/usr/bin/python3` is often 3.9 and cannot install `slima2a`; use Homebrew `python@3.12` or set `PYTHON=/path/to/python3.12`). **.NET 8 SDK** on `PATH` (`dotnet`) for the .NET fixture; the suite runs `dotnet build` in [`fixtures/dotnet`](fixtures/dotnet) and restores `Agntcy.SlimA2A` from nuget.org. **JDK 21** for the Java fixture (compiled with `--enable-preview`); the suite runs `mvn clean package` in [`fixtures/java`](fixtures/java) and resolves `slim-a2a-java` from Maven Central — set **`CSIT_SLIM_JAVA_HOME`** (or `JAVA_HOME`) to a JDK 21 if your default `java` is a different major. **Node.js 18+** with `npm` for the Node fixture; the suite runs `npm ci && npm run build` in [`fixtures/node`](fixtures/node) and resolves the published `@agntcy/slim-a2a` from npm. All fixture builds need **network** on first run.

## Run

### Quickstart (released baseline, full 5×5)

Start the matching released node (Docker is simplest):

```bash
docker run -d --name slim-node -p 46357:46357 \
  -v "$PWD/integrations/agntcy-a2a-slimrpc/ci/slim-server-config.yaml:/config.yaml:ro" \
  ghcr.io/agntcy/slim:2.1.1 /slim --config /config.yaml   # insecure TLS on :46357
```

Then, from this repo's root (no override env vars):

```bash
task integrations:a2a-slimrpc:test   # 25 pairs: {go,python,dotnet,java,node} client × {…} server
```

`task integrations:a2a-slimrpc:versions` prints the exact pinned node image / release / bindings.

### Commands

From repo root:

```bash
task integrations:a2a-slimrpc:test
```

From the `integrations/agntcy-a2a-slimrpc` directory:

```bash
task test
```

To **skip** the suite (for example when no SLIM node is available):

```bash
SKIP_SLIM_A2A=1 task integrations:a2a-slimrpc:test
```

If `SKIP_SLIM_A2A` is unset but the SLIM TCP endpoint is unreachable, each spec **skips** (does not fail).

**`CSIT_SLIM_PYTHON_REQUIREMENTS`** (optional): path to a requirements file used instead of `fixtures/python/requirements.txt`. Relative paths are resolved under `fixtures/python/`. Use this to point at an alternate pin set (e.g. when testing a `slim` `main` node). After switching files, remove the cached venv: `rm -rf integrations/agntcy-a2a-slimrpc/.cache/csit-slim-venv`.

### Go native and `slim` `main` (dev-only)

GitHub ships **one** prebuilt static library per release tag (`slim-bindings-v2.1.0`, …). A **slim server** built from `cargo` on **`main`** may speak a **newer** dataplane than that zip, while CGO still loads libraries from the **`v2.1.0` cache directory**. If you keep a newer `slim-bindings-go` line in `fixtures/go/go.mod` but leave the default zip in place, the **Rust encoder** and **server decoder** disagree → node errors like **`invalid wire type: Varint (expected LengthDelimited)`** and probes report **`Session handshake failed`**.

**Fix:** build `libslim_bindings_<triple>.a` from the **same `slim` commit** as your running node, then point the suite at it:

```bash
# 1) In your slim checkout — generate Go bindings + copy the .a (once per bindings change):
cd /path/to/slim/data-plane/bindings/go
task generate PROFILE=release

# 2) Point CSIT at that tree (Go FFI must match the .a and the running node):
export CSIT_SLIM_BINDINGS_GO_REPLACE=/path/to/slim/data-plane/bindings/go/slim_bindings
export CSIT_SLIM_NATIVE_LIB=/path/to/slim/data-plane/bindings/go/slim_bindings/libslim_bindings_aarch64_darwin.a
export CSIT_SLIM_BINDINGS_CGO_VERSION=devel   # optional; default when REPLACE is set

export CSIT_SLIM_GO_ONLY=1
rm -rf integrations/agntcy-a2a-slimrpc/.cache
task integrations:a2a-slimrpc:test
```

Keep **`fixtures/go/go.mod`** `slim-bindings-go/v2` on a revision that matches that native build. After changing the module pin, clear **`.cache`** again.

## What is tested

- **Matrix**: each of the five languages as **server** is probed by each of the five as **client** — 25 pairs.
- **Behaviors** (per pair): each spec is tagged `behavior-<name>`, which is also the column it occupies in the compatibility-matrix dashboard (see *Reports* below).
  - **`behavior-echo`**: four payloads (ascii, unicode, symbols, a multi-KiB frame) are echoed back verbatim; the response must contain the sent text.
  - **`behavior-lifecycle`**: the probe reports the observed terminal task state and artifact presence; the spec asserts **`TASK_STATE_COMPLETED`** with an **echoed artifact** containing the sent text. Probes emit a parseable block (`CSIT_SLIM_RESULT_KIND`, `CSIT_SLIM_TASK_STATE`, `CSIT_SLIM_ARTIFACT_PRESENT`, `CSIT_SLIM_ARTIFACT_TEXT`).
  - **Scenarios** (`behavior-message-only`, `behavior-task-failure`, `behavior-input-required`, `behavior-streaming`, `behavior-task-cancel`, `behavior-multi-turn`): drive the matching A2A response shape and assert the observed result kind / terminal task state (and, for streaming, the aggregated artifact chunks). **`behavior-multi-turn`** is a two-send conversation: turn 1 reaches `input-required`, then the probe continues the *same* task/context to **`TASK_STATE_COMPLETED`** (asserting the `multi-turn complete` continuation artifact).
- **Identities**: servers use `agntcy/a2a_csit_slim/server_<lang>`; clients use `agntcy/a2a_csit_slim/client_<lang>`.

Run a single behavior slice or pair with a Ginkgo label filter, e.g.:

```bash
cd integrations && go test ./agntcy-a2a-slimrpc/tests -ginkgo.label-filter='behavior-lifecycle'
cd integrations && go test ./agntcy-a2a-slimrpc/tests -ginkgo.label-filter='pair-node-go'
```

## Troubleshooting

- **Huge / “infinite” test output, IDE freeze, or Cursor quitting:** the dataplane can print reconnect lines faster than the UI can render. The harness **no longer tees fixture server stdout/stderr to your terminal by default** (only a capped in-memory buffer used for failures and ready detection). Set **`CSIT_SLIM_STREAM_SERVER_LOGS=1`** only when you want live server logs in the console.

- **Ephemeral `GOMODCACHE` / IDE sandbox:** `slim-bindings-go` resolves the native library via a path relative to `$HOME/go/pkg/mod`. The test harness forces `GOMODCACHE=$HOME/go/pkg/mod` during `slim-bindings-setup` and `go build` when that directory exists. If the linker still cannot find `libslim_bindings_*`, run `go run github.com/agntcy/slim-bindings-go/v2/cmd/slim-bindings-setup@v2.1.0` from `integrations/` (the **release** tag used by `slimBindingsSetupModule` in `tests/launchers_test.go`), then retry.

- **`Session handshake failed` / node `invalid wire type` (`Varint` vs `LengthDelimited`) on `SlimHeader` / `Subscribe`:** almost always **slim-bindings native (.a) vs slim server** skew — not the shared secret. `slim-bindings-setup` installs the **`v2.1.0` release `.a`**; if your node is built from **`main`**, set **`CSIT_SLIM_NATIVE_LIB`** to a **`libslim_bindings_<triple>.a` built from the same `slim` revision as the node**, clear **`integrations/agntcy-a2a-slimrpc/.cache`**, and re-run (see [Go native and `slim` `main`](#go-native-and-slim-main-dev-only)). Align **`fixtures/go/go.mod`** `slim-bindings-go/v2` with that line.

- **Shared secret issues:** if you only see **`Session handshake failed`** without the protobuf decode text above, check **`SLIM_SHARED_SECRET`** matches the node and fixtures.

- **Java `NoSuchMethodError` / `cannot find symbol` after a `slim-a2a-java` bump:** Maven's incremental compiler does not recompile sources when only a dependency version changes. The suite runs `mvn clean package` for exactly this reason; if you build the fixture by hand, use `clean` too, or `rm -rf fixtures/java/target`.

- **Stale Go fixture binaries after changing `slim-bindings-go`:** you may still see wire or handshake oddities until you rebuild. The suite names cached binaries by bindings tag (`slimBindingsCacheTag` in `tests/launchers_test.go`) and rebuilds when `fixtures/go/go.mod` is newer; you can `rm -rf integrations/agntcy-a2a-slimrpc/.cache` and re-run `slim-bindings-setup` at the same tag.

## Reports

JUnit / JSON and a dashboard are written under `reports/`, same pattern as `agntcy-a2a`. From `integrations/agntcy-a2a-slimrpc`:

```bash
task reports:dashboard
```

The dashboard (rendered by the shared `agntcy-a2a/tools/report_dashboard.go`) opens with a **Compatibility Matrix**: one row per direction (`Go→Node`, `Node→Java`, …) and one `SlimRPC` column per behavior (`echo`, `lifecycle`, and each scenario). Cells aggregate the worst observed state for that direction × behavior. In CI it is published to GitHub Pages under `/<repo>/a2a-slimrpc/`.

## CI

This suite runs in GitHub Actions via the **`run-tests-a2a-slimrpc`** matrix job in [`.github/workflows/test-a2a-slimrpc.yaml`](../../.github/workflows/test-a2a-slimrpc.yaml) — one parallel job per client→server pair (25 pairs), mirroring the `agntcy-a2a` layout. Each job:

- Starts the **pinned released slim node** `ghcr.io/agntcy/slim:2.1.1` (env `SLIM_IMAGE`) as a background container (`docker run … /slim --config …`, mounting [`ci/slim-server-config.yaml`](ci/slim-server-config.yaml) — insecure TLS on a loopback `:46357`). GitHub `services:` containers can't pass the required `--config` arg, so a `docker run` step is used.
- Sets up Python + Go + .NET + Java + Node via [`.github/actions/setup-env`](../../.github/actions/setup-env), then runs its pair task (e.g. `task integrations:a2a-slimrpc:test:go-node`), no overrides.
- Uploads `reports/*` as a per-pair `a2a-slimrpc-test-result-<pair>` artifact. The Pages workflow merges all pairs into the published dashboard (one "Saved Run" per pair plus the compatibility matrix). Run the full 5×5 locally with `task integrations:a2a-slimrpc:test`.

Gating mirrors the other integrations: an `a2a-slimrpc` paths filter triggers it on pushes/PRs that touch this directory. The released node image and the published `slim-bindings` 2.x SDK packages line up off-the-shelf, so no SDK fork is required.

## Pinned upstream versions

The released baseline is the **single source of truth**, recorded as `SLIM_IMAGE` / `SLIM_RELEASE` / `SLIM_BINDINGS_RELEASE` vars in [`Taskfile.yml`](Taskfile.yml) (`task versions` prints them) and used by both local runs and CI. Each fixture consumes its language's **latest published SDK**, which pins its own `slim-bindings` 2.x; the released 2.1.x node bridges the `2.0.x`–`2.1.x` client range.

| Component | Pin |
|-----------|-----|
| SLIM node (CI + recommended local) | `ghcr.io/agntcy/slim:2.1.1` (locally: `docker run … ghcr.io/agntcy/slim:2.1.1`, or `git checkout slim-v2.1.1` then `cargo run --bin slim`). Node images and `slim-bindings` version independently in 2.x; a 2.1.x node serves the SDKs' 2.0.x–2.1.x bindings. |
| Go fixtures (`fixtures/go/go.mod`) | `github.com/agntcy/slim-a2a-go` v0.3.0; `github.com/agntcy/slim-bindings-go/v2` v2.1.0; `a2a-go/v2` v2.1.0 |
| Go native (CGO) | Default: prebuilt from `go run …/slim-bindings-go/v2/cmd/slim-bindings-setup@v2.1.0` → `$GOPATH/.cgo-cache/slim-bindings/v2.1.0/`. For **`slim` `main`** (dev-only): override with **`CSIT_SLIM_NATIVE_LIB`**. |
| Python (`fixtures/python/requirements.txt`) | `slima2a==0.7.0`, `slim-bindings>=2.0.0,<3.0.0` on PyPI (a2a-sdk 1.1.0 pulled transitively) |
| .NET (`fixtures/dotnet/Csit.csproj`) | `Agntcy.SlimA2A` 0.2.0 (NuGet); transitively `Agntcy.Slim.SlimRpc` 2.x |
| Java (`fixtures/java/pom.xml`) | `io.agntcy.slim:slim-a2a-java` 0.2.1 (Maven Central); `slim-bindings-java` 2.0.0; a2a-java-sdk 1.1.0.Final; JDK 21 + `--enable-preview` |
| Node fixture (`fixtures/node/package.json`) | `@agntcy/slim-a2a` 0.2.0 (npm) + `@a2a-js/sdk` 1.0.0-beta.0; `@agntcy/slim-bindings` 2.0.0 |
