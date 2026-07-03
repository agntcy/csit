# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-03 12:54:17

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
Best sender write throughput: `6468.05` msg/sec with 95% CI [6449.91, 6486.18]
Best sender-completed throughput: `6468.05` msg/sec with 95% CI [6449.91, 6486.18]
Best node CPU: `44.35` % with 95% CI [44.21, 44.50]
Best total CPU: `186.49` % with 95% CI [186.01, 186.98]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6468.05 | [6449.91, 6486.18] | 6468.05 | [6449.91, 6486.18] | 1931.17 | 0.00 | true | 44.35 | [44.21, 44.50] | 186.49 | [186.01, 186.98] | 0 |
| 2 | coarse | 144000 | 25 | 6442.04 | [6424.09, 6459.99] | 6442.04 | [6424.09, 6459.99] | 1891.50 | -0.40 | false | 44.09 | [43.95, 44.23] | 185.53 | [185.06, 185.99] | 0 |
| 3 | coarse | 162000 | 25 | 6466.02 | [6453.40, 6478.64] | 6466.02 | [6453.40, 6478.64] | 935.12 | -0.03 | false | 44.49 | [44.38, 44.59] | 186.97 | [186.61, 187.33] | 0 |
| 4 | refine | 145000 | 25 | 6466.84 | [6445.72, 6487.96] | 6466.84 | [6445.72, 6487.96] | 2617.94 | -0.02 | false | 44.48 | [44.33, 44.62] | 187.01 | [186.55, 187.47] | 0 |
| 5 | refine | 136500 | 25 | 6478.42 | [6458.10, 6498.74] | 6478.42 | [6458.10, 6498.74] | 2423.41 | 0.16 | false | 44.57 | [44.37, 44.76] | 186.96 | [186.39, 187.54] | 0 |
| 6 | refine | 132250 | 25 | 6493.29 | [6464.25, 6522.32] | 6493.29 | [6464.25, 6522.32] | 4949.18 | 0.39 | false | 44.66 | [44.42, 44.90] | 187.49 | [187.10, 187.87] | 0 |
| 7 | refine | 130125 | 25 | 6501.71 | [6487.49, 6515.93] | 6501.71 | [6487.49, 6515.93] | 1186.78 | 0.52 | false | 44.60 | [44.49, 44.72] | 187.35 | [186.98, 187.72] | 0 |
