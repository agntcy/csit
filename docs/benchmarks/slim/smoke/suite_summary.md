# SLIM Benchmark Statistical Summary

**Generated:** 2026-08-25 08:50:06

**Server:** http://127.0.0.1:41239
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
| 1 | 16B | 100 | 25 | 0.63 | 0.00 | [0.63, 0.63] | 0.63 | [0.62, 0.63] | 0.72 | [0.71, 0.73] | 9.60 | [9.47, 9.73] | 2.56 | [2.32, 2.80] | 17.76 | [17.40, 18.12] | 0 |

### Fire-And-Forget Results

| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95% CI | Observed Node Throughput Mean msg/sec | Observed Node Throughput Variance | Observed Node Throughput 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Responder Mean CPU % | Responder CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Mean Sender Msgs | Mean Observed Msgs | Total Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | 16B | 1000 | 25 | 922.72 | 163.63 | [917.44, 928.00] | 973.30 | 83.88 | [969.52, 977.08] | 16.76 | [16.37, 17.15] | 5.26 | [5.09, 5.44] | 1.83 | [1.72, 1.94] | 23.85 | [23.39, 24.31] | 999.72 | 999.72 | 0 |

### Write Results

| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95% CI | Sender Write Throughput Mean msg/sec | Sender Write Throughput Variance | Sender Write Throughput 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Responder Mean CPU % | Responder CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Mean Sender Msgs | Mean Observed Msgs | Total Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | 16B | 1000 | 25 | 923.10 | 235.73 | [916.76, 929.44] | 923.10 | 235.73 | [916.76, 929.44] | 17.25 | [16.86, 17.64] | 0.00 | [0.00, 0.00] | 1.65 | [1.41, 1.89] | 18.89 | [18.43, 19.35] | 999.84 | 999.84 | 0 |
