# SLIM analitics — evidence dashboard

This directory builds a **static HTML evidence dashboard** organized by agentic-system
taxonomy (C1 / C2 / C3). Each class section lists use cases; each use case links to
test-derived evidence artifacts.

## Naming

| Term | Scope | Examples |
|------|-------|----------|
| **`agentic-evidence`** | Umbrella: dashboard pipeline, CI, GitHub Pages | `test-agentic-evidence` workflow, `docs/agentic-evidence/` on `gh-pages`, `agentic-evidence-dashboard` artifact |
| **`c{N}-evidence`** | Class-specific tests and reports (C1 today; C2/C3 later) | Ginkgo label `c1-evidence`, `reports/c1-evidence.json`, `test:c1-evidence` task |
| **`analitics/`** | Implementation root (Go module, templates, scripts) | This directory; not the public Pages URL slug |

## Structure

```text
Class (C1 / C2 / C3)
  └── Use cases (table rows)
        └── Evidence links (markdown reports, integration docs, rerun commands)
```

## Primary output

After build, open:

- **`published/index.html`** — generated static dashboard

Optional:

- **`published/c1-evidence-summary.md`** — C1-only markdown summary
- **`published/evidence/c1-evidence.json`** — synced Ginkgo assertion report

## Build

From repo root:

```bash
# Full pipeline: run C1 Ginkgo evidence tests, then render dashboard
task -t analitics/Taskfile.yml dashboard:build

# Fast rebuild from existing evidence report (no test run)
task -t analitics/Taskfile.yml dashboard:build:only
```

`dashboard:build` runs `test:c1-evidence` — three Ginkgo specs in `tests/`
that assert behavioral proof for each C1 use case (not the benchmark throughput matrix).

Steps:

1. Run `test:c1-evidence` → writes `reports/c1-evidence.json`
2. Sync JSON into `published/evidence/`
3. Evaluate C1 row status from `c1-evidence.json`
4. Render `published/index.html`

## Layout

```text
analitics/
├── README.md
├── Taskfile.yml
├── go.mod
├── harness/                   # SLIM stack + client build helpers for C1 tests
├── clients/
│   ├── echo-client/           # Responder/sink helper (shared with benchmarks)
│   └── rate-client/           # Traffic generator (shared with benchmarks)
├── tests/
│   ├── c1_evidence_test.go
│   └── c1_evidence_report.go
├── test-dashboard.html          # legacy mockup reference
├── templates/
│   ├── dashboard.html.tmpl    # static HTML source template
│   └── c1-summary.md.tmpl
├── scripts/
│   ├── evidence-lib.sh
│   ├── render-dashboard.sh
│   └── render-c1-summary.sh
├── reports/                   # generated (gitignored)
└── published/                 # generated (gitignored)
    ├── index.html
    ├── c1-evidence-summary.md
    └── evidence/
```

## Planning references

- Epic: `docs/plans/slim-dashboard-epic.md`
- C1 contract: `docs/plans/slim-c1-evidence-contract-v1.md`

## Status evaluation (C1)

C1 use-case status is derived from `reports/c1-evidence.json`
(produced by Ginkgo tests in `tests/c1_evidence_test.go`):

- `verified` — assertion-based test passed for the mode
- `failed` — test failed or case reports non-zero errors
- `unknown` — no case entry for the mode

C2/C3 rows use static status until their test evidence is wired into the build flow.

## GitHub Pages

On `main`, the `test-agentic-evidence` workflow publishes the generated dashboard to the shared
CSIT test reports site under **`docs/agentic-evidence/`** on the `gh-pages` branch (linked from
`docs/index.html` as **Agentic evidence dashboard**). C1 class proof remains in
`published/evidence/c1-evidence.json`.

## Prerequisites

- Go 1.22+
- `task` in the shell
- Native SLIM bindings for CGO (`task -t analitics/Taskfile.yml deps:slim-bindings-setup`)
- `slimctl` on `PATH`, or install via `task -t analitics/Taskfile.yml deps:slimctl-download`
