# C1 evidence contract (v1)

**Status:** Frozen for dashboard slice 1  
**Epic:** [`slim-dashboard-epic.md`](slim-dashboard-epic.md)  
**Issue:** `slim-c1-scope-and-evidence-contract`

This document locks the three **C1** dashboard rows, the `slim-benchmark-smoke-report` artifact bundle, and per-row metadata sources. Template and Pages work should consume this contract only — do not duplicate field definitions elsewhere.

---

## C1 use-case matrix (frozen)

| Row ID | Class | Use case (reader) | SLIM mechanism | Why SLIM | CSIT scenario | July status |
|--------|-------|-------------------|----------------|----------|---------------|-------------|
| `c1-request-reply` | **C1** | Agent A calls B and waits for a reply | `request-reply` | Named endpoints with synchronous round-trip on one node | Local messaging smoke (`benchmarks/agntcy-slim`, CI smoke) | Proven |
| `c1-fire-and-forget` | **C1** | Agent fires an event; consumer handles async | `fire-and-forget` | One-way delivery through a single SLIM authority with sink observation | Same | Proven |
| `c1-write` | **C1** | Publish into the mesh without a paired responder | `write` | Ingress/write path without a bound responder process | Same | Proven |

**Scenario anchor (shared):**

| Property | Value |
|----------|--------|
| Scenario name | Local messaging smoke |
| Repo path | `benchmarks/agntcy-slim/` |
| CI task | `task benchmarks:slim:benchmark:ci:suite-smoke` |
| CI workflow | `.github/workflows/test-benchmarks-slim.yaml` → job `slim-benchmark-smoke` |
| Ginkgo label | `benchmark-suite` (via `scripts/run_suite.sh`) |
| Smoke matrix (CI) | modes: `request-reply`, `fire-and-forget`, `write`; clients: `1`; size: `16` B; duration: `1s`; repeats: `25` (see `benchmark:ci:suite-smoke` in `benchmarks/agntcy-slim/Taskfile.yml`) |

All three rows share one evidence bundle. Per-row proof is distinguished by **mode** inside that bundle (see [Row metadata](#row-metadata-dashboard-fields)).

---

## Smoke artifact contract (`slim-benchmark-smoke-report` v1)

### Producer

| Step | Location |
|------|----------|
| Run suite | `task benchmarks:slim:benchmark:ci:suite-smoke` → writes under `benchmarks/agntcy-slim/reports/` |
| Stage + render | `.github/workflows/test-benchmarks-slim.yaml` → copies into `benchmarks/agntcy-slim/published/smoke/`, runs `task benchmarks:slim:reports:dashboard` |
| Upload | Artifact name **`slim-benchmark-smoke-report`**, path `benchmarks/agntcy-slim/published/smoke`, retention 30 days |
| Pages ingest | `.github/workflows/publish-test-reports-pages.yaml` → `site/benchmarks/slim/smoke/` then merged into `site/benchmarks/slim/index.html` |

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

Each C1 row uses the same field set. Values are populated at dashboard build or publish time.

### Field definitions

| Field | Type | Description |
|-------|------|-------------|
| `row_id` | string | Stable key: `c1-request-reply`, `c1-fire-and-forget`, `c1-write` |
| `class` | string | Always `C1` |
| `mode` | string | SLIM mechanism slug: `request-reply`, `fire-and-forget`, `write` |
| `status` | enum | `verified` \| `failed` \| `unknown` — see [Status rules](#status-rules) |
| `last_run` | string | GitHub Actions run ID for workflow `test-benchmarks-slim`, job `slim-benchmark-smoke` |
| `last_run_url` | URL | `https://github.com/{owner}/{repo}/actions/runs/{run_id}` |
| `evidence_url` | URL | Primary reader link (Pages); see [URL templates](#url-templates) |
| `artifact_url` | URL | Fallback: Actions run artifact `slim-benchmark-smoke-report` |
| `rerun_cmd` | string | `task benchmarks:slim:benchmark:ci:suite-smoke` (after `slimctl` on `PATH`; see epic runbook issue) |

### Status rules

Apply in order:

1. If workflow job `slim-benchmark-smoke` **conclusion** is not `success` → `failed` for all three rows.
2. Else parse latest `MODE_SUMMARY` for the row’s `mode` from `ci-smoke.log`:
   - `MODE_SUMMARY mode=<mode> ... total_errors=0` → `verified`
   - `MODE_SUMMARY mode=<mode> ... total_errors=<n>` with `n > 0` → `failed`
3. Else parse `results.tsv` rows where column `mode` equals the row’s `mode` (CI smoke: expect 25 rows per mode for `clients=1`, `size=16`):
   - All rows have `sender_runtime_errors=0` and `sink_errors=0` → `verified`
   - Any row with non-zero errors → `failed`
4. If sources are missing → `unknown` (do not mark `verified`).

**Log line shapes** (from `benchmarks/agntcy-slim/tests/benchmark_suite_test.go`):

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

Published smoke files on Pages (after `publish-test-reports-pages`):

- `site/benchmarks/slim/smoke/` — raw bundle mirror
- `site/benchmarks/slim/index.html` — merged dashboard (includes smoke section when artifact present)

---

## Per-row source mapping

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
- [x] Artifact contract **v1** lists files produced by `.github/workflows/test-benchmarks-slim.yaml` staging step
- [x] Each row maps to documented `status`, `last_run`, `evidence_url`, `artifact_url`, `rerun_cmd` sources
- [ ] Next: implement extraction in dashboard template / runbook (`slim-c1-runbook-and-verification`, `slim-c1-dashboard-template`)

---

## References

- `benchmarks/agntcy-slim/tools/report_dashboard.go` — smoke section artifact list
- `benchmarks/agntcy-slim/README.md` — suite and CI smoke description
- Dashboard: [`../../analitics/published/index.html`](../../analitics/published/index.html)
