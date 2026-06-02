# C1 Analitics Implementation (Draft)

This directory is the base for the first C1-focused implementation slice.
It packages planning, evidence contracts, templates, and generated markdown
artifacts used to publish and review CSIT evidence for SLIM.

## Initial goal summary

Starting from the agentic systems taxonomy:

- **C1 Centralized:** a single authority controls routing/invocation.
- **C2 Decentralized:** workflow/route graph coordinates agents.
- **C3 Distributed:** federated peers across hosts/clusters.

This implementation focuses on **C1** use cases and their evidence:

- `request-reply`
- `fire-and-forget`
- `write`

Evidence source is the smoke benchmark workflow and artifacts documented in:

- `docs/plans/slim-c1-evidence-contract-v1.md`

## Scope of this scaffolding

- Define a reusable local build flow for C1 evidence markdown.
- Sync smoke markdown outputs from benchmark reports.
- Render a C1 summary markdown page for publishing/review.
- Keep implementation artifacts under this `analitics` base directory (repo root).

## Directory layout

```text
analitics/
├── README.md
├── Taskfile.yml
├── dsu-3.txt
├── test-dashboard.html
├── templates/
│   └── c1-summary.md.tmpl
├── scripts/
│   └── render-c1-summary.sh
└── published/
    ├── README.md
    ├── c1-evidence-summary.md
    └── smoke/
        └── README.md
```

## Resources

- **Task files**
  - `analitics/Taskfile.yml` (C1 sync/render workflow)
  - Existing benchmark tasks under `benchmarks/agntcy-slim/Taskfile.yml`
- **Templates**
  - `analitics/templates/c1-summary.md.tmpl`
  - HTML mockup `analitics/test-dashboard.html`
- **Published pages / markdown outputs**
  - `analitics/published/c1-evidence-summary.md`
  - `analitics/published/smoke/*.md` (synced from reports when available)

## How to run

From repo root:

```bash
task -t analitics/Taskfile.yml c1:build
```

This will:

1. create publish folders under `analitics/published/`,
2. sync smoke markdown reports from `benchmarks/agntcy-slim/reports/`,
3. render `published/c1-evidence-summary.md` from template + report data.

## Relationship to prior planning

- Epic: `docs/plans/slim-dashboard-epic.md`
- C1 contract: `docs/plans/slim-c1-evidence-contract-v1.md`

This directory is the implementation-facing layer for those planning artifacts.
