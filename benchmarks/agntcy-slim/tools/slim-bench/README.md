# SLIM Benchmark Tool & Suite

`slim-bench` is a load generator and benchmarking tool for the SLIM (Secure Light-weight Instant Messaging) protocol. It measures latency (Min, P50, P99, Max), throughput, and runtime errors across payload sizes, client counts, and input rates.

The suite is automated by a Ginkgo benchmark spec in `benchmarks/agntcy-slim/tests/benchmark_suite_test.go`. The `run_suite.sh` script is only a thin launcher for that spec.

## Components

1.  **slim-bench**: The core Go binary. It connects to the SLIM Data Plane, generates traffic, and calculates statistics.
2.  **benchmark_suite_test.go**: The Ginkgo-based suite runner. It starts a local SLIM node plus echo responder, runs the benchmark matrix, and writes markdown reports.
3.  **run_suite.sh**: Thin wrapper that launches the labeled Ginkgo suite with `go test`.

## Usage

### Prerequisites
- Go 1.22+
- `task` available in the shell
- `slimctl` available in `PATH`, or install it with `task benchmarks:slim:deps:slimctl-download`

### Running the Suite

The reproducible entrypoint is the Task target from the repository root:

```bash
task benchmarks:slim:benchmark:suite
```

That target always:
1. Runs from the correct benchmark directory.
2. Kills leftover local `slimctl`, `echo-client`, and `rate-client` processes from previous runs.
3. Applies a stable default benchmark matrix unless you override variables.

You can override the suite inputs directly on the task invocation:

```bash
task benchmarks:slim:benchmark:suite \
    BENCH_MODES='pub' \
    BENCH_CLIENTS='10' \
    BENCH_SIZES='128 1024' \
    BENCH_DURATION='2s' \
    BENCH_REPEATS='3'
```

For the repeated adaptive knee test used in the recent capacity study, use:

```bash
task benchmarks:slim:benchmark:capacity:pub-16kb
```

That task runs the fixed reproducible profile for:
- mode: `pub`
- clients: `10`
- payload: `128B`
- duration: `2s`
- adaptive sweep start/max: `128000` -> `176000`
- sweep growth factor: `1.125`
- sweep repeats per step: `3`

Additional reproducible workload-specific triggers:

```bash
task benchmarks:slim:benchmark:request
task benchmarks:slim:benchmark:ping-pong
task benchmarks:slim:benchmark:capacity:request-16kb
task benchmarks:slim:benchmark:capacity:ping-pong-16kb
```

CI-oriented bounded triggers:

```bash
task benchmarks:slim:benchmark:ci:suite-smoke
task benchmarks:slim:benchmark:ci:capacity
```

The CI targets are intentionally smaller than the developer-facing defaults:
- `benchmark:ci:suite-smoke` runs a `1s` single-client smoke matrix across `request`, `ping-pong`, and `pub`
- `benchmark:ci:capacity` runs bounded `1s` adaptive sweeps for `pub`, `request`, and `ping-pong`
- both targets are intended to be called directly from CI without shell preamble

The GitHub Actions workflow uploads a single Markdown artifact per job so the result is directly readable:
- smoke job artifact: `ci-smoke-report.md`
- capacity job artifact: `ci-capacity-report.md`

For CI runners that do not already provide `slimctl`, install it first with:

```bash
task benchmarks:slim:deps:slimctl-download SLIMCTL_PATH="$HOME/.local/bin/slimctl"
export PATH="$HOME/.local/bin:$PATH"
```

To run the full benchmark suite:

```bash
task benchmarks:slim:benchmark:suite

# equivalent low-level launcher
./run_suite.sh
```

By default this will:
1. Build the benchmark helpers through the Go test suite.
2. Start a local SLIM node via `slimctl`.
3. Execute `request`, `ping-pong`, and `pub` benchmarks from the Ginkgo suite.
4. Generate per-run reports under `reports/raw/`, plus `reports/results.tsv`, `reports/suite_summary.md`, and `reports/technical_report.md`.

The suite is configurable through environment variables:

```bash
MODES='request ping-pong pub'
CLIENTS='1 10 50'
SIZES='16 128 1024 10240'
REQUEST_RATES='1000'
PUB_RATES='1000'
DURATION='5s'
bash ./run_suite.sh
```

You can also enable an exploratory adaptive capacity sweep that increases the configured rate until sink throughput plateaus:

```bash
CAPACITY_SWEEP=1 \
CAPACITY_SWEEP_MODES='pub' \
CAPACITY_SWEEP_CLIENTS='10' \
CAPACITY_SWEEP_SIZES='128 1024' \
CAPACITY_SWEEP_START_RATE='1000' \
CAPACITY_SWEEP_GROWTH_FACTOR='2.0' \
CAPACITY_SWEEP_PLATEAU_THRESHOLD='0.05' \
CAPACITY_SWEEP_PLATEAU_STEPS='2' \
CAPACITY_SWEEP_MAX_STEPS='8' \
bash ./run_suite.sh
```

Notes:
- `request` and `ping-pong` are round-trip modes and should use paced rates.
- `pub` defaults to a safe automatic rate profile when `PUB_RATES` is unset.
- Set `PUB_RATES='0'` only when you want an explicit unpaced stress run.
- Live `sub` benchmarking is not implemented yet and is intentionally rejected by the binary.
- When `CAPACITY_SWEEP=1`, the suite writes an additional `reports/capacity_sweep.md` report.
- The sweep stops when sink throughput fails to improve by the configured threshold for the configured number of consecutive steps, or when `CAPACITY_SWEEP_MAX_STEPS` / `CAPACITY_SWEEP_MAX_RATE` is reached.

### Manual Usage

You can also run `slim-bench` individually:

```bash
go run . -mode=ping-pong -size=1024 -rate=100 -duration=10s -server=http://127.0.0.1:46357 -dest=agntcy/demo/echo -output=my_report.md
```

## Workflow Diagram

The following diagram illustrates the execution flow when running the full suite:

```mermaid
sequenceDiagram
    autonumber
    participant Runner as Test Runner
    participant Bench as slim-bench
    participant SUT as SLIM System
    participant Report as Report File

    Note over Runner: Loop over Parameters<br/>Sizes: 100B, 1KB, 10KB<br/>Rates: 10, 100, 1000 MPS

    Runner->>Bench: Launch (Mode=ping-pong)
    activate Bench

    Bench->>SUT: Connect (TLS/TCP)
    activate SUT
    SUT-->>Bench: Accepted

    loop Test Duration (e.g. 5s)
        Bench->>SUT: Send Message
        SUT-->>Bench: Echo/Ack
        Bench->>Bench: Record Latency
    end

    Bench->>Report: Generate Report
    deactivate SUT
    deactivate Bench

    Runner->>Report: Append to Suite Summary
    Note over Runner: Repeat for next parameter set
```

## Benchmark Logic

1.  **Connection**: The tool establishes a live connection to a SLIM node.
2.  **Traffic Generation**:
    *   **Request / Ping-Pong Mode**: Sends a message and waits for a response before sending the next. Ideal for RTT measurement.
    *   **Pub Mode**: Sends messages as fast as allowed by the configured rate. Ideal for throughput measurement.
3.  **Measurement**:
    *   Accesses system clock with high precision.
    *   Tracks operation latency for every message.
4.  **Analysis**:
    *   Sorts collected latencies to find percentiles.
    *   Records runtime errors alongside throughput and latency.
    *   Formats output into a human-readable Markdown report.
