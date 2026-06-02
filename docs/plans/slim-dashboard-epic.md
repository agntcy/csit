# Epic: SLIM value through agentic-system taxonomy (→ dashboard evidence)

**Status:** Working draft — not final. Taxonomy **v0.2** uses three classes (centralized, decentralized, distributed). Refine via discussion before July dashboard freeze.

**End goal:** Publish `site/benchmarks/slim/index.html` so readers see which **classes of agentic systems** and **use cases** SLIM addresses, backed by CSIT evidence — not benchmark leaderboards.

**Context:** [Discussion #195](https://github.com/agntcy/csit/discussions/195) · deadline **2026-07-31** · dashboard: [`../../analitics/published/index.html`](../../analitics/published/index.html)

---

## Workstream flow

| Phase | Question | Output |
|-------|----------|--------|
| **0** | How do we classify agentic systems? | Taxonomy (below) — refinable `v0.x` |
| **1** | Per class, which use cases does SLIM tackle? | Use-case × SLIM matrix |
| **2** | What does CSIT prove today? | Dashboard rows + scenario cards |

Performance benchmarks → **appendix only** (existing `benchmarks/agntcy-slim/tools/report_dashboard.go`).

---

## Phase 0 — Taxonomy of agentic systems (v0.2)

Three categories by **who controls message flow and agent coordination** — not by LLM vendor, language, or single vs multi-agent count alone.

### C1 — Centralized

| Aspect | Description |
|--------|-------------|
| **Definition** | A **single authority** (runtime, broker, or control plane) owns routing, discovery, or invocation. Agents connect **through** that authority; they do not independently negotiate peers or paths. |
| **Coordination** | Hub-and-spoke or star: one SLIM node / controller entry, one namespace, or one orchestrator process dispatches work. |
| **Typical needs** | Stable named endpoints, predictable **request-reply**, ingress (**write**), simple ops model. |
| **SLIM relevance** | Strong fit for **local node + controller** patterns: modes, named identities, controller registration. |
| **Examples** | Single-cluster agent + tools on one SLIM node; agent asks a remote service via one gateway; CSIT **local messaging smoke** (`request-reply`, `fire-and-forget`, `write` on one `slimctl` node). |
| **CSIT anchors (draft)** | `benchmarks/agntcy-slim/` smoke; `integrations/agntcy-slim` sanity tests. |

---

### C2 — Decentralized (workflow managers)

| Aspect | Description |
|--------|-------------|
| **Definition** | **No single global brain**, but coordination is **scripted**: a workflow manager, scenario engine, or **declared route graph** defines who talks to whom and in what order. Agents follow the workflow; autonomy is bounded by the graph. |
| **Coordination** | Pipeline, fan-out/fan-in, or multi-step campaigns — often **TopologyTest-style** routers (`slim-0` forwarding to `slim-1` / `slim-2`) or external WFSM driving steps. |
| **Typical needs** | Multi-hop **routing by name**, repeatable scenario deploy, log/assertion proof that messages reached the right agent. |
| **SLIM relevance** | Strong fit for **declarative routes** + controller API (`slimctl` routes/links) on a **known topology**. |
| **Examples** | Alice waits, Bob sends ten hellos through intermediate SLIM servers ([`TopologyTest.md`](../../integrations/agntcy-slim/TopologyTest.md)); marketing-campaign style multi-agent email flow (`integrations/agntcy-apps`). |
| **CSIT anchors (draft)** | `integrations/agntcy-slim` topology tests; generated Helm from topology YAML. |

---

### C3 — Distributed (fully autonomous agents)

| Aspect | Description |
|--------|-------------|
| **Definition** | Agents (or dataplanes) operate as **peers** across **hosts, clusters, or org boundaries** with **no standing workflow script** defining every hop. Paths may be established via **control-plane links** and policy, not a fixed campaign file. |
| **Coordination** | Federation: downstream nodes register, links **APPLIED** across clusters; peers may change over time; emphasis on **mesh membership** and durability, not a single WFSM run. |
| **Typical needs** | Multi-cluster **Connected** nodes, cross-cluster links, control-plane survival (e.g. link ID after restart). |
| **SLIM relevance** | Strong fit for **controller + multicluster dataplane** story; routing may combine with C2 but **proof** is membership and link state, not log assertions on a fixed script. |
| **Examples** | Dataplane on cluster B registers with controller on cluster A; **APPLIED** link B→A; link stable after controller rollout ([`kind-slim-multicluster`](../../integrations/kind-slim-multicluster/)); future: peer-to-peer configs in `integrations/agntcy-slim/config/peer-to-peer.yaml`. |
| **CSIT anchors (draft)** | `kind-slim-multi-host/` + `integrations/kind-slim-multicluster/` (July distributed scenario **B1**). |

---

### Taxonomy comparison (at a glance)

| | **C1 Centralized** | **C2 Decentralized** | **C3 Distributed** |
|--|-------------------|----------------------|----------------------|
| **Who decides the path?** | Central authority | Workflow / route graph | Peers + control plane (policy) |
| **Deployment shape** | Often single runtime / cluster | Multi-node, **known** topology | Multi-cluster / multi-host |
| **Autonomy** | Low | Medium (bounded by workflow) | High (fully autonomous agents) |
| **Primary SLIM proof (July)** | Messaging modes, delivery | Topology routing (partial/Aug) | Multicluster link + node state |

**Refinement notes (open):**

- A deployment can span classes (e.g. decentralized workflow **on** distributed infrastructure) — dashboard should tag **primary class per use case**, not per deployment.
- A2A cross-SDK interop is **adjacent** (class “interop”) — separate Directory/A2A dashboards per #195; not expanded here.

---

## Phase 1 — Use cases per class (draft matrix v0.2)

| Use case | Class | SLIM mechanism | CSIT scenario (candidate) | July status |
|----------|-------|----------------|---------------------------|-------------|
| Agent A calls B and waits for reply | C1 | `request-reply` | `benchmarks/agntcy-slim` smoke | Proven |
| Agent fires event; consumer handles async | C1 | `fire-and-forget` | Same | Proven |
| Publish into mesh without paired responder | C1 | `write` | Same | Proven |
| Multi-agent scenario with fixed routes (Alice/Bob) | C2 | Declarative routes + servers | `integrations/agntcy-slim` topology | Partial / Aug |
| Multi-step app driven by workflow manager | C2 | SLIM under workflow steps | `integrations/agntcy-apps` | Gap / narrative only |
| Downstream cluster joins controller mesh | C3 | Node **Connected**, link **APPLIED** | `kind-slim-multicluster` | Proven (July) |
| Control plane survives rollout; same link ID | C3 | Link persistence | Same suite | Proven |
| Autonomous peers across hosts (no fixed WFSM) | C3 | Federation + optional routes | Multicluster and/or peer-to-peer config | Refine |

**C1 rows (frozen):** see [`slim-c1-evidence-contract-v1.md`](slim-c1-evidence-contract-v1.md). **Dashboard:** [`analitics/published/index.html`](../../analitics/published/index.html).

**Deliverable (next iteration):** expand C2/C3 rows and gaps; C1 slice is frozen in [`slim-c1-evidence-contract-v1.md`](slim-c1-evidence-contract-v1.md).

---

## Phase 2 — Dashboard (what to display)

**Reader question:** For **centralized / decentralized / distributed** agentic systems, does CSIT show SLIM solving a real use case — and can I rerun it?

### Block 1 — Context

- One paragraph: SLIM as messaging for agentic systems across **three classes** (table above, compact).
- Link to this doc + taxonomy version badge (`v0.2 draft`).
- Last updated + CI run links.

### Block 2 — Use-case evidence grid (primary)

Columns: **Use case** · **Class (C1/C2/C3)** · **SLIM fit** · **Status** · **Evidence**

Rows come from Phase 1 matrix only.

### Block 3 — Scenario cards

| Scenario | Classes served | Use cases proved |
|----------|----------------|------------------|
| Local messaging smoke | **C1** | Reply-wait, async event, write ingress |
| Multicluster controller link (July B1) | **C3** | Cross-cluster join, APPLIED link, restart stability |
| Topology routing (optional B2 / Aug) | **C2** | Named multi-hop delivery |

Each card: topology sketch, pass/fail, rerun command, **class tags**.

### Block 4 — Performance appendix (collapsed)

Existing benchmark HTML; disclaimer: perf supports capacity planning, not **class-level** adoption rationale.

---

## Refinement loop

1. Review taxonomy v0.2 (this doc) in discussion.  
2. Adjust definitions or examples → bump `v0.3`.  
3. Grow use-case matrix; mark July vs backlog.  
4. Update dashboard mockup columns to **C1 / C2 / C3**.  
5. Run CSIT; update **Status** only.  
6. July freeze: taxonomy **v1.0 for dashboard purposes**.

---

## First implementation slice (C1-focused)

Goal: ship a minimum, rerunnable C1 evidence loop that produces linkable artifacts and a clean dashboard shell without waiting for C2/C3 expansion.

**Implementation base directory:** `analitics/` (repo root)  
**Generated dashboard:** `analitics/published/index.html`

### Scope boundaries (slice 1)

- In scope: C1 rows only (`request-reply`, `fire-and-forget`, `write`) + C1 rerun workflow + report links + initial dashboard template.
- Out of scope: C2/C3 scenario cards as required evidence (keep placeholders only).
- Success criteria: one PR can regenerate C1 evidence and publish/update the C1 dashboard block with stable links.

### Workstreams and dependent issues

Execution detail lives in GitHub issues (titles, bodies, acceptance criteria). This epic tracks **outcomes** and **dependencies** only.

| Workstream | Outcome (epic-level) | Dependent issue |
|------------|----------------------|-----------------|
| C1 scope + evidence contract | 3 C1 rows frozen; artifact + row-metadata contract `v1` | `slim-c1-scope-and-evidence-contract` → [`slim-c1-evidence-contract-v1.md`](slim-c1-evidence-contract-v1.md) |
| Runbook + verification | Repeatable smoke rerun; reviewer checklist | `slim-c1-runbook-and-verification` |
| Dashboard template | Taxonomy-structured static HTML; C1 live status | `slim-c1-dashboard-template` → `analitics/templates/dashboard.html.tmpl` |
| Pages / link wiring | Published + fallback links stable per row | `slim-c1-pages-wiring` |

**Critical path:** scope/contract → (runbook ∥ template scaffold) → template finalize + pages validation.

Broader epic issues (taxonomy, full matrix, July gate) remain separate — see [Suggested issues](#suggested-issues).

### Supporting test workflows in CSIT (C1)

Primary (CI-equivalent smoke):

```bash
task benchmarks:slim:deps:slimctl-download SLIMCTL_PATH="$HOME/.local/bin/slimctl"
export PATH="$HOME/.local/bin:$PATH"
task benchmarks:slim:benchmark:ci:suite-smoke
```

Expected C1 outputs:

- `benchmarks/agntcy-slim/reports/results.tsv`
- `benchmarks/agntcy-slim/reports/suite_summary.md`
- `benchmarks/agntcy-slim/reports/technical_report.md`
- `benchmarks/agntcy-slim/reports/ci-smoke.log`

Local dashboard render from outputs:

```bash
task benchmarks:slim:reports:dashboard \
  SMOKE_DIR="benchmarks/agntcy-slim/reports" \
  CAPACITY_DIR="benchmarks/agntcy-slim/reports" \
  OUTPUT_FILE="benchmarks/agntcy-slim/reports/index.html"
```

Workflow anchors:

- CI producer: `.github/workflows/test-benchmarks-slim.yaml` (`slim-benchmark-smoke` job)
- Artifact name: `slim-benchmark-smoke-report`
- Pages publisher: `.github/workflows/publish-test-reports-pages.yaml`

### C1 evidence contract (v1)

Frozen in [`slim-c1-evidence-contract-v1.md`](slim-c1-evidence-contract-v1.md):

- three C1 rows (`c1-request-reply`, `c1-fire-and-forget`, `c1-write`)
- `slim-benchmark-smoke-report` required/optional files
- per-row metadata: `status`, `last_run`, `evidence_url`, `artifact_url`, `rerun_cmd` with extraction rules (`MODE_SUMMARY`, `results.tsv`, workflow conclusion)

### Clean initial dashboard template (C1-first)

Template structure in `analitics/templates/dashboard.html.tmpl` (output: `analitics/published/index.html`):

1. **Header/context**
   - short statement of C1 scope for slice 1
   - last updated timestamp
   - links to benchmark workflow run and discussion
2. **C1 evidence grid (primary block)**
   - rows: request-reply / fire-and-forget / write
   - columns: use case, SLIM fit, status, evidence
3. **C1 scenario card**
   - topology: single-node smoke
   - rerun command
   - artifact bundle links
4. **Planned next blocks**
   - compact C2/C3 placeholders marked “planned”
5. **Performance appendix**
   - collapsed section, explicitly non-primary rationale

### 3-person parallel execution

Three lanes, one handoff (contract `v1` from Lane A). Issue bodies and acceptance criteria live in GitHub only.

| Lane | Owner | Dependent issue(s) | Start now? | Handoff from |
|------|-------|--------------------|------------|--------------|
| **A — Evidence contract** | Person 1 | `slim-c1-scope-and-evidence-contract`; `slim-c1-runbook-and-verification` | Yes | — |
| **B — Dashboard template** | Person 2 | `slim-c1-dashboard-template` | Yes (stub fields) | Lane A |
| **C — Pages / links** | Person 3 | `slim-c1-pages-wiring` | Yes (mapping draft) | Lane A, then Lane B |

**Sequence:** kickoff (row + field names) → parallel block 1 → contract `v1` handoff → parallel block 2 → checklist sign-off.

**Slice done when:** all C1 rows render with status + links; smoke bundle reruns cleanly; reviewer can run checklist without author help.

---

## Timeline

| When | Outcome |
|------|---------|
| **2026-05-31** | Taxonomy v0.2 agreed for drafting; matrix v0.2 started |
| **2026-06-30** | July use cases locked per class; distributed scenario B1 vs B2 |
| **2026-07-31** | Published Pages aligned to matrix; gaps → August |

---

## Acceptance criteria

- [ ] Three-class taxonomy documented with definitions and examples  
- [ ] Each July dashboard row maps to **class + use case**  
- [ ] Reader can answer which **class** SLIM is demonstrated for  
- [ ] Gaps explicit; perf only in appendix  
- [ ] Published `site/benchmarks/slim/index.html`

---

## Suggested issues

Epic-wide (not C1-slice-specific):

| Issue | Focus |
|-------|--------|
| `slim-taxonomy-v0.2` | Review C1/C2/C3 definitions |
| `slim-use-case-matrix` | Phase 1 matrix |
| `slim-dashboard-use-case-grid` | Block 2 spec |
| `slim-dashboard-mockup-update` | C1/C2/C3 in `test-dashboard.html` |
| `slim-july-publish-checklist` | #195 gate |

C1 first slice (tracked as dependent issues — details in GitHub, not duplicated here):

| Issue | Focus |
|-------|--------|
| `slim-c1-scope-and-evidence-contract` | C1 rows + smoke artifact + row metadata contract |
| `slim-c1-runbook-and-verification` | Rerun commands + verification checklist |
| `slim-c1-dashboard-template` | C1-first dashboard template |
| `slim-c1-pages-wiring` | Pages link mapping + validation |

---

## Repo anchors

- [`slim-c1-evidence-contract-v1.md`](slim-c1-evidence-contract-v1.md) — C1 rows + artifact/metadata contract  
- `benchmarks/agntcy-slim/` — C1 proof  
- `integrations/agntcy-slim/TopologyTest.md` — C2 proof  
- `kind-slim-multi-host/`, `integrations/kind-slim-multicluster/` — C3 proof  
- `integrations/agntcy-apps/` — C2 example (workflow-driven)  
- `.github/workflows/publish-test-reports-pages.yaml` — Pages publish  
