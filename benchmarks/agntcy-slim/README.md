# SLIM Benchmark Suite

This package contains the reproducible local SLIM benchmark suite, its benchmark-focused tests, a thin launcher script, and generated report artifacts.

The suite is implemented by [tests/benchmark_suite_test.go](./tests/benchmark_suite_test.go) and the launcher lives at [scripts/run_suite.sh](./scripts/run_suite.sh).

## Components

1. `tests/benchmark_suite_test.go`: Ginkgo suite runner that starts a local SLIM node, provisions responder topology per workload, executes the benchmark matrix, and writes reports.
2. `tests/rate-client`: Live traffic generator used by the suite and benchmark-focused tests.
3. `tests/echo-client`: Responder and sink helper used by request-reply, fire-and-forget, and write setup.
4. `scripts/run_suite.sh`: Thin wrapper that launches the labeled Ginkgo benchmark suite.
5. `reports/`: Generated benchmark artifacts such as `results.tsv`, summary markdown, technical markdown, and per-run raw outputs.

## Usage

### Prerequisites

- Go 1.22+
- `task` available in the shell
- `slimctl` available in `PATH`, or install it with `task benchmarks:slim:deps:slimctl-download`

### Running The Suite

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
    BENCH_MODES='fire-and-forget write' \
    BENCH_CLIENTS='10' \
    BENCH_SIZES='128 1024' \
    BENCH_DURATION='2s' \
    BENCH_REPEATS='3'
```

For the repeated adaptive knee test used in the recent capacity study, use:

```bash
task benchmarks:slim:benchmark:capacity:fire-and-forget-16kb
```

That task runs the fixed reproducible profile for:

- mode: `fire-and-forget`
- clients: `1`
- payload: `16kB`
- duration: `2s`
- adaptive sweep start/max: `128000` -> `176000`
- sweep growth factor: `1.125`
- sweep repeats per step: `3`

Additional reproducible workload-specific triggers:

```bash
task benchmarks:slim:benchmark:request-reply
task benchmarks:slim:benchmark:fire-and-forget
task benchmarks:slim:benchmark:write
task benchmarks:slim:benchmark:capacity:request-reply-16kb
task benchmarks:slim:benchmark:capacity:write-16kb
```

The old task names and old mode spellings have been removed. Use only `request-reply`, `fire-and-forget`, and `write`.

CI-oriented bounded triggers:

```bash
task benchmarks:slim:benchmark:ci:suite-smoke
task benchmarks:slim:benchmark:ci:capacity
```

The CI targets are intentionally narrower than the developer-facing defaults, but they now use `25` repeats so the reported confidence intervals are based on statistically meaningful reruns:

- `benchmark:ci:suite-smoke` runs a `1s` single-client smoke matrix across `request-reply`, `fire-and-forget`, and `write` with `25` repeats per benchmark case
- `benchmark:ci:capacity` runs bounded `5s` adaptive sweeps for `fire-and-forget`, `request-reply`, and `write` using `1` client at `16kB`, with `25` repeats for the top-level suite case and `25` repeats for each adaptive sweep step
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
bash ./benchmarks/agntcy-slim/scripts/run_suite.sh
```

By default this will:

1. Build the benchmark helpers through the Go test suite.
2. Start a local SLIM node via `slimctl`.
3. Execute `request-reply`, `fire-and-forget`, and `write` benchmarks from the Ginkgo suite.
4. Generate per-run reports under `reports/raw/`, plus `reports/results.tsv`, `reports/suite_summary.md`, and `reports/technical_report.md`.

The suite is configurable through environment variables:

```bash
MODES='request-reply fire-and-forget write'
CLIENTS='1 10 50'
SIZES='16 128 1024 10240'
REQUEST_RATES='1000'
PUB_RATES='1000'
WRITE_RATES='2000'
DURATION='5s'
bash ./benchmarks/agntcy-slim/scripts/run_suite.sh
```

You can also enable an exploratory adaptive capacity sweep that increases the configured rate until sink throughput plateaus:

```bash
CAPACITY_SWEEP=1 \
CAPACITY_SWEEP_MODES='fire-and-forget' \
CAPACITY_SWEEP_CLIENTS='1' \
CAPACITY_SWEEP_SIZES='16384' \
CAPACITY_SWEEP_START_RATE='1000' \
CAPACITY_SWEEP_GROWTH_FACTOR='2.0' \
CAPACITY_SWEEP_PLATEAU_THRESHOLD='0.05' \
CAPACITY_SWEEP_PLATEAU_STEPS='2' \
CAPACITY_SWEEP_MAX_STEPS='8' \
bash ./benchmarks/agntcy-slim/scripts/run_suite.sh
```

Notes:

- `request-reply` is the round-trip mode and should use paced rates.
- `fire-and-forget` uses a sink responder and defaults to a safe automatic rate profile when `PUB_RATES` is unset.
- `write` measures sender-to-node write rate with no responder process; it uses `WRITE_RATES` when set, otherwise it falls back to `PUB_RATES`.
- Set `PUB_RATES='0'` only when you want an explicit unpaced stress run.
- Live `sub` benchmarking is not implemented yet and is intentionally rejected by the binary.
- When `CAPACITY_SWEEP=1`, the suite writes an additional `reports/capacity_sweep.md` report.
- The sweep stops when the mode-specific effective throughput fails to improve by the configured threshold for the configured number of consecutive steps, or when `CAPACITY_SWEEP_MAX_STEPS` or `CAPACITY_SWEEP_MAX_RATE` is reached.
