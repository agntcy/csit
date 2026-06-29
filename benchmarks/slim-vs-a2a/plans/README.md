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
  payloadBytes: 0      # optional fixed-size padding added to every finding
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
  -payload-bytes 0 \
  -output plans/sweeps/hypothesis-convergence-10ag-20ms.yaml
```

### Payload size (transport-stress) knob

`spec.payloadBytes` adds fixed-size deterministic padding to every finding. It is
semantically inert (it does not affect the consensus math or the number of
rounds) — it only inflates the wire size of each finding. Because A2A relays each
finding twice through the central runner (`agent → runner → N−1 agents`) while
SLIM publishes once and the dataplane fans out, larger payloads disproportionately
load the A2A relay. Combine it with high agent counts and a short think window to
make the multicast advantage most visible:

```bash
# Sweep agents × payload at a 5ms think window (0 vs 10kB), then build the report.
task compare:sweep:payload \
  SWEEP_AGENTS=10,50 SWEEP_THINK_MS=5 SWEEP_PAYLOAD_BYTES=0,10240
```

> Caveat: a2a-go's streaming task store appends each relayed finding to the task
> history and deep-copies the task per update, so very large payloads also amplify
> A2A cost super-linearly. That is a property of the A2A streaming/task model, but
> note it when attributing the delta.

## Run comparison

```bash
task build
task compare:plan PLAN=hypothesis-convergence-5ag-20ms
task compare:report
```

Primary metric: `consensus_wall_ms` in `reports/results.tsv`.
