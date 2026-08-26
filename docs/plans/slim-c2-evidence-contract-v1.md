# C2 evidence contract (v1) — topology routing slice

**Status:** Draft for dashboard slice 2 (first C2 row)  
**Epic:** [`slim-dashboard-epic.md`](slim-dashboard-epic.md)  
**C1 reference:** [`slim-c1-evidence-contract-v1.md`](slim-c1-evidence-contract-v1.md)

This document locks the first **C2** dashboard row (`c2-topology-routing`) and its evidence source. The analitics dashboard reads `c2-evidence.json` for row status; the Ginkgo HTML report under `docs/slim-integration/` remains supplementary detail.

---

## C2 use-case matrix (slice 2)

| Row ID | Class | Use case (reader) | SLIM mechanism | CSIT scenario | Status source |
|--------|-------|-------------------|----------------|---------------|---------------|
| `c2-topology-routing` | **C2** | Multi-agent flow over fixed, named routes | `declarative-routes` | TopologyTest Ginkgo (`integrations/agntcy-slim/topology/tests`) | `c2-evidence.json` |
| `c2-workflow-app` | **C2** | Multi-step app driven by workflow manager | `workflow + SLIM` | `integrations/agntcy-apps` (future) | static pending |

---

## Scenario anchor (topology routing)

| Property | Value |
|----------|--------|
| Scenario name | TopologyTest Ginkgo |
| Repo path | `integrations/agntcy-slim/topology/` |
| CI workflow | `.github/workflows/test-slim-integration.yaml` |
| CI task | `task integrations:slim:test:topology` |
| Ginkgo focus | `Slim topology test` |
| Evidence output | `integrations/agntcy-slim/topology/reports/c2-evidence.json` |
| Detail report | `integrations/agntcy-slim/topology/reports/index.html` → Pages `docs/slim-integration/` |

### C2 routing scenarios (in `c2-evidence.json`)

| Scenario key | TopologyTest `When`/`Then` | Proof |
|--------------|----------------------------|-------|
| `isolated-routes` | Segments isolate b↔c; alice→bob and alice→carol | End-to-end delivery + negative isolation + link APPLIED state |
| `linked-routes` | Add b↔c link | Bob→carol delivery after graph change |
| `route-survives-restart` | Gateway pod restart on b↔c | Delivery after dataplane failover |

**Out of scope for C2 row:** control-plane restart link persistence (reserved for future **C3** evidence).

---

## `c2-evidence.json` schema (v1)

### Producer

| Step | Location |
|------|----------|
| Run topology suite | `task integrations:slim:test:topology:run` with `C2_EVIDENCE_REPORT_DIR=<topology>/reports` |
| Writer | `integrations/agntcy-slim/topology/tests/c2_evidence_report.go` |
| Dashboard ingest | `analitics/scripts/evidence-lib.sh` → `row_status "c2-topology-routing" 3` |

### Required fields per case

| Field | Role |
|-------|------|
| `row_id` | Stable key (`c2-topology-routing`) |
| `scenario` | Scenario slug (`isolated-routes`, `linked-routes`, `route-survives-restart`) |
| `mechanism` | `declarative-routes` |
| `status` | `verified` when the Ginkgo spec passes |
| `assertions` | Human-readable behavioral proof lines |

### Row status rules (dashboard)

Apply in order (`analitics/scripts/evidence-lib.sh`):

1. Read `c2-evidence.json` (from `analitics/reports/` or synced from topology reports).
2. Filter cases where `row_id == c2-topology-routing`.
3. Any `failed` → row **Failed**.
4. Fewer than 3 `verified` scenarios and none failed → row **Partial**.
5. Exactly 3 `verified` scenarios → row **Verified**.
6. Missing file or cases → row **Unknown**.

**C2 class badge:** **Partial** when topology row is verified (workflow row still pending); **Failed** if topology row failed; **Unknown** otherwise.

---

## Local workflow

```bash
# 1. Run topology test (KinD + controller + clusters required)
task integrations:slim:test:topology

# 2. Build dashboard (auto-syncs c2-evidence.json from topology reports)
task -t analitics/Taskfile.yml dashboard:build:only

# 3. Open
open analitics/published/index.html
```

Manual sync override:

```bash
cp integrations/agntcy-slim/topology/reports/c2-evidence.json analitics/reports/
C2_EVIDENCE_SOURCE=analitics/reports/c2-evidence.json task -t analitics/Taskfile.yml dashboard:build:only
```

---

## Pages / CI wiring

| Artifact | Contents |
|----------|----------|
| `slim-integration-test-result` | `report-slim-topology.json`, `c2-evidence.json`, `index.html`, … |
| `agentic-evidence-dashboard` | `published/index.html`, `published/evidence/c1-evidence.*`, `published/evidence/c2-evidence.*` |

The `test-agentic-evidence` workflow attempts to download `slim-integration-test-result` from the same commit before rendering so C2 status appears on `docs/agentic-evidence/` when both suites ran on that SHA.

---

## References

- [`TopologyTest.md`](../../integrations/agntcy-slim/topology/TopologyTest.md)
- Dashboard template: `analitics/templates/dashboard.html.tmpl`
- Topology evidence writer: `integrations/agntcy-slim/topology/tests/topology_test.go`
