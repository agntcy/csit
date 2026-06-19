# Execution plans (Phase 1)

Hand-authored YAML DAG plans for the SLIM vs A2A multi-agent benchmark.
Each plan defines named agents, interdependent tasks (mocked via timing fields),
optional failure injection, and context updates that exercise sync / context
sharing / failure propagation.

## Plans

| File | Domain | Agents | Tasks | Notes |
|------|--------|--------|-------|-------|
| [k8s-incident-response.yaml](domains/k8s-incident-response.yaml) | Kubernetes troubleshooting | 6 | 16 | OOM diagnosis; `probe-service-mesh` injects failure; 2 context updates |
| [urban-traffic-reroute.yaml](domains/urban-traffic-reroute.yaml) | Realtime traffic | 6 | 14 | I-5 closure; N/S/E/W parallel route recalc; detour cancel on ETA revision |
| [mobile-nav-assistant.yaml](domains/mobile-nav-assistant.yaml) | Android voice navigation | 6 | 12 | Pharmacy→airport intent; coffee-first amendment obsoletes pharmacy branch |

## Schema

```yaml
apiVersion: bench.agntcy.io/v1
kind: ExecutionPlan
metadata:
  name: <plan-id>
  domain: <domain-label>
  description: <text>
spec:
  defaults:
    completionTimeSec: <float>
    maxCompletionTimeSec: <float>
  maxRetries: <int>
agents:
  - id: <role-id>
    slimName: agntcy/bench/<role-id>
    a2aPort: <port>          # unique within plan; k8s=91xx, traffic=92xx, mobile=93xx
tasks:
  - id: <task-id>
    name: <human-readable label>
    agent: <role-id>
    dependsOn: [<task-id>, ...]
    completionTimeSec: <float>
    maxCompletionTimeSec: <float>
    injectFailure: <bool>    # optional; default false
    output: <mock result string>
contextUpdates:              # optional
  - afterTask: <task-id>
    payload: <context string>
    targetAgents: [<role-id>, ...]
```

## Port allocation

Ports are unique per plan to allow isolated local runs:

- **k8s-incident-response:** A2A gRPC `9101`–`9106`
- **urban-traffic-reroute:** A2A gRPC `9201`–`9206`
- **mobile-nav-assistant:** A2A gRPC `9301`–`9306`

SLIM identities use `agntcy/bench/<agent-id>` from each plan's `agents[].slimName`.
