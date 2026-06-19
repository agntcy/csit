# Consensus scenarios (v2)

YAML `ConsensusScenario` plans for the SLIM vs A2A v2 consensus streaming benchmark.
Each scenario defines N agents that run a deterministic **distributed hypothesis convergence**
workload: agents think in parallel, emit findings, and must reach identical local consensus.

## Topologies compared

The two implementations highlight one structural difference:

- **SLIM (`slim-group-session`)** — agents join a native SLIM group session and
  broadcast findings peer-to-peer; the dataplane fans them out. **There is no
  application relay.** The runner only moderates (creates the session, invites
  agents, sends one start) and then passively observes snapshots that agents
  push to it on convergence.
- **A2A (`a2a-relay-stream`)** — A2A has no peer multicast, so the runner is an
  explicit **relay hub**. Two server-streaming legs carry findings: agents stream
  findings to the runner (`OpStreamFindings`) and subscribe to the runner's
  relayed-finding stream (`OpStreamRelay`). Every finding therefore makes an
  extra relay hop, which is the overhead `consensus_wall_ms` is expected to show.

## Schema

```yaml
apiVersion: bench.agntcy.io/v2
kind: ConsensusScenario
metadata:
  name: hypothesis-convergence-10agents-10ms
  domain: hypothesis-convergence
spec:
  agents: 10
  thinkTimeMs: 10
  findingEmitDelayMs: 1
  maxRounds: 200
  targetMode: majority
  seed: 42
  valueSpace: 3
agents:
  - id: agent-0
    slimName: agntcy/bench-v2/agent-0
    a2aPort: 9700
    role: worker        # legacy field; both transports now treat all agents as peers
  - id: agent-1
    slimName: agntcy/bench-v2/agent-1
    a2aPort: 9711
    role: worker
```

## Generate sweep scenarios

```bash
go run ./tools/gen_scenario \
  -family hypothesis-convergence \
  -agents 10 \
  -think-ms 20 \
  -output plans/sweeps/hypothesis-convergence-10ag-20ms.yaml
```

## Run comparison

```bash
task build
task compare:plan PLAN=hypothesis-convergence-5ag-20ms
task compare:report
```

Primary metric: `consensus_wall_ms` in `reports/results.tsv`.
