# A2A SLIMRPC interop — composed context

Handoff summary for **CSIT** `integrations/agntcy-a2a-slimrpc/`: what it does, what we learned on **slim `main`**, what landed in the repo, and how to run it.

---

## Purpose

Cross-language **A2A over SLIMRPC** smoke/interop tests:

- **Go** via [slim-a2a-go](https://github.com/agntcy/slim-a2a-go) + [slim-bindings-go](https://github.com/agntcy/slim-bindings-go)
- **Python** via PyPI `slima2a` + `slim-bindings` (when versions align)

**Matrix (intended):** go/python × server/client → **4 specs**, echo payload `Hello there!`.

**Separate from:** `integrations/agntcy-a2a` (JSON-RPC/REST/gRPC) and `integrations/agntcy-slim` (K8s topology).

**Entry point:** `task integrations:a2a-slimrpc:test` from repo root.

---

## Layout

```
integrations/agntcy-a2a-slimrpc/
├── Taskfile.yml              # test + reports:dashboard
├── README.md                 # runbook + troubleshooting
├── tests/
│   ├── suite_test.go
│   ├── matrix_test.go        # BeforeSuite, ordered 4 specs (or 1 in go-only)
│   └── launchers_test.go     # setup, venv, builds, env hooks
├── fixtures/
│   ├── go/cmd/{server,probe}/  # minimal echo A2A over SLIMRPC
│   └── python/                 # csit_server.py, csit_probe.py, requirements*
└── .cache/                   # built Go bins, Python venv (gitignored)
```

Reports: `reports/*.json`, `*.xml`, `index.html` (same pattern as `agntcy-a2a`).

---

## Runtime model

1. **BeforeSuite:** `slim-bindings-setup`, optional native lib override, build Go fixtures, optional Python venv.
2. **Each spec:** start fixture server → wait for `CSIT_SLIM_SERVER_READY` → run probe → assert echo substring.
3. **Identities:** `agntcy/a2a_csit_slim/server_<lang>`, `client_<lang>`.
4. **Ordered suite:** first failure skips remaining specs.

**Requires:** reachable SLIM node (default `http://127.0.0.1:46357`). The harness checks TCP reachability; `curl` returning `000` means nothing is listening.

---

## The slim `main` constraint (core lesson)

Many developers **must** run **slim from `main`** (e.g. `cargo run --bin slim`), not an older release.

That creates a **three-layer alignment** problem:

| Layer | Released / default CSIT | slim `main` (≈2.0) |
|--------|-------------------------|---------------------|
| **SLIM node** | v1.4.x wire | **Newer dataplane / protos** |
| **Rust static lib** (`.a`) | `slim-bindings-setup@v1.4.1` zip | **Built from same slim commit** |
| **Go FFI** (`slim-bindings-go`) | PyPI / pseudo `v1.4.1` | **Generated from `data-plane/bindings/go`** |
| **Python** | PyPI `slima2a` + `slim-bindings` 1.4.x | **`slim-bindings` 2.x** — **incompatible with `slima2a==0.5.0`** |

**Symptoms when misaligned:**

- Node: `invalid wire type: Varint (expected LengthDelimited)` on `SlimHeader` / `Subscribe`
- Probe: `Session handshake failed: failed to add participant to session`
- Reconnect / `connection not found` spam on the node

These are usually **not** shared-secret issues when protobuf decode errors appear on the node.

---

## What we implemented in CSIT

### Harness / environment variables

| Variable | Role |
|----------|------|
| `SLIM_SERVER` | Node URL (fixtures default `http://127.0.0.1:46357`) |
| `SLIM_SHARED_SECRET` | Must match node config (fixture default: `my_shared_secret_for_testing_purposes_only`) |
| `SKIP_SLIM_A2A=1` | Skip entire suite |
| `SKIP_SLIM_BINDINGS_SETUP=1` | Skip download; native override still runs if native lib set |
| **`CSIT_SLIM_NATIVE_LIB`** | Path to `.a` **or** directory with `libslim_bindings*.a` → copied into `$GOPATH/.cgo-cache/slim-bindings/<version>/` |
| **`CSIT_SLIM_BINDINGS_CGO_VERSION`** | Cache tier (default `v1.4.1`; use **`devel`** with monorepo bindings) |
| **`CSIT_SLIM_BINDINGS_GO_REPLACE`** | Directory with generated `slim_bindings.go` → builds via `go.csit.mod` replace |
| **`CSIT_SLIM_GO_ONLY=1`** | **1 spec** (go→go); skips Python venv |
| `CSIT_SLIM_PYTHON_REQUIREMENTS` | Alternate requirements file under `fixtures/python/` |
| `CSIT_SLIM_STREAM_SERVER_LOGS=1` | Tee server logs to terminal (default off; avoids IDE flood) |
| `PYTHON` | Python ≥3.10 for venv |

### Native lib resolution

- Accepts `libslim_bindings_aarch64_apple_darwin.a` (CGO cache name), `libslim_bindings_aarch64_darwin.a` (slim repo copy), or `libslim_bindings.a` (cargo `target/.../release`).
- Directory paths search common names and `libslim_bindings*.a` via glob.

### Go build with monorepo bindings

- `CSIT_SLIM_BINDINGS_GO_REPLACE` → generates `fixtures/go/go.csit.mod` with `replace github.com/agntcy/slim-bindings-go => <path>`.
- Requires **`slim_bindings.go`** from `task generate` in the slim repo (`data-plane/bindings/go`).

### Log / stability

- Server stdout/stderr are **not** teed to the terminal by default; a capped in-memory buffer is used for the ready marker and failure output.

### Python on `main`

- `fixtures/python/requirements-slim-main.txt` is **comment-only** — pip cannot resolve `slima2a~=1.1` with `slim-bindings` 2.x from `main`.
- Full matrix is blocked until upstream **`slima2a` supports 2.x**.

### Pins (fixtures)

- **Go:** `slim-a2a-go v0.2.0`, `slim-bindings-go` pseudo from main (in `go.mod`); **effective FFI** via `CSIT_SLIM_BINDINGS_GO_REPLACE` on slim `main`.
- **Setup module:** always `slim-bindings-setup@v1.4.1` (release zip exists); **overwritten** by `CSIT_SLIM_NATIVE_LIB` when needed.
- **CGO cache path in upstream 1.4.x Go module:** still `.../slim-bindings/v1.4.1` unless using monorepo `devel` tier via env.

---

## Known-good recipe (slim `main`, Go-only)

Replace paths with your slim checkout. Example layout: slim at `~/WORK/slim`, CSIT at `~/WORK/csit`.

```bash
# 1) In slim — once per bindings change
cd ~/WORK/slim/data-plane/bindings/go
task generate PROFILE=release

# 2) Environment
export SLIM_SERVER=http://127.0.0.1:46357
export SLIM_SHARED_SECRET='my_shared_secret_for_testing_purposes_only'   # or match node config

export CSIT_SLIM_BINDINGS_GO_REPLACE=~/WORK/slim/data-plane/bindings/go/slim_bindings
export CSIT_SLIM_NATIVE_LIB=~/WORK/slim/data-plane/bindings/go/slim_bindings/libslim_bindings_aarch64_darwin.a
export CSIT_SLIM_BINDINGS_CGO_VERSION=devel   # optional when REPLACE is set

export CSIT_SLIM_GO_ONLY=1
unset CSIT_SLIM_PYTHON_REQUIREMENTS

# 3) Run
cd ~/WORK/csit
rm -rf integrations/agntcy-a2a-slimrpc/.cache
task integrations:a2a-slimrpc:test
```

**Expected:** `Will run 1 of 1 specs` → **SUCCESS** (go client → go server).

**Pre-check:** `nc -zv 127.0.0.1 46357` succeeds; slim process is running.

---

## Current status

| Item | Status |
|------|--------|
| Go → Go on slim `main` | **Passing** (with REPLACE + native lib + GO_ONLY) |
| Python matrix (3 other specs) | **Blocked** — `slima2a` vs `slim-bindings` 2.x |
| CI in GitHub Actions | **Not wired** (needs slim on runner + reproducible bindings) |
| Integration tree | Under `integrations/agntcy-a2a-slimrpc/` |

---

## Suggested next steps

1. **Land CSIT changes** — PR with harness, README, fixtures; document required env for slim `main` (machine paths via env vars, not hardcoded in repo).
2. **Optional:** Task target e.g. `test:go-main` that validates env vars and prints the recipe.
3. **Python / full matrix** — wait for or contribute **slima2a + slim-bindings 2.x**; then remove `CSIT_SLIM_GO_ONLY` and fix `requirements-slim-main.txt`.
4. **CI** — only after slim image + bindings install is deterministic on the runner.
5. **Upstream** — published `slim-bindings-go` / CGO cache version for 2.0 so developers avoid manual `REPLACE` + `.a` copy.

---

## Pitfalls (quick reference)

- Placeholder paths like `/absolute/path/to/...` in docs are **examples** — use real paths under your **slim** checkout, not **csit**.
- **`curl` → `000`** = nothing listening on `SLIM_SERVER` (start slim first).
- **`CSIT_SLIM_PYTHON_REQUIREMENTS=requirements-slim-main.txt`** on `main` → pip conflict; use **`CSIT_SLIM_GO_ONLY=1`** instead.
- **Only `.a` without `CSIT_SLIM_BINDINGS_GO_REPLACE`** → handshake/wire errors (Go FFI still 1.4.x).
- **Stale binaries** → `rm -rf integrations/agntcy-a2a-slimrpc/.cache` after pin or env changes.
- **IDE freeze** → avoid `CSIT_SLIM_STREAM_SERVER_LOGS=1` unless actively debugging reconnect noise.

---

## Key files

- `integrations/agntcy-a2a-slimrpc/README.md` — detailed runbook
- `integrations/agntcy-a2a-slimrpc/tests/launchers_test.go` — env + setup logic
- `integrations/agntcy-a2a-slimrpc/tests/matrix_test.go` — suite + go-only matrix
- `integrations/Taskfile.yml` — `includes: a2a-slimrpc`
