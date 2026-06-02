# C1 Evidence Summary

**Generated at:** 2026-06-02T16:29:20Z  
**Scope:** C1 use cases only (`request-reply`, `fire-and-forget`, `write`)  
**Source contract:** `docs/plans/slim-c1-evidence-contract-v1.md`

## Goal

Summarize C1 evidence for SLIM from the local smoke benchmark report set, aligned
to the agentic systems taxonomy and the C1 use-case contract.

## Taxonomy context (condensed)

- **C1 Centralized:** one authority controls routing/invocation.
- **C2 Decentralized:** workflow graph coordinates communication.
- **C3 Distributed:** federated peers across hosts/clusters.

This summary intentionally covers only **C1**.

## C1 use-case status

| Row ID | Use case | Mechanism | Status | Runs seen | Evidence |
|--------|----------|-----------|--------|-----------|----------|
| `c1-request-reply` | Agent waits for reply | `request-reply` | verified | 12 | `published/smoke/suite_summary.md` |
| `c1-fire-and-forget` | Agent emits async event | `fire-and-forget` | verified | 12 | `published/smoke/suite_summary.md` |
| `c1-write` | Publish without paired responder | `write` | verified | 12 | `published/smoke/suite_summary.md` |

## Evidence sources

- `benchmarks/agntcy-slim/reports/results.tsv`
- `benchmarks/agntcy-slim/reports/ci-smoke.log`
- `benchmarks/agntcy-slim/reports/suite_summary.md`
- `benchmarks/agntcy-slim/reports/technical_report.md`

## Re-run command

```bash
task benchmarks:slim:deps:slimctl-download SLIMCTL_PATH="$HOME/.local/bin/slimctl"
export PATH="$HOME/.local/bin:$PATH"
task benchmarks:slim:benchmark:ci:suite-smoke
```
