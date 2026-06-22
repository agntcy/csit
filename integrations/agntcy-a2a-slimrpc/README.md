# A2A SLIMRPC interoperability (CSIT)

Cross-language checks that **A2A over SLIMRPC** behaves consistently across **Go** ([slim-a2a-go](https://github.com/agntcy/slim-a2a-go)) and **Python** ([slim-a2a-python](https://github.com/agntcy/slim-a2a-python) / PyPI `slima2a`).

This is separate from [`integrations/agntcy-a2a`](../agntcy-a2a) (JSON-RPC / REST / gRPC only) and from [`integrations/agntcy-slim`](../agntcy-slim) (Kubernetes SLIM topology).

> **Default baseline: a pinned _released_ slim node (`slim-v1.4.0` / `ghcr.io/agntcy/slim:1.4.0`).** Against that node the full **Go + Python 2×2** matrix works off-the-shelf — the prebuilt `slim-bindings` 1.4.x static lib and PyPI `slima2a` pins line up with the node's dataplane wire, so **no overrides and no SDK fork** are needed. Running a node from **`slim` `main` / 2.0** is a **dev-only** mode that requires the override env vars below.

## Prerequisites

1. **SLIM node** reachable from your machine (default `http://127.0.0.1:46357`). **Bindings must match the dataplane wire** of the slim you run.
   - **Recommended (released baseline):** run **`slim-v1.4.0`** — locally `git checkout slim-v1.4.0 && cargo run --bin slim -- --config data-plane/config/base/server-config.yaml` in your slim checkout, or pull `ghcr.io/agntcy/slim:1.4.0`. The default `slim-bindings-setup@v1.4.1` prebuilt lib (Go) and PyPI `slim-bindings` 1.4.x (Python) match it, so the **full 2×2 passes with no overrides**.
   - **Dev-only (`slim` `main` / 2.0):** the v1.4.1 prebuild is **too old** for a `main` node → protobuf / handshake failures unless you supply a matching native library (see **`CSIT_SLIM_NATIVE_LIB`** and [Go native and `slim` `main`](#go-native-and-slim-main-dev-only)), and Python is limited to go-only (see [Python and `slim` `main`](#python-and-slim-main-dev-only)).
2. **Environment**
   - `SLIM_SERVER` — SLIM HTTP endpoint (optional; default `http://127.0.0.1:46357`).
   - `SLIM_SHARED_SECRET` — shared secret for apps (optional; default matches Go/Python fixtures and must match the SLIM node configuration).
   - `CSIT_SLIM_STREAM_SERVER_LOGS=1` — stream fixture server stdout/stderr to your terminal (default off; avoids IDE overload when the dataplane logs reconnect storms).
   - **`CSIT_SLIM_NATIVE_LIB`** — absolute path to a locally built `libslim_bindings_*.a`, **or** a directory that contains that file (see [Go native and `slim` `main`](#go-native-and-slim-main-dev-only)). The suite copies it into `$GOPATH/.cgo-cache/slim-bindings/v1.4.1/` before linking Go fixtures (same tier upstream CGO uses).
   - **`CSIT_SLIM_GO_ONLY=1`** — run only the **go→go** interop spec (required for **slim 2.0 / `main`** until PyPI `slima2a` supports `slim-bindings` 2.x).
3. **Go SLIM fixtures (native library)**: `slim-bindings-go` links against `$GOPATH/.cgo-cache/slim-bindings/v1.4.1/` (that **`v1.4.1` segment is fixed in upstream `slim_bindings.go`**, not derived from a `go.mod` pseudo-version). The suite runs **`go run …/slim-bindings-setup@v1.4.1`** so the prebuilt zip exists on [slim releases](https://github.com/agntcy/slim/releases), then optionally **overwrites** that copy when `CSIT_SLIM_NATIVE_LIB` is set. Needs **network** the first time unless you skip setup and provide the override library yourself. To skip the download step: `SKIP_SLIM_BINDINGS_SETUP=1` (native override still runs if `CSIT_SLIM_NATIVE_LIB` is set).

4. **Toolchains**: Go version **≥ the `go` line in [`fixtures/go/go.mod`](fixtures/go/go.mod)** (currently **1.25.2**, driven by [`slim-a2a-go`](https://github.com/agntcy/slim-a2a-go) v0.2.0); the repo root [`integrations/go.mod`](../go.mod) may differ. **Python 3.10+** on `PATH` with `venv` (macOS `/usr/bin/python3` is often 3.9 and cannot install `slima2a`; use Homebrew `python@3.12` or set `PYTHON=/path/to/python3.10`).

## Run

### Quickstart (released baseline, full 2×2)

In your `slim` checkout, run the matching released node:

```bash
git checkout slim-v1.4.0
cargo run --bin slim -- --config data-plane/config/base/server-config.yaml   # insecure TLS on :46357
```

Then, from this repo's root (no override env vars):

```bash
task integrations:a2a-slimrpc:test   # expect 4/4 specs: go/python × go/python
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

**`CSIT_SLIM_PYTHON_REQUIREMENTS`** (optional): path to a requirements file used instead of `fixtures/python/requirements.txt`. Relative paths are resolved under `fixtures/python/`. Use this to point at `requirements-slim-main.txt` when your node is built from `slim` `main`. After switching files, remove the cached venv: `rm -rf integrations/agntcy-a2a-slimrpc/.cache/csit-slim-venv`.

The Python probe uses **non-streaming** `ClientConfig` (`streaming=False`) so `send_message` finishes like the Go unary client. If you ever see a probe **hang** with a custom client, avoid `streaming=True` unless the server drives full stream completion; you can cap wait time with **`CSIT_SLIM_PYTHON_PROBE_TIMEOUT`** (seconds, default `180`).

### Python and `slim` `main` (dev-only)

On current **`slim` `main`**, `slim-bindings` is **2.0.x** while PyPI **`slima2a==0.5.0`** requires **`slim-bindings~=1.1`**, so pip cannot install both. Use **Go-only** interop against a 2.0 node:

```bash
export CSIT_SLIM_GO_ONLY=1
unset CSIT_SLIM_PYTHON_REQUIREMENTS
```

That runs a single spec (**go client → go server**) instead of the full 2×2 matrix. When upstream ships `slima2a` compatible with `slim-bindings` 2.x, extend `fixtures/python/requirements-slim-main.txt` and drop `CSIT_SLIM_GO_ONLY`.

### Go native and `slim` `main` (dev-only)

GitHub only ships **one** prebuilt static library per release tag (`slim-bindings-v1.4.1`, …). Your **slim server** from `cargo` on **`main`** often speaks a **newer** dataplane than that zip, while CGO still loads libraries from the **`v1.4.1` cache directory**. If you keep a newer `slim-bindings-go` line in `fixtures/go/go.mod` but leave the default zip in place, the **Rust encoder** and **server decoder** disagree → node errors like **`invalid wire type: Varint (expected LengthDelimited)`** and probes report **`Session handshake failed`**.

**Fix:** build `libslim_bindings_<triple>.a` from the **same `slim` commit** as your running node (see the `slim` repo / Taskfile for the Rust bindings crate), then point the suite at it:

```bash
# 1) In your slim checkout — generate Go bindings + copy the .a (once per bindings change):
cd /path/to/slim/data-plane/bindings/go
task generate PROFILE=release

# 2) Point CSIT at that tree (Go FFI must match the .a and the running node):
export CSIT_SLIM_BINDINGS_GO_REPLACE=/path/to/slim/data-plane/bindings/go/slim_bindings
export CSIT_SLIM_NATIVE_LIB=/path/to/slim/data-plane/bindings/go/slim_bindings/libslim_bindings_aarch64_darwin.a
export CSIT_SLIM_BINDINGS_CGO_VERSION=devel   # optional; default when REPLACE is set

export CSIT_SLIM_GO_ONLY=1
unset CSIT_SLIM_PYTHON_REQUIREMENTS
rm -rf integrations/agntcy-a2a-slimrpc/.cache
task integrations:a2a-slimrpc:test
```

Keep **`fixtures/go/go.mod`** `slim-bindings-go` on a revision that matches that native build (often `go get github.com/agntcy/slim-bindings-go@main` in `fixtures/go`, then `go mod tidy`). After changing the module pin, clear **`.cache`** again.

## What is tested

- **Matrix**: each of `go` and `python` as **server** is probed by each language as **client** (four pairs).
- **Behaviors** (per pair): each spec is tagged `behavior-<name>`, which is also the column it occupies in the compatibility-matrix dashboard (see *Reports* below).
  - **`behavior-echo`**: probes send `Hello there!`; the response must contain that substring.
  - **`behavior-lifecycle`**: the probe reports the observed terminal task state and artifact presence; the spec asserts **`TASK_STATE_COMPLETED`** with an **echoed artifact** containing the sent text. Probes emit a parseable block (`CSIT_SLIM_RESULT_KIND`, `CSIT_SLIM_TASK_STATE`, `CSIT_SLIM_ARTIFACT_PRESENT`, `CSIT_SLIM_ARTIFACT_TEXT`).
  - **Scenarios** (`behavior-message-only`, `behavior-task-failure`, `behavior-input-required`, `behavior-streaming`, `behavior-task-cancel`, `behavior-multi-turn`): drive the matching A2A response shape and assert the observed result kind / terminal task state (and, for streaming, the aggregated artifact chunks). **`behavior-multi-turn`** is a two-send conversation: turn 1 reaches `input-required`, then the probe continues the *same* task/context to **`TASK_STATE_COMPLETED`** (asserting the `multi-turn complete` continuation artifact).
- **Payload**: `Hello there!`.
- **Identities**: servers use `agntcy/a2a_csit_slim/server_<lang>`; clients use `agntcy/a2a_csit_slim/client_<lang>`.

Run a single behavior slice with a Ginkgo label filter, e.g.:

```bash
cd integrations && go test ./agntcy-a2a-slimrpc/tests -ginkgo.label-filter='behavior-lifecycle'
```

## Troubleshooting

- **Huge / “infinite” test output, IDE freeze, or Cursor quitting:** the dataplane can print reconnect lines faster than the UI can render. The harness **no longer tees fixture server stdout/stderr to your terminal by default** (only a capped in-memory buffer used for failures and ready detection). Set **`CSIT_SLIM_STREAM_SERVER_LOGS=1`** only when you want live server logs in the console. Log capture still **truncates** (stream prefix + tail) to avoid heavy `logs.String()` scans.

- **Ephemeral `GOMODCACHE` / IDE sandbox:** `slim-bindings-go` resolves the native library via a path relative to `$HOME/go/pkg/mod`. The test harness forces `GOMODCACHE=$HOME/go/pkg/mod` during `slim-bindings-setup` and `go build` when that directory exists. If the linker still cannot find `libslim_bindings_*`, run `go run github.com/agntcy/slim-bindings-go/cmd/slim-bindings-setup@v1.4.1` from `integrations/` (the **release** tag used by `slimBindingsSetupModule` in `tests/launchers_test.go`), then retry.

- **`library not found for platform …` from `slim-bindings-setup` with a pseudo-version in the URL:** GitHub only ships prebuilt zips for tags like `slim-bindings-v1.4.1`, not for `go.mod` pseudo-versions. Upstream `slim_bindings.go` still adds `-L…/.cgo-cache/slim-bindings/v1.4.1`, so the harness runs **`slim-bindings-setup@v1.4.1`** even when `fixtures/go/go.mod` uses a newer `slim-bindings-go` pseudo-version for Go code.

- **`Session handshake failed` / node `invalid wire type` (`Varint` vs `LengthDelimited`) on `SlimHeader` / `Subscribe`:** almost always **slim-bindings native (.a) vs slim server** skew — not the shared secret. **`slim-bindings-setup` only installs the v1.4.1 release `.a`**, while your node may be **`main`**. Set **`CSIT_SLIM_NATIVE_LIB`** to a **`libslim_bindings_<triple>.a` built from the same `slim` revision as the node**, clear **`integrations/agntcy-a2a-slimrpc/.cache`**, and re-run (see [Go native and `slim` `main`](#go-native-and-slim-main-dev-only)). Align **`fixtures/go/go.mod`** `slim-bindings-go` with that line. **Python:** use `CSIT_SLIM_PYTHON_REQUIREMENTS=requirements-slim-main.txt` (or a local checkout) and recreate `.cache/csit-slim-venv` when Python fixtures hit the same class of errors.

- **Shared secret issues:** if you only see **`Session handshake failed`** without the protobuf decode text above, check **`SLIM_SHARED_SECRET`** matches the node and fixtures.

- **Stale Go fixture binaries after changing `slim-bindings-go`:** you may still see wire or handshake oddities until you rebuild. The suite names cached binaries by bindings tag and rebuilds when `fixtures/go/go.mod` is newer; you can `rm -rf integrations/agntcy-a2a-slimrpc/.cache` and re-run `slim-bindings-setup` at the same tag as `slimBindingsSetupModule` in `tests/launchers_test.go`.

## Reports

JUnit / JSON and a dashboard are written under `reports/`, same pattern as `agntcy-a2a`. From `integrations/agntcy-a2a-slimrpc`:

```bash
task reports:dashboard
```

The dashboard (rendered by the shared `agntcy-a2a/tools/report_dashboard.go`) opens with a **Compatibility Matrix**: one row per direction (`Go→Go`, `Go→Python`, `Python→Go`, `Python→Python`) and one `SlimRPC` column per behavior (`echo`, `lifecycle`, and each scenario). Cells aggregate the worst observed state for that direction × behavior. In CI it is published to GitHub Pages under `/<repo>/a2a-slimrpc/`.

## CI

This suite runs in GitHub Actions via the **`run-tests-a2a-slimrpc`** matrix job in [`.github/workflows/test-integrations.yaml`](../../.github/workflows/test-integrations.yaml) — one parallel job per client→server pair (`go-go`, `go-python`, `python-go`, `python-python`), mirroring the `agntcy-a2a` layout. Each job:

- Starts the **pinned released slim node** `ghcr.io/agntcy/slim:1.4.0` as a background container (`docker run … /slim --config …`, mounting [`ci/slim-server-config.yaml`](ci/slim-server-config.yaml) — insecure TLS on a loopback `:46357`). GitHub `services:` containers can't pass the required `--config` arg, so a `docker run` step is used.
- Sets up Python + Go, then runs its pair task (e.g. `task integrations:a2a-slimrpc:test:go-python`), no overrides.
- Uploads `reports/*` as a per-pair `a2a-slimrpc-test-result-<pair>` artifact. The Pages workflow merges all four into the published dashboard (one "Saved Run" per pair plus the compatibility matrix). Run the full 2×2 locally with `task integrations:a2a-slimrpc:test`.

Gating mirrors the other integrations: a `a2a-slimrpc` paths filter triggers it on pushes/PRs that touch this directory, and a `skip_slim_a2a_test` `workflow_dispatch` input skips it on demand. The released node image and `slima2a==0.5.0` / `slim-bindings` 1.4.x pins line up off-the-shelf, so no SDK fork is required.

## Pinned upstream versions

The released baseline is the **single source of truth**, recorded as `SLIM_IMAGE` / `SLIM_RELEASE` / `SLIM_BINDINGS_RELEASE` vars in [`Taskfile.yml`](Taskfile.yml) (`task versions` prints them) and used by both local runs and CI.

| Component | Pin |
|-----------|-----|
| SLIM node (CI + recommended local) | `ghcr.io/agntcy/slim:1.4.0` (locally: `git checkout slim-v1.4.0` then `cargo run --bin slim`). There is no `slim-v1.4.1` node tag; `1.4.0` is the closest node release to the `slim-bindings` 1.4.x family and shares its dataplane wire. |
| Go fixtures (`fixtures/go/go.mod`) | `github.com/agntcy/slim-a2a-go` v0.2.0; `slim-bindings-go` 1.4.x (a `main` pseudo-version is only needed for the dev-only `slim main` path) |
| Go native (CGO) | Default: prebuilt from `go run …/slim-bindings-setup@v1.4.1` → `$GOPATH/.cgo-cache/slim-bindings/v1.4.1/`. For **`slim` `main`** (dev-only): override with **`CSIT_SLIM_NATIVE_LIB`** (see below). |
| Python (default) | `slima2a==0.5.0`, `slim-bindings` 1.4.x on PyPI (`fixtures/python/requirements.txt`) |
| Python (`slim` `main` / 2.0 node) | Dev-only, not supported by `slima2a` yet (`slim-bindings` 2.x); use **`CSIT_SLIM_GO_ONLY=1`** |
