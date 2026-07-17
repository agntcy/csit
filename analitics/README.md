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
        └── Evidence links (JSON report, integration docs, rerun commands)
```

## Primary output

After build, open:

- **`published/index.html`** — taxonomy dashboard (HTML)
- **`published/evidence/c1-evidence.json`** — machine-readable assertion report
- **`published/evidence/c1-evidence.md`** — human-readable report (from JSON)
- **`published/evidence/c1-evidence.html`** — standalone HTML report (from JSON)

## Build

From repo root:

```bash
# Full pipeline: run C1 tests, sync JSON, render dashboard
task -t analitics/Taskfile.yml dashboard:build

# Re-render from existing reports/c1-evidence.json (no test run)
task -t analitics/Taskfile.yml dashboard:build:only
```

`dashboard:build` runs `test:agentic-evidence` (currently `test:c1-evidence` only) — three
Ginkgo specs that assert behavioral proof for each C1 use case.

Steps:

1. Run `test:c1-evidence` → writes `reports/c1-evidence.json`
2. `dashboard:sync` copies JSON into `published/evidence/`
3. `dashboard:render` builds `published/index.html` plus `published/evidence/c1-evidence.{md,html}` from JSON

## Task reference

| Task | Purpose |
|------|---------|
| `deps:slim-bindings-setup` | Install native SLIM bindings for CGO clients |
| `deps:slimctl-download` | Download `slimctl` into `analitics/bin/` |
| `test:c1-evidence` | Run C1 Ginkgo specs only |
| `test:agentic-evidence` | Run all class evidence tests (C1 today) |
| `dashboard:build` | Tests + sync + render |
| `dashboard:build:only` | Sync + render from existing JSON |
| `dashboard:clean` | Remove generated `published/` artifacts |

## Layout

```text
analitics/
├── README.md
├── Taskfile.yml
├── go.mod
├── harness/                   # SLIM stack + client build helpers for C1 tests
├── clients/
│   ├── echo-client/           # Responder/sink helper
│   └── rate-client/           # Traffic generator
├── tests/
│   ├── suite_test.go              # Ginkgo bootstrap
│   ├── c1_evidence_test.go        # C1 use-case specs + behavioral assertions
│   ├── c1_evidence_report.go      # JSON report model + upsert helpers
│   └── c1_evidence_report_test.go # unit tests for the report helpers
├── templates/
│   └── dashboard.html.tmpl
├── scripts/
│   ├── evidence-lib.sh
│   ├── render-dashboard.sh
│   └── render-evidence-report.sh
├── bin/                       # generated (gitignored): downloaded slimctl
├── reports/                   # generated (gitignored)
└── published/                 # generated (gitignored)
    ├── index.html
    └── evidence/
        ├── c1-evidence.json
        ├── c1-evidence.md
        └── c1-evidence.html
```

## Planning references

- Epic: `docs/plans/slim-dashboard-epic.md`
- C1 contract: `docs/plans/slim-c1-evidence-contract-v1.md`

## Status evaluation (C1)

C1 use-case status is read from the `status` field of each case in
`reports/c1-evidence.json` (produced by Ginkgo tests in `tests/c1_evidence_test.go`):

- `verified` — the mode's behavioral assertions passed; the test records this case as `verified`
- `failed` — the case is recorded with a `failed` status
- `unknown` — no case entry exists for the mode (e.g. the report is missing or the spec did not run)

The render step reads these values as-is; it does not re-run or re-evaluate the tests.

C2/C3 rows use static status until their test evidence is wired into the build flow.

## GitHub Pages

On `main`, the `test-agentic-evidence` workflow publishes the generated dashboard to the shared
CSIT test reports site under **`docs/agentic-evidence/`** on the `gh-pages` branch (linked from
`docs/index.html` as **Agentic evidence dashboard**). C1 class proof remains in
`published/evidence/c1-evidence.json`.

## Prerequisites

- Go 1.24+ (matches `go.mod`)
- `task` in the shell
- `jq` and `bash` on `PATH` (used by the render scripts)
- Native SLIM bindings for CGO (`task -t analitics/Taskfile.yml deps:slim-bindings-setup`)
- `slimctl` on `PATH`, or install via `task -t analitics/Taskfile.yml deps:slimctl-download`
