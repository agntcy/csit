# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-14 14:14:23

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
Best sender write throughput: `6401.74` msg/sec with 95% CI [6383.82, 6419.65]
Best sender-completed throughput: `6401.74` msg/sec with 95% CI [6383.82, 6419.65]
Best node CPU: `44.58` % with 95% CI [44.44, 44.72]
Best total CPU: `187.01` % with 95% CI [186.68, 187.34]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6401.74 | [6383.82, 6419.65] | 6401.74 | [6383.82, 6419.65] | 1884.58 | 0.00 | true | 44.58 | [44.44, 44.72] | 187.01 | [186.68, 187.34] | 0 |
| 2 | coarse | 144000 | 25 | 6392.71 | [6377.31, 6408.12] | 6392.71 | [6377.31, 6408.12] | 1393.08 | -0.14 | false | 44.47 | [44.35, 44.59] | 186.74 | [186.36, 187.12] | 0 |
| 3 | coarse | 162000 | 25 | 6408.67 | [6395.93, 6421.42] | 6408.67 | [6395.93, 6421.42] | 952.70 | 0.11 | false | 44.49 | [44.37, 44.61] | 187.05 | [186.75, 187.36] | 0 |
| 4 | refine | 145000 | 25 | 6399.10 | [6383.35, 6414.86] | 6399.10 | [6383.35, 6414.86] | 1456.81 | -0.04 | false | 44.45 | [44.32, 44.58] | 186.81 | [186.42, 187.21] | 0 |
| 5 | refine | 136500 | 25 | 6406.97 | [6387.95, 6425.98] | 6406.97 | [6387.95, 6425.98] | 2122.48 | 0.08 | false | 44.51 | [44.38, 44.63] | 186.91 | [186.55, 187.28] | 0 |
| 6 | refine | 132250 | 25 | 6432.66 | [6421.62, 6443.69] | 6432.66 | [6421.62, 6443.69] | 714.69 | 0.48 | false | 44.58 | [44.49, 44.68] | 187.46 | [187.14, 187.78] | 0 |
| 7 | refine | 130125 | 25 | 6421.58 | [6410.78, 6432.38] | 6421.58 | [6410.78, 6432.38] | 684.59 | 0.31 | false | 44.60 | [44.48, 44.71] | 187.29 | [187.00, 187.59] | 0 |
