# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-06-26 08:19:41

**Modes:** fire-and-forget
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

- Modes: `fire-and-forget`
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

### Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `7210.38` msg/sec with 95% CI [7174.95, 7245.80]
Best sender-completed throughput: `7130.47` msg/sec with 95% CI [7093.50, 7167.44]
Best node CPU: `41.15` % with 95% CI [40.91, 41.39]
Best total CPU: `246.58` % with 95% CI [245.57, 247.60]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7130.47 | [7093.50, 7167.44] | 7210.38 | [7174.95, 7245.80] | 7364.00 | 0.00 | true | 41.15 | [40.91, 41.39] | 246.58 | [245.57, 247.60] | 0 |
| 2 | coarse | 144000 | 25 | 7163.89 | [7134.92, 7192.86] | 7241.88 | [7215.27, 7268.48] | 4154.58 | 0.44 | false | 41.41 | [41.31, 41.51] | 247.78 | [247.29, 248.27] | 0 |
| 3 | coarse | 162000 | 25 | 7144.52 | [7124.39, 7164.65] | 7215.58 | [7194.94, 7236.22] | 2500.06 | 0.07 | false | 41.29 | [41.19, 41.38] | 247.37 | [246.80, 247.93] | 0 |
| 4 | refine | 145000 | 25 | 7118.52 | [7087.43, 7149.60] | 7209.75 | [7181.02, 7238.48] | 4844.33 | -0.01 | false | 41.34 | [41.25, 41.43] | 247.62 | [247.21, 248.04] | 0 |
| 5 | refine | 136500 | 25 | 7156.42 | [7133.80, 7179.05] | 7248.20 | [7226.31, 7270.10] | 2812.80 | 0.52 | false | 41.19 | [41.07, 41.30] | 246.55 | [245.88, 247.22] | 0 |
| 6 | refine | 132250 | 25 | 7141.96 | [7109.99, 7173.94] | 7210.67 | [7180.83, 7240.52] | 5226.64 | 0.00 | false | 41.14 | [41.00, 41.28] | 246.51 | [245.89, 247.14] | 0 |
| 7 | refine | 130125 | 25 | 7191.67 | [7162.79, 7220.56] | 7274.59 | [7241.60, 7307.58] | 6386.30 | 0.89 | false | 41.45 | [41.36, 41.53] | 247.96 | [247.53, 248.40] | 0 |
