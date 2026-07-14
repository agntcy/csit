# SLIM Benchmark Technical Report

## Scope

This report documents the repeated benchmark campaign executed by the Ginkgo benchmark suite against a local SLIM node. Each case in the suite matrix is rerun multiple times to estimate mean performance, sample variance, and confidence intervals.

## Test Setup

- Runtime: local SLIM node on `http://127.0.0.1:41481`
- Destination identity: `agntcy/demo/echo`
- Sender: `tests/rate-client`
- Sink / responder: `tests/echo-client` (used by request-reply and fire-and-forget; write mode runs without a responder)
- Suite driver: Ginkgo spec in `benchmarks/agntcy-slim/tests/benchmark_suite_test.go`
- Modes: `request-reply fire-and-forget write`
- Client counts: `1`
- Payload sizes: `16` bytes
- Request-reply rates: `100` msg/sec
- One-way rates: `1000`
- Write rates: `1000`
- Duration per run: `1s`
- Repeats per case: `25`
- Adaptive capacity sweep enabled: `false`

## Measurement Methodology

### Execution Model

Each benchmark case in the matrix is executed 25 times. A benchmark case is uniquely identified by:

- mode
- client count
- payload size
- configured rate

For this statistical rerun, each individual run uses a configured sender duration of 1s.

### Sender-Side Measurement

Sender throughput is measured by tests/rate-client.

For each run:

1. The sender starts its timed send loop.
2. It records the actual wall-clock run duration.
3. It counts the total number of successfully completed sends.
4. It computes sender throughput as:

$$
\text{sender mps} = \frac{\text{total successful messages}}{\text{actual run duration in seconds}}
$$

### Responder-Side Measurement

For request-reply and fire-and-forget, responder throughput is measured by tests/echo-client.

For each run:

1. The sink counts received messages and received bytes.
2. It records the timestamp of the first payload message received.
3. It records the timestamp of the last payload message received.
4. It computes active receive throughput over the active message window, not over sink process lifetime:

$$
\text{sink mps} = \frac{\text{received messages}}{\text{last message time} - \text{first message time}}
$$

If only one message is observed, the sink falls back to elapsed lifetime-based timing to avoid division by zero.

Write mode does not start a responder. In that mode, the sender-completed write rate is the only throughput measurement and represents how fast the sender can successfully enqueue writes into the node.

### CPU Measurement

CPU usage is collected for the three benchmark processes involved in each run:

- sender process: tests/rate-client
- responder process: tests/echo-client
- node process: slimctl slim start

The sender CPU time is read from the child process state after exit as user time plus system time.

The responder and node CPU time are read as deltas of cumulative process CPU time between the start and end of the benchmark window.

Average CPU percent for each process is computed as:

$$
\text{cpu percent} = 100 \cdot \frac{\text{cpu time consumed during benchmark}}{\text{benchmark wall-clock duration}}
$$

The total CPU percent for a run is the sum of sender, responder, and node average CPU percent.

### Statistical Treatment

For each case, the report computes:

- mean
- sample variance
- standard deviation
- Student's t 95% confidence interval for the mean

The sample variance is:

$$
s^2 = \frac{1}{n-1} \sum_{i=1}^n (x_i - \bar{x})^2
$$

The Student's t 95% confidence interval is:

$$
\bar{x} \pm t_{1-\alpha/2, n-1} \cdot \frac{s}{\sqrt{n}}
$$

where $\alpha = 0.05$ and $n = 25$ for each case in this report.


## Test Types

### Request-Reply

Request-reply sends one message and waits for the echoed reply before sending the next. It measures paced round-trip behavior.

### Fire-And-Forget

Fire-and-forget sends one-way traffic to a sink responder. It measures end-to-end one-way delivery through the node without waiting for per-message replies.

### Write

Write measures how fast the sender can successfully write messages into the node without any sink or responder process. In this mode, sender-completed throughput is the primary metric.

## Full Matrix

### Request-Reply Results

Request-reply prioritizes latency statistics. The configured rate is retained as load context, but the primary reported metrics are mean, p50, and p99 latency.

| Clients | Payload | Rate | Repeats | Mean Latency ms | Mean Latency Variance | Mean Latency 95% CI | P50 Latency ms | P50 Latency 95% CI | P99 Latency ms | P99 Latency 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Total Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | 16B | 100 | 25 | 0.64 | 0.00 | [0.63, 0.65] | 0.63 | [0.62, 0.64] | 0.81 | [0.79, 0.83] | 6.65 | [6.43, 6.86] | 2.08 | [1.83, 2.33] | 12.85 | [12.42, 13.28] | 0 |

### Fire-And-Forget Results

| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95% CI | Observed Node Throughput Mean msg/sec | Observed Node Throughput Variance | Observed Node Throughput 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Responder Mean CPU % | Responder CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Mean Sender Msgs | Mean Observed Msgs | Total Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | 16B | 1000 | 25 | 925.12 | 234.63 | [918.80, 931.44] | 971.69 | 57.59 | [968.56, 974.82] | 11.83 | [11.68, 11.98] | 4.62 | [4.55, 4.68] | 1.66 | [1.38, 1.93] | 18.10 | [17.75, 18.45] | 999.96 | 999.96 | 0 |

### Write Results

| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95% CI | Sender Write Throughput Mean msg/sec | Sender Write Throughput Variance | Sender Write Throughput 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Responder Mean CPU % | Responder CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Mean Sender Msgs | Mean Observed Msgs | Total Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | 16B | 1000 | 25 | 928.43 | 85.55 | [924.61, 932.25] | 928.43 | 85.55 | [924.61, 932.25] | 11.76 | [11.66, 11.85] | 0.00 | [0.00, 0.00] | 1.66 | [1.44, 1.87] | 13.41 | [13.18, 13.64] | 999.92 | 999.92 | 0 |


## Result Interpretation

- For request-reply, the primary metrics are sender-observed latency statistics, especially mean, p50, and p99 latency.
- For fire-and-forget and write, the primary metrics are throughput statistics because those workloads are intended to characterize node write and forwarding capacity.
- CPU percentages represent average process CPU utilization during the benchmark window for sender, responder, and node processes.
- Confidence intervals estimate the uncertainty around the reported latency or throughput means under repeated execution.
- For request-reply and fire-and-forget capacity sweeps, sink throughput remains the better end-to-end capacity indicator when it diverges from sender throughput.
