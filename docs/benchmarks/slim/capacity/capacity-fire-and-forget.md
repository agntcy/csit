# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-08-13 12:38:44

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
Best observed node throughput: `5901.35` msg/sec with 95% CI [5854.62, 5948.07]
Best sender-completed throughput: `5835.86` msg/sec with 95% CI [5787.74, 5883.98]
Best node CPU: `43.11` % with 95% CI [42.58, 43.63]
Best total CPU: `251.13` % with 95% CI [248.80, 253.46]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5835.86 | [5787.74, 5883.98] | 5901.35 | [5854.62, 5948.07] | 12812.29 | 0.00 | true | 43.11 | [42.58, 43.63] | 251.13 | [248.80, 253.46] | 0 |
| 2 | coarse | 144000 | 25 | 5894.79 | [5815.63, 5973.94] | 5962.59 | [5881.34, 6043.83] | 38737.87 | 1.04 | false | 43.42 | [42.79, 44.05] | 252.79 | [250.95, 254.62] | 0 |
| 3 | coarse | 162000 | 25 | 5909.57 | [5837.85, 5981.30] | 5975.65 | [5902.26, 6049.04] | 31612.43 | 1.26 | false | 43.47 | [42.91, 44.02] | 252.85 | [251.18, 254.51] | 0 |
| 4 | refine | 145000 | 25 | 5855.75 | [5840.33, 5871.16] | 5917.92 | [5902.64, 5933.19] | 1369.85 | 0.28 | false | 43.44 | [43.31, 43.56] | 253.36 | [252.90, 253.82] | 0 |
| 5 | refine | 136500 | 25 | 5917.50 | [5801.00, 6034.00] | 5973.95 | [5856.16, 6091.74] | 81423.43 | 1.23 | false | 43.39 | [42.52, 44.26] | 252.70 | [250.00, 255.40] | 0 |
| 6 | refine | 132250 | 25 | 5838.53 | [5671.38, 6005.69] | 5898.30 | [5729.04, 6067.56] | 168147.97 | -0.05 | false | 43.03 | [41.75, 44.30] | 251.88 | [248.07, 255.68] | 0 |
| 7 | refine | 130125 | 25 | 5821.50 | [5671.39, 5971.60] | 5881.35 | [5729.36, 6033.34] | 135578.60 | -0.34 | false | 42.99 | [41.88, 44.09] | 252.00 | [248.67, 255.33] | 0 |
