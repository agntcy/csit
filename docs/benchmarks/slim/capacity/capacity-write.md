# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-14 14:19:48

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
Best sender write throughput: `7361.18` msg/sec with 95% CI [7340.68, 7381.69]
Best sender-completed throughput: `7361.18` msg/sec with 95% CI [7340.68, 7381.69]
Best node CPU: `41.46` % with 95% CI [41.35, 41.56]
Best total CPU: `181.55` % with 95% CI [181.23, 181.87]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7361.18 | [7340.68, 7381.69] | 7361.18 | [7340.68, 7381.69] | 2467.76 | 0.00 | true | 41.46 | [41.35, 41.56] | 181.55 | [181.23, 181.87] | 0 |
| 2 | coarse | 144000 | 25 | 7357.45 | [7335.40, 7379.50] | 7357.45 | [7335.40, 7379.50] | 2854.49 | -0.05 | false | 41.26 | [41.14, 41.37] | 181.03 | [180.55, 181.50] | 0 |
| 3 | coarse | 162000 | 25 | 7336.79 | [7317.80, 7355.78] | 7336.79 | [7317.80, 7355.78] | 2116.64 | -0.33 | false | 41.19 | [41.05, 41.33] | 180.53 | [180.12, 180.93] | 0 |
| 4 | refine | 145000 | 25 | 7382.07 | [7360.22, 7403.91] | 7382.07 | [7360.22, 7403.91] | 2801.04 | 0.28 | false | 41.51 | [41.39, 41.64] | 181.87 | [181.48, 182.26] | 0 |
| 5 | refine | 136500 | 25 | 7384.32 | [7363.55, 7405.08] | 7384.32 | [7363.55, 7405.08] | 2530.62 | 0.31 | false | 41.48 | [41.33, 41.63] | 181.81 | [181.29, 182.32] | 0 |
| 6 | refine | 132250 | 25 | 7373.66 | [7354.16, 7393.15] | 7373.66 | [7354.16, 7393.15] | 2230.08 | 0.17 | false | 41.56 | [41.46, 41.66] | 181.99 | [181.65, 182.33] | 0 |
| 7 | refine | 130125 | 25 | 7343.36 | [7323.43, 7363.28] | 7343.36 | [7323.43, 7363.28] | 2329.99 | -0.24 | false | 41.48 | [41.34, 41.61] | 181.64 | [181.14, 182.14] | 0 |
