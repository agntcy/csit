# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-11-23 13:22:16

**Modes:** write
**Clients:** 1
**Sizes:** 16384
**Start Rate:** 128000
**Max Rate:** 176000
**Growth Factor:** 1.12
**Plateau Threshold:** 3.00%
**Plateau Steps:** 2
**Max Steps:** 6
**Repeats Per Sweep Step:** 25

## Adaptive Capacity Sweep

This sweep first increases the configured send rate geometrically to find the saturation region, then performs midpoint refinement to narrow the offered-rate interval that saturates the node.
Results are reported separately for each fixed `(mode, clients, payload)` case. The reported rate is the aggregate offered load across all clients in that case. For request-reply and fire-and-forget, effective throughput is sink-observed total node throughput. For write mode, effective throughput is sender-completed write throughput because no responder is running.

- Modes: `write`
- Clients: `1`
- Sizes: `16384` bytes
- Start rate: `128000` msg/sec
- Max rate: `176000` msg/sec (0 means unbounded by rate cap)
- Growth factor: `1.12`
- Plateau threshold: `3.00%` effective throughput gain
- Plateau steps: `2`
- Max steps: `6`
- Repeats per sweep step: `25`
- Refinement steps after coarse sweep: `4`
- Minimum offered-rate interval after refinement: `1000` msg/sec

### Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `6373.09` msg/sec with 95% CI [6331.23, 6414.96]
Best sender-completed throughput: `6373.09` msg/sec with 95% CI [6331.23, 6414.96]
Best node CPU: `44.79` % with 95% CI [44.42, 45.16]
Best total CPU: `187.84` % with 95% CI [186.48, 189.19]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6373.09 | [6331.23, 6414.96] | 6373.09 | [6331.23, 6414.96] | 10288.09 | 0.00 | true | 44.79 | [44.42, 45.16] | 187.84 | [186.48, 189.19] | 0 |
| 2 | coarse | 144000 | 25 | 6434.60 | [6403.13, 6466.08] | 6434.60 | [6403.13, 6466.08] | 5814.50 | 0.97 | false | 45.02 | [44.74, 45.31] | 188.40 | [187.49, 189.31] | 0 |
| 3 | coarse | 162000 | 25 | 6404.29 | [6355.02, 6453.57] | 6404.29 | [6355.02, 6453.57] | 14249.56 | 0.49 | false | 44.86 | [44.39, 45.33] | 187.58 | [185.99, 189.16] | 0 |
| 4 | refine | 145000 | 25 | 6397.26 | [6348.24, 6446.29] | 6397.26 | [6348.24, 6446.29] | 14107.39 | 0.38 | false | 44.85 | [44.42, 45.28] | 187.48 | [185.96, 189.00] | 0 |
| 5 | refine | 136500 | 25 | 6450.31 | [6411.42, 6489.21] | 6450.31 | [6411.42, 6489.21] | 8877.49 | 1.21 | false | 45.14 | [44.77, 45.52] | 188.08 | [186.86, 189.31] | 0 |
| 6 | refine | 132250 | 25 | 6434.66 | [6393.72, 6475.60] | 6434.66 | [6393.72, 6475.60] | 9837.80 | 0.97 | false | 45.05 | [44.72, 45.39] | 188.16 | [187.07, 189.26] | 0 |
| 7 | refine | 130125 | 25 | 6410.01 | [6364.68, 6455.35] | 6410.01 | [6364.68, 6455.35] | 12063.41 | 0.58 | false | 44.76 | [44.36, 45.16] | 187.33 | [186.03, 188.63] | 0 |
