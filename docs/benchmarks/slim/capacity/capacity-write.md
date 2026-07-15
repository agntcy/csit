# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-15 09:07:25

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
Best sender write throughput: `5937.01` msg/sec with 95% CI [5818.60, 6055.41]
Best sender-completed throughput: `5937.01` msg/sec with 95% CI [5818.60, 6055.41]
Best node CPU: `43.74` % with 95% CI [42.85, 44.62]
Best total CPU: `184.97` % with 95% CI [183.52, 186.41]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5937.01 | [5818.60, 6055.41] | 5937.01 | [5818.60, 6055.41] | 82283.13 | 0.00 | true | 43.74 | [42.85, 44.62] | 184.97 | [183.52, 186.41] | 0 |
| 2 | coarse | 144000 | 25 | 6006.92 | [5993.68, 6020.16] | 6006.92 | [5993.68, 6020.16] | 1029.04 | 1.18 | false | 44.16 | [44.04, 44.28] | 185.10 | [184.64, 185.55] | 0 |
| 3 | coarse | 162000 | 25 | 5980.18 | [5964.09, 5996.28] | 5980.18 | [5964.09, 5996.28] | 1520.82 | 0.73 | false | 44.16 | [44.03, 44.29] | 185.27 | [184.83, 185.71] | 0 |
| 4 | refine | 145000 | 25 | 5994.62 | [5981.53, 6007.70] | 5994.62 | [5981.53, 6007.70] | 1004.71 | 0.97 | false | 44.39 | [44.27, 44.52] | 185.43 | [185.03, 185.83] | 0 |
| 5 | refine | 136500 | 25 | 5963.70 | [5948.98, 5978.43] | 5963.70 | [5948.98, 5978.43] | 1272.35 | 0.45 | false | 44.23 | [44.12, 44.34] | 185.69 | [185.30, 186.08] | 0 |
| 6 | refine | 132250 | 25 | 5993.76 | [5979.29, 6008.22] | 5993.76 | [5979.29, 6008.22] | 1228.60 | 0.96 | false | 44.15 | [44.02, 44.28] | 185.01 | [184.53, 185.50] | 0 |
| 7 | refine | 130125 | 25 | 6020.98 | [6006.21, 6035.75] | 6020.98 | [6006.21, 6035.75] | 1281.10 | 1.41 | false | 44.34 | [44.18, 44.49] | 185.84 | [185.38, 186.30] | 0 |
