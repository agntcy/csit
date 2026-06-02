# SLIM analitics — evidence dashboard

This directory builds a **static HTML evidence dashboard** organized by agentic-system
taxonomy (C1 / C2 / C3). Each class section lists use cases; each use case links to
test-derived evidence artifacts.

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
- **`published/smoke/*.md`** — synced smoke benchmark reports

## Build

From repo root:

```bash
task -t analitics/Taskfile.yml dashboard:build
```

Steps:

1. Sync smoke markdown from `benchmarks/agntcy-slim/reports/`
2. Evaluate C1 row status from `results.tsv`
3. Render `published/index.html` from `templates/dashboard.html.tmpl`

## Layout

```text
analitics/
├── README.md
├── Taskfile.yml
├── test-dashboard.html          # legacy mockup reference
├── templates/
│   ├── dashboard.html.tmpl    # static HTML source template
│   └── c1-summary.md.tmpl
├── scripts/
│   ├── evidence-lib.sh
│   ├── render-dashboard.sh
│   └── render-c1-summary.sh
└── published/
    ├── index.html             # generated dashboard
    ├── c1-evidence-summary.md
    └── smoke/
```

## Planning references

- Epic: `docs/plans/slim-dashboard-epic.md`
- C1 contract: `docs/plans/slim-c1-evidence-contract-v1.md`

## Status evaluation (C1)

C1 use-case status is derived from `benchmarks/agntcy-slim/reports/results.tsv`:

- `verified` — all rows for the mode have zero sender/sink errors
- `failed` — any row has errors
- `unknown` — no rows for the mode

C2/C3 rows use static status until their test evidence is wired into the build flow.
