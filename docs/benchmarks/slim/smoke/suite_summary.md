# SLIM Benchmark Statistical Summary

**Generated:** 2026-11-05 18:07:23

**Server:** http://127.0.0.1:33257
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
| 1 | 16B | 100 | 25 | 0.90 | 0.00 | [0.87, 0.92] | 0.88 | [0.86, 0.90] | 1.29 | [1.16, 1.42] | 9.90 | [9.67, 10.14] | 3.29 | [3.09, 3.48] | 20.21 | [19.74, 20.68] | 0 |

### Fire-And-Forget Results

| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95% CI | Observed Node Throughput Mean msg/sec | Observed Node Throughput Variance | Observed Node Throughput 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Responder Mean CPU % | Responder CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Mean Sender Msgs | Mean Observed Msgs | Total Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | 16B | 1000 | 25 | 928.33 | 158.81 | [923.12, 933.53] | 972.76 | 95.95 | [968.72, 976.80] | 17.79 | [17.47, 18.11] | 6.69 | [6.39, 6.99] | 2.46 | [2.10, 2.83] | 26.94 | [26.18, 27.70] | 999.88 | 999.88 | 0 |

### Write Results

| Clients | Payload | Rate | Repeats | Sender Mean msg/sec | Sender Variance | Sender 95% CI | Sender Write Throughput Mean msg/sec | Sender Write Throughput Variance | Sender Write Throughput 95% CI | Sender Mean CPU % | Sender CPU 95% CI | Responder Mean CPU % | Responder CPU 95% CI | Node Mean CPU % | Node CPU 95% CI | Total Mean CPU % | Total CPU 95% CI | Mean Sender Msgs | Mean Observed Msgs | Total Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | 16B | 1000 | 25 | 929.00 | 238.14 | [922.63, 935.37] | 929.00 | 238.14 | [922.63, 935.37] | 16.92 | [16.47, 17.37] | 0.00 | [0.00, 0.00] | 2.51 | [2.02, 3.00] | 19.43 | [18.56, 20.30] | 999.88 | 999.88 | 0 |
