# C1 evidence contract (v1)

**Status:** Frozen for dashboard slice 1  
**Epic:** [`slim-dashboard-epic.md`](slim-dashboard-epic.md)  
**Issue:** `slim-c1-scope-and-evidence-contract`

This document locks the three **C1** dashboard rows and their evidence sources. The
**analitics** dashboard (`analitics/published/index.html`) uses only `c1-evidence.json`.
Benchmark smoke artifacts (`slim-benchmark-smoke-report`) are documented separately below
for the benchmarks CI/Pages pipeline and are **not** part of analitics.

---

## C1 use-case matrix (frozen)

| Row ID | Class | Use case (reader) | SLIM mechanism | Why SLIM | CSIT scenario | July status |
|--------|-------|-------------------|----------------|----------|---------------|-------------|
| `c1-request-reply` | **C1** | Agent A calls B and waits for a reply | `request-reply` | Named endpoints with synchronous round-trip on one node | C1 Ginkgo evidence (`analitics/tests`) | Proven |
| `c1-fire-and-forget` | **C1** | Agent fires an event; consumer handles async | `fire-and-forget` | One-way delivery through a single SLIM authority with sink observation | Same | Proven |
| `c1-write` | **C1** | Publish into the mesh without a paired responder | `write` | Ingress/write path without a bound responder process | Same | Proven |

**Scenario anchor (shared):**

| Property | Value |
|----------|--------|
| Scenario name | C1 Ginkgo evidence |
| Repo path | `analitics/` (`tests/`, `harness/`, `clients/`) |
| CI task (dashboard) | `task -t analitics/Taskfile.yml test:c1-evidence` |
| slimctl install | `task -t analitics/Taskfile.yml deps:slimctl-download` |
| CI task (benchmark matrix, legacy) | `task benchmarks:slim:benchmark:ci:suite-smoke` |
| CI workflow | `.github/workflows/test-slim-benchmarks.yaml` → job `slim-benchmark-smoke` |
| Ginkgo label (C1 evidence) | `c1-evidence` (`analitics/tests/c1_evidence_test.go`) |
| Ginkgo label (benchmark matrix) | `benchmark-suite` (via `scripts/run_suite.sh`) |
| Smoke matrix (CI) | modes: `request-reply`, `fire-and-forget`, `write`; clients: `1`; size: `16` B; duration: `1s`; repeats: `25` (see `benchmark:ci:suite-smoke` in `benchmarks/agntcy-slim/Taskfile.yml`) |

All three rows share the `c1-evidence.json` bundle. Per-row proof is distinguished by **mode** inside that file (see [Row metadata](#row-metadata-dashboard-fields)).

---

## C1 Ginkgo evidence report (`c1-evidence.json` v1)

### Producer

| Step | Location |
|------|----------|
| Run C1 specs | `task -t analitics/Taskfile.yml test:c1-evidence` → Ginkgo label `c1-evidence` |
| Output | `analitics/reports/c1-evidence.json` |
| Dashboard ingest | `analitics/scripts/evidence-lib.sh` reads per-mode `status` from JSON |

Each case asserts behavioral proof (message counts, errors, round-trip latency) rather than
benchmark throughput statistics. See `analitics/tests/c1_evidence_test.go`.

### Required JSON fields per case

| Field | Role |
|-------|------|
| `row_id` | Stable key (`c1-request-reply`, …) |
| `mode` | SLIM mechanism slug |
| `status` | `verified` when Ginkgo assertions pass |
| `sender_messages` / `sender_errors` | Sender-side proof |
| `sink_received` / `sink_replies` / `sink_errors` | Sink proof (request-reply, fire-and-forget) |
| `assertions` | Human-readable assertion summary lines |

---

## Smoke artifact contract (`slim-benchmark-smoke-report` v1)

> **Note:** Benchmark smoke artifacts remain for throughput dashboards. The C1 evidence
> dashboard (`analitics/`) uses `c1-evidence.json` as the primary status source.

### Producer

| Step | Location |
|------|----------|
| Run suite | `task benchmarks:slim:benchmark:ci:suite-smoke` → writes under `benchmarks/agntcy-slim/reports/` |
| Stage + render | `.github/workflows/test-slim-benchmarks.yaml` → copies into `benchmarks/agntcy-slim/published/smoke/`, runs `task benchmarks:slim:reports:dashboard` |
| Upload | Artifact name **`slim-benchmark-smoke-report`**, path `benchmarks/agntcy-slim/published/smoke`, retention 30 days |
| Pages ingest | `.github/actions/publish-test-reports-pages` (suite: `slim-benchmarks`) → `gh-pages` branch `docs/benchmarks/slim/smoke/` merged into `docs/benchmarks/slim/index.html` |

### Required files (artifact root)

These files **must** be present after a successful smoke job staging step (CI copies only if generated; `index.html` is always produced at stage time).

| File | Role |
|------|------|
| `index.html` | Generated smoke/capacity dashboard (from `report_dashboard.go`) |
| `results.tsv` | Tabular per-run results; **primary source for per-mode status** |
| `suite_summary.md` | Human-readable statistical summary |
| `technical_report.md` | Methodology and run context |
| `ci-smoke-report.md` | CI job wrapper (workflow ref, SHA, embedded summaries) |
| `ci-smoke.log` | Raw stdout; contains `BENCHMARK_RESULT` and `MODE_SUMMARY` lines |

### Optional files

| File | When present |
|------|----------------|
| `reports/raw/*.md` | Local dev runs only; **not** uploaded in CI artifact |
| Per-mode anchors inside `index.html` | Section `smoke-suite` and mode tables from `results.tsv` |

### Local paths (developer)

| Phase | Directory |
|-------|-----------|
| Generator output | `benchmarks/agntcy-slim/reports/` |
| CI-equivalent publish dir | `benchmarks/agntcy-slim/published/smoke/` |

---

## Row metadata (dashboard fields)

Each C1 row uses the same field set. Values are populated at analitics dashboard build time.

### Analitics dashboard status rules

Apply in order (`analitics/scripts/evidence-lib.sh`):

1. Read `analitics/reports/c1-evidence.json` (or synced `published/evidence/c1-evidence.json`).
2. For each row, find the case where `mode` matches the row’s mechanism.
3. Use the case `status` field (`verified`, `failed`, or `unknown` if missing).

No benchmark artifacts (`results.tsv`, `suite_summary.md`, `technical_report.md`) are used by analitics.

### Field definitions (analitics publish)

| Field | Type | Description |
|-------|------|-------------|
| `row_id` | string | Stable key: `c1-request-reply`, `c1-fire-and-forget`, `c1-write` |
| `class` | string | Always `C1` |
| `mode` | string | SLIM mechanism slug: `request-reply`, `fire-and-forget`, `write` |
| `status` | enum | `verified` \| `failed` \| `unknown` — from `c1-evidence.json` |
| `evidence_url` | path | `published/evidence/c1-evidence.json` or `published/c1-evidence-summary.md` |
| `rerun_cmd` | string | `task -t analitics/Taskfile.yml test:c1-evidence` |

### Benchmark smoke status rules (benchmarks CI only)

> Not used by `analitics/`. Retained for `benchmarks/agntcy-slim` and GH Pages benchmark dashboards.

| Field | Type | Description |
|-------|------|-------------|
| `last_run` | string | GitHub Actions run ID for workflow `test-slim-benchmarks`, job `slim-benchmark-smoke` |
| `last_run_url` | URL | `https://github.com/{owner}/{repo}/actions/runs/{run_id}` |
| `artifact_url` | URL | Fallback: Actions run artifact `slim-benchmark-smoke-report` |
| `rerun_cmd` | string | `task benchmarks:slim:benchmark:ci:suite-smoke` |

Apply in order:

1. If workflow job `slim-benchmark-smoke` **conclusion** is not `success` → `failed` for all three rows.
2. Else parse latest `MODE_SUMMARY` for the row’s `mode` from `ci-smoke.log`:
   - `MODE_SUMMARY mode=<mode> ... total_errors=0` → `verified`
   - `MODE_SUMMARY mode=<mode> ... total_errors=<n>` with `n > 0` → `failed`
3. Else parse `results.tsv` rows where column `mode` equals the row’s `mode` (CI smoke: expect 25 rows per mode for `clients=1`, `size=16`):
   - All rows have `sender_runtime_errors=0` and `sink_errors=0` → `verified`
   - Any row with non-zero errors → `failed`
4. If sources are missing → `unknown` (do not mark `verified`).

**Log line shapes** (from `analitics/tests/c1_evidence_test.go` via `analitics/harness`):

```text
MODE_SUMMARY mode=request-reply runs=%d cases=%d ... total_errors=%d
MODE_SUMMARY mode=fire-and-forget runs=%d cases=%d ... total_errors=%d
MODE_SUMMARY mode=write runs=%d cases=%d ... total_errors=%d
```

### URL templates

Replace `{owner}`, `{repo}`, `{run_id}`, `{pages_base}` at publish time.

| Field | Template |
|-------|----------|
| `last_run_url` | `https://github.com/{owner}/{repo}/actions/runs/{run_id}` |
| `artifact_url` | `https://github.com/{owner}/{repo}/actions/runs/{run_id}#artifacts` (artifact: `slim-benchmark-smoke-report`) |
| `evidence_url` (primary) | `{pages_base}/benchmarks/slim/index.html#smoke-suite` |
| `evidence_url` (mode detail, optional) | `{pages_base}/benchmarks/slim/smoke/index.html` or anchor within merged dashboard when mode table exists |
| `evidence_url` (fallback file) | Artifact path `suite_summary.md` or `technical_report.md` |

Published smoke files on Pages (after `publish-test-reports-pages` action):

- `docs/benchmarks/slim/smoke/` on `gh-pages` — raw bundle mirror
- `docs/benchmarks/slim/index.html` on `gh-pages` — merged dashboard (includes smoke section when artifact present)

---

## Per-row source mapping

### Analitics dashboard

| Row ID | `mode` | `status` source | Detail evidence |
|--------|--------|-----------------|-----------------|
| `c1-request-reply` | `request-reply` | `c1-evidence.json` case `mode=request-reply` | `published/evidence/c1-evidence.json` |
| `c1-fire-and-forget` | `fire-and-forget` | `c1-evidence.json` case `mode=fire-and-forget` | same |
| `c1-write` | `write` | `c1-evidence.json` case `mode=write` | same |

### Benchmark smoke (benchmarks CI / Pages only)

| Row ID | `mode` | `status` sources | Detail evidence |
|--------|--------|------------------|-----------------|
| `c1-request-reply` | `request-reply` | Job conclusion; `MODE_SUMMARY mode=request-reply`; `results.tsv` filter `mode=request-reply` | `suite_summary.md` § Request-Reply; `index.html` smoke tables |
| `c1-fire-and-forget` | `fire-and-forget` | Job conclusion; `MODE_SUMMARY mode=fire-and-forget`; `results.tsv` filter `mode=fire-and-forget` | `suite_summary.md` § Fire-And-Forget |
| `c1-write` | `write` | Job conclusion; `MODE_SUMMARY mode=write`; `results.tsv` filter `mode=write` | `suite_summary.md` § Write |

### `results.tsv` columns used for status

| Column | Index (1-based) | Rule |
|--------|-----------------|------|
| `mode` | 1 | Match row `mode` |
| `sender_runtime_errors` | 13 | Must be `0` for every matching row |
| `sink_errors` | 16 | Must be `0` for every matching row |

Header row (verified in repo):

```text
mode	clients	size	rate	repeat	...	sender_runtime_errors	...	sink_errors	...
```

---

## Contract acceptance checklist

- [x] Exactly **3** rows, each tagged **C1**
- [x] Artifact contract **v1** lists files produced by `.github/workflows/test-slim-benchmarks.yaml` staging step
- [x] Each row maps to documented `status`, `last_run`, `evidence_url`, `artifact_url`, `rerun_cmd` sources
- [ ] Next: implement extraction in dashboard template / runbook (`slim-c1-runbook-and-verification`, `slim-c1-dashboard-template`)

---

## References

- `benchmarks/agntcy-slim/tools/report_dashboard.go` — smoke section artifact list
- `benchmarks/agntcy-slim/README.md` — suite and CI smoke description
- Dashboard: [`../../analitics/published/index.html`](../../analitics/published/index.html)
