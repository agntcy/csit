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
  payloadBytes: 0        # optional fixed-size padding added to every finding
  epochs: 10             # consensus attempts per run (default 10)
  maxEpochTimeMs: 120000 # per-epoch consensus budget; exceeding it fails the epoch (default 120000)
agents:
  - id: agent-0
    slimName: agntcy/bench-v2/agent-0
    a2aPort: 9700
    role: worker        # legacy field; both transports now treat all agents as peers
  - id: agent-1
    slimName: agntcy/bench-v2/agent-1
    a2aPort: 9711
    role: worker
    latencyMs: 100      # optional per-agent one-way network delay (distant region)
```

## Generate sweep scenarios

```bash
go run ./tools/gen_scenario \
  -family hypothesis-convergence \
  -agents 10 \
  -think-ms 20 \
  -payload-bytes 0 \
  -epochs 10 \
  -max-epoch-ms 120000 \
  -output plans/sweeps/hypothesis-convergence-10ag-20ms.yaml
```

### Epochs (reliability knob)

Each run repeats the *same* consensus attempt over `spec.epochs` epochs. An epoch
succeeds if every agent reaches global consensus within `spec.maxEpochTimeMs`
(each attempt still runs up to `maxRounds`); otherwise it is counted as a failed
epoch. Between epochs the agents reset their engine in place and re-run — the SLIM
group session and A2A relay subscriptions stay up for the whole run, so there is a
single teardown. The runner reports `epochs`, `epochs_succeeded`, and
`epochs_failed`; overall `success` is true only when no epoch failed. This
surfaces reliability differences under load: A2A may miss the per-epoch budget
while SLIM still converges.

Sweeps expose two variables (defaults shown): `SWEEP_EPOCHS=10` and
`SWEEP_MAX_EPOCH_MS=30000` (a modest budget keeps failed A2A epochs capping
quickly). For example:

```bash
task compare:sweep:payload \
  SWEEP_AGENTS=10,50 SWEEP_THINK_MS=5 SWEEP_PAYLOAD_BYTES=0,10240 \
  SWEEP_EPOCHS=10 SWEEP_MAX_EPOCH_MS=30000
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

### Selective agent latency (distant-region) knob

`agents[].latencyMs` adds a per-agent one-way network delay, simulating members
hosted in a distant region. Set it on a subset of agents (the sweep tool stamps
the trailing ~1/3, keeping the coordinator local) to model a geo-distributed
group. Generate latency variants with:

```bash
go run ./tools/gen_scenario \
  -family hypothesis-convergence -agents 30 -think-ms 20 \
  -payload-bytes 10240 -epochs 5 -max-epoch-ms 5000 \
  -latency-ms 100 \        # one-way delay stamped on the laggy subset
  -latency-count 10 \      # optional; defaults to round(agents/3)
  -output plans/sweeps/hypothesis-convergence-30ag-10240b-lat100.yaml
```

Two injection models are available via the `BENCH_LAT_MODEL` env var (zero-latency
scenarios are unaffected by the choice):

- `relay` (**default**) — latency is charged at the network hop. The A2A relay
  pays each distant agent's delay on **both** legs (agent→relay and relay→agent)
  during its sequential fan-out, while native SLIM multicast pays it **once**, in
  parallel. This models the real 1-hop-vs-2-hop cost of a central relay for
  geo-distributed members and is the shipped default because it is deterministic
  (both transports keep every epoch green; the difference is purely wall time).
- `boundary` — latency is applied at each agent's own send/receive boundary
  (non-blocking), identical for both transports; differentiation then comes only
  from topology/backpressure and mostly appears at high agent counts.

The committed latency plans (`*-lat100`, order 7–9) hold the CI-safe base params
(`thinkTimeMs: 20`, `epochs: 5`, `maxEpochTimeMs: 5000`) so they stay within the
suite's time budget.

## Run comparison

```bash
task build
task compare:plan PLAN=hypothesis-convergence-5ag
task compare:report
```

Primary metric: `consensus_wall_ms` in `reports/results.tsv`.
