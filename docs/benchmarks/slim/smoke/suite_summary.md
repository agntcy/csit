# SLIM Benchmark Statistical Summary

**Generated:** 2026-07-14 13:48:46

**Server:** http://127.0.0.1:41481
**Destination:** agntcy/demo/echo
**Modes:** request-reply fire-and-forget write
**Clients:** 1
**Sizes:** 16
**Request-Reply Rates:** 100
**One-Way Rates:** 1000
**Write Rates:** 1000
**Duration Per Run:** 1s
**Repeats Per Case:** 25

This summary reports mean, sample variance, and confidence intervals over repeated executions of each benchmark case.

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
