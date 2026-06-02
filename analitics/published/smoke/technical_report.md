# SLIM Benchmark Technical Report

## Scope

This report documents the repeated benchmark campaign executed by the Ginkgo benchmark suite against a local SLIM node. Each case in the suite matrix is rerun multiple times to estimate mean performance, sample variance, and confidence intervals.

## Test Setup

- Runtime: local SLIM node on `http://127.0.0.1:65158`
- Destination identity: `agntcy/demo/echo`
- Sender: `tests/rate-client`
- Sink / responder: `tests/echo-client` (used by request-reply and fire-and-forget; write mode runs without a responder)
- Suite driver: Ginkgo spec in `benchmarks/agntcy-slim/tests/benchmark_suite_test.go`
- Modes: `request-reply fire-and-forget write`
- Client counts: `1 10 50`
- Payload sizes: `16 128 1024 10240` bytes
- Request-reply rates: `1000` msg/sec
- One-way rates: `auto (safe default profile)`
- Write rates: `auto (falls back to one-way rate profile)`
- Duration per run: `5s`
- Repeats per case: `1`
- Adaptive capacity sweep enabled: `false`

## Measurement Methodology

### Execution Model

Each benchmark case in the matrix is executed 1 times. A benchmark case is uniquely identified by:

- mode
- client count
- payload size
- configured rate

For this statistical rerun, each individual run uses a configured sender duration of 5s.

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

where $\alpha = 0.05$ and $n = 1$ for each case in this report.


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
| 1 | 16B | 1000 | 1 | 0.57 | 0.00 | [0.57, 0.57] | 0.58 | [0.58, 0.58] | 0.78 | [0.78, 0.78] | 26.65 | [26.65, 26.65] | 16.28 | [16.28, 16.28] | 61.94 | [61.94, 61.94] | 0 |
| 1 | 128B | 1000 | 1 | 0.57 | 0.00 | [0.57, 0.57] | 0.58 | [0.58, 0.58] | 0.69 | [0.69, 0.69] | 28.65 | [28.65, 28.65] | 17.50 | [17.50, 17.50] | 66.04 | [66.04, 66.04] | 0 |
| 1 | 1024B | 1000 | 1 | 0.57 | 0.00 | [0.57, 0.57] | 0.58 | [0.58, 0.58] | 0.69 | [0.69, 0.69] | 28.41 | [28.41, 28.41] | 17.12 | [17.12, 17.12] | 65.04 | [65.04, 65.04] | 0 |
| 1 | 10240B | 1000 | 1 | 0.59 | 0.00 | [0.59, 0.59] | 0.60 | [0.60, 0.60] | 0.72 | [0.72, 0.72] | 29.89 | [29.89, 29.89] | 15.52 | [15.52, 15.52] | 66.09 | [66.09, 66.09] | 0 |
| 10 | 16B | 1000 | 1 | 0.76 | 0.00 | [0.76, 0.76] | 0.77 | [0.77, 0.77] | 1.12 | [1.12, 1.12] | 27.97 | [27.97, 27.97] | 9.55 | [9.55, 9.55] | 57.22 | [57.22, 57.22] | 0 |
| 10 | 128B | 1000 | 1 | 1.43 | 0.00 | [1.43, 1.43] | 1.39 | [1.39, 1.39] | 3.20 | [3.20, 3.20] | 30.70 | [30.70, 30.70] | 10.14 | [10.14, 10.14] | 62.70 | [62.70, 62.70] | 0 |
| 10 | 1024B | 1000 | 1 | 1.24 | 0.00 | [1.24, 1.24] | 1.12 | [1.12, 1.12] | 2.57 | [2.57, 2.57] | 32.67 | [32.67, 32.67] | 11.13 | [11.13, 11.13] | 68.44 | [68.44, 68.44] | 0 |
| 10 | 10240B | 1000 | 1 | 1.01 | 0.00 | [1.01, 1.01] | 1.00 | [1.00, 1.00] | 1.47 | [1.47, 1.47] | 32.03 | [32.03, 32.03] | 12.14 | [12.14, 12.14] | 70.03 | [70.03, 70.03] | 0 |
| 50 | 16B | 1000 | 1 | 0.59 | 0.00 | [0.59, 0.59] | 0.58 | [0.58, 0.58] | 1.09 | [1.09, 1.09] | 25.73 | [25.73, 25.73] | 9.14 | [9.14, 9.14] | 50.77 | [50.77, 50.77] | 0 |
| 50 | 128B | 1000 | 1 | 0.62 | 0.00 | [0.62, 0.62] | 0.61 | [0.61, 0.61] | 1.10 | [1.10, 1.10] | 25.86 | [25.86, 25.86] | 8.54 | [8.54, 8.54] | 50.87 | [50.87, 50.87] | 0 |
| 50 | 1024B | 1000 | 1 | 0.76 | 0.00 | [0.76, 0.76] | 0.74 | [0.74, 0.74] | 1.18 | [1.18, 1.18] | 26.71 | [26.71, 26.71] | 8.93 | [8.93, 8.93] | 54.30 | [54.30, 54.30] | 0 |
| 50 | 10240B | 1000 | 1 | 1.00 | 0.00 | [1.00, 1.00] | 0.96 | [0.96, 0.96] | 2.02 | [2.02, 2.02] | 30.80 | [30.80, 30.80] | 11.12 | [11.12, 11.12] | 66.96 | [66.96, 66.96] | 0 |

### Fire-And-Forget Results

| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95% CI | Observed Node Throughput Mean msg/sec | Observed Node Throughput Variance | Observed Node Throughput 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Responder Mean CPU % | Responder CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Mean Sender Msgs | Mean Observed Msgs | Total Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | 16B | 1000 | 1 | 998.48 | 0.00 | [998.48, 998.48] | 1000.16 | 0.00 | [1000.16, 1000.16] | 23.37 | [23.37, 23.37] | 13.12 | [13.12, 13.12] | 12.52 | [12.52, 12.52] | 49.01 | [49.01, 49.01] | 5000.00 | 5000.00 | 0 |
| 1 | 128B | 1000 | 1 | 998.85 | 0.00 | [998.85, 998.85] | 1000.19 | 0.00 | [1000.19, 1000.19] | 24.61 | [24.61, 24.61] | 13.73 | [13.73, 13.73] | 13.33 | [13.33, 13.33] | 51.67 | [51.67, 51.67] | 5000.00 | 5000.00 | 0 |
| 1 | 1024B | 1000 | 1 | 998.61 | 0.00 | [998.61, 998.61] | 1000.20 | 0.00 | [1000.20, 1000.20] | 25.10 | [25.10, 25.10] | 14.13 | [14.13, 14.13] | 13.33 | [13.33, 13.33] | 52.56 | [52.56, 52.56] | 5000.00 | 5000.00 | 0 |
| 1 | 10240B | 500 | 1 | 499.21 | 0.00 | [499.21, 499.21] | 500.15 | 0.00 | [500.15, 500.15] | 20.03 | [20.03, 20.03] | 10.54 | [10.54, 10.54] | 8.16 | [8.16, 8.16] | 38.74 | [38.74, 38.74] | 2500.00 | 2500.00 | 0 |
| 10 | 16B | 1000 | 1 | 997.75 | 0.00 | [997.75, 997.75] | 1001.45 | 0.00 | [1001.45, 1001.45] | 21.27 | [21.27, 21.27] | 13.33 | [13.33, 13.33] | 6.37 | [6.37, 6.37] | 40.96 | [40.96, 40.96] | 5000.00 | 5000.00 | 0 |
| 10 | 128B | 1000 | 1 | 997.79 | 0.00 | [997.79, 997.79] | 1001.26 | 0.00 | [1001.26, 1001.26] | 21.06 | [21.06, 21.06] | 10.35 | [10.35, 10.35] | 6.17 | [6.17, 6.17] | 37.58 | [37.58, 37.58] | 5000.00 | 5000.00 | 0 |
| 10 | 1024B | 500 | 1 | 498.94 | 0.00 | [498.94, 498.94] | 501.75 | 0.00 | [501.75, 501.75] | 16.65 | [16.65, 16.65] | 8.94 | [8.94, 8.94] | 4.37 | [4.37, 4.37] | 29.97 | [29.97, 29.97] | 2500.00 | 2500.00 | 0 |
| 10 | 10240B | 200 | 1 | 199.63 | 0.00 | [199.63, 199.63] | 201.97 | 0.00 | [201.97, 201.97] | 10.44 | [10.44, 10.44] | 4.57 | [4.57, 4.57] | 2.19 | [2.19, 2.19] | 17.20 | [17.20, 17.20] | 1000.00 | 1000.00 | 0 |
| 50 | 16B | 500 | 1 | 497.98 | 0.00 | [497.98, 497.98] | 508.89 | 0.00 | [508.89, 508.89] | 18.15 | [18.15, 18.15] | 7.54 | [7.54, 7.54] | 3.18 | [3.18, 3.18] | 28.87 | [28.87, 28.87] | 2500.00 | 2500.00 | 0 |
| 50 | 128B | 500 | 1 | 498.20 | 0.00 | [498.20, 498.20] | 509.09 | 0.00 | [509.09, 509.09] | 17.89 | [17.89, 17.89] | 7.14 | [7.14, 7.14] | 3.37 | [3.37, 3.37] | 28.41 | [28.41, 28.41] | 2500.00 | 2500.00 | 0 |
| 50 | 1024B | 250 | 1 | 249.07 | 0.00 | [249.07, 249.07] | 259.86 | 0.00 | [259.86, 259.86] | 14.65 | [14.65, 14.65] | 4.36 | [4.36, 4.36] | 1.59 | [1.59, 1.59] | 20.61 | [20.61, 20.61] | 1250.00 | 1250.00 | 0 |
| 50 | 10240B | 100 | 1 | 99.63 | 0.00 | [99.63, 99.63] | 110.85 | 0.00 | [110.85, 110.85] | 12.01 | [12.01, 12.01] | 2.78 | [2.78, 2.78] | 1.19 | [1.19, 1.19] | 15.98 | [15.98, 15.98] | 500.00 | 500.00 | 0 |

### Write Results

| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95% CI | Sender Write Throughput Mean msg/sec | Sender Write Throughput Variance | Sender Write Throughput 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Responder Mean CPU % | Responder CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Mean Sender Msgs | Mean Observed Msgs | Total Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | 16B | 1000 | 1 | 998.62 | 0.00 | [998.62, 998.62] | 998.62 | 0.00 | [998.62, 998.62] | 24.33 | [24.33, 24.33] | 0.00 | [0.00, 0.00] | 13.13 | [13.13, 13.13] | 37.46 | [37.46, 37.46] | 5000.00 | 5000.00 | 0 |
| 1 | 128B | 1000 | 1 | 998.78 | 0.00 | [998.78, 998.78] | 998.78 | 0.00 | [998.78, 998.78] | 26.26 | [26.26, 26.26] | 0.00 | [0.00, 0.00] | 14.54 | [14.54, 14.54] | 40.80 | [40.80, 40.80] | 5000.00 | 5000.00 | 0 |
| 1 | 1024B | 1000 | 1 | 998.54 | 0.00 | [998.54, 998.54] | 998.54 | 0.00 | [998.54, 998.54] | 25.99 | [25.99, 25.99] | 0.00 | [0.00, 0.00] | 13.91 | [13.91, 13.91] | 39.90 | [39.90, 39.90] | 5000.00 | 5000.00 | 0 |
| 1 | 10240B | 500 | 1 | 499.34 | 0.00 | [499.34, 499.34] | 499.34 | 0.00 | [499.34, 499.34] | 19.58 | [19.58, 19.58] | 0.00 | [0.00, 0.00] | 7.97 | [7.97, 7.97] | 27.55 | [27.55, 27.55] | 2500.00 | 2500.00 | 0 |
| 10 | 16B | 1000 | 1 | 997.94 | 0.00 | [997.94, 997.94] | 997.94 | 0.00 | [997.94, 997.94] | 19.51 | [19.51, 19.51] | 0.00 | [0.00, 0.00] | 5.36 | [5.36, 5.36] | 24.87 | [24.87, 24.87] | 5000.00 | 5000.00 | 0 |
| 10 | 128B | 1000 | 1 | 997.85 | 0.00 | [997.85, 997.85] | 997.85 | 0.00 | [997.85, 997.85] | 18.51 | [18.51, 18.51] | 0.00 | [0.00, 0.00] | 4.97 | [4.97, 4.97] | 23.48 | [23.48, 23.48] | 5000.00 | 5000.00 | 0 |
| 10 | 1024B | 500 | 1 | 498.98 | 0.00 | [498.98, 498.98] | 498.98 | 0.00 | [498.98, 498.98] | 11.90 | [11.90, 11.90] | 0.00 | [0.00, 0.00] | 2.79 | [2.79, 2.79] | 14.69 | [14.69, 14.69] | 2500.00 | 2500.00 | 0 |
| 10 | 10240B | 200 | 1 | 199.49 | 0.00 | [199.49, 199.49] | 199.49 | 0.00 | [199.49, 199.49] | 12.21 | [12.21, 12.21] | 0.00 | [0.00, 0.00] | 3.18 | [3.18, 3.18] | 15.39 | [15.39, 15.39] | 1000.00 | 1000.00 | 0 |
| 50 | 16B | 500 | 1 | 498.25 | 0.00 | [498.25, 498.25] | 498.25 | 0.00 | [498.25, 498.25] | 16.83 | [16.83, 16.83] | 0.00 | [0.00, 0.00] | 2.78 | [2.78, 2.78] | 19.62 | [19.62, 19.62] | 2500.00 | 2500.00 | 0 |
| 50 | 128B | 500 | 1 | 497.35 | 0.00 | [497.35, 497.35] | 497.35 | 0.00 | [497.35, 497.35] | 15.74 | [15.74, 15.74] | 0.00 | [0.00, 0.00] | 2.78 | [2.78, 2.78] | 18.52 | [18.52, 18.52] | 2500.00 | 2500.00 | 0 |
| 50 | 1024B | 250 | 1 | 249.03 | 0.00 | [249.03, 249.03] | 249.03 | 0.00 | [249.03, 249.03] | 14.30 | [14.30, 14.30] | 0.00 | [0.00, 0.00] | 1.78 | [1.78, 1.78] | 16.09 | [16.09, 16.09] | 1250.00 | 1250.00 | 0 |
| 50 | 10240B | 100 | 1 | 99.62 | 0.00 | [99.62, 99.62] | 99.62 | 0.00 | [99.62, 99.62] | 12.87 | [12.87, 12.87] | 0.00 | [0.00, 0.00] | 1.19 | [1.19, 1.19] | 14.06 | [14.06, 14.06] | 500.00 | 500.00 | 0 |


## Result Interpretation

- For request-reply, the primary metrics are sender-observed latency statistics, especially mean, p50, and p99 latency.
- For fire-and-forget and write, the primary metrics are throughput statistics because those workloads are intended to characterize node write and forwarding capacity.
- CPU percentages represent average process CPU utilization during the benchmark window for sender, responder, and node processes.
- Confidence intervals estimate the uncertainty around the reported latency or throughput means under repeated execution.
- For request-reply and fire-and-forget capacity sweeps, sink throughput remains the better end-to-end capacity indicator when it diverges from sender throughput.
