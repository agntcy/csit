# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-08-13 13:12:15

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
Best sender write throughput: `5915.38` msg/sec with 95% CI [5893.07, 5937.70]
Best sender-completed throughput: `5915.38` msg/sec with 95% CI [5893.07, 5937.70]
Best node CPU: `43.66` % with 95% CI [43.52, 43.80]
Best total CPU: `184.34` % with 95% CI [183.94, 184.73]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5915.38 | [5893.07, 5937.70] | 5915.38 | [5893.07, 5937.70] | 2922.22 | 0.00 | true | 43.66 | [43.52, 43.80] | 184.34 | [183.94, 184.73] | 0 |
| 2 | coarse | 144000 | 25 | 5873.88 | [5850.75, 5897.02] | 5873.88 | [5850.75, 5897.02] | 3141.17 | -0.70 | false | 43.66 | [43.51, 43.82] | 184.48 | [184.02, 184.95] | 0 |
| 3 | coarse | 162000 | 25 | 5864.26 | [5842.18, 5886.33] | 5864.26 | [5842.18, 5886.33] | 2859.89 | -0.86 | false | 43.67 | [43.57, 43.77] | 184.77 | [184.48, 185.05] | 0 |
| 4 | refine | 145000 | 25 | 5883.18 | [5809.64, 5956.71] | 5883.18 | [5809.64, 5956.71] | 31735.06 | -0.54 | false | 43.48 | [42.91, 44.05] | 184.14 | [183.21, 185.07] | 0 |
| 5 | refine | 136500 | 25 | 5839.96 | [5707.94, 5971.99] | 5839.96 | [5707.94, 5971.99] | 102296.91 | -1.27 | false | 43.18 | [42.16, 44.20] | 183.60 | [181.98, 185.23] | 0 |
| 6 | refine | 132250 | 25 | 5834.85 | [5719.72, 5949.98] | 5834.85 | [5719.72, 5949.98] | 77796.14 | -1.36 | false | 43.19 | [42.31, 44.07] | 183.78 | [182.36, 185.21] | 0 |
| 7 | refine | 130125 | 25 | 5923.29 | [5903.61, 5942.96] | 5923.29 | [5903.61, 5942.96] | 2271.99 | 0.13 | false | 43.76 | [43.61, 43.91] | 184.66 | [184.28, 185.03] | 0 |
