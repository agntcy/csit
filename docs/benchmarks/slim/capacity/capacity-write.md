# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-03 11:23:36

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
Best sender write throughput: `5913.18` msg/sec with 95% CI [5710.58, 6115.78]
Best sender-completed throughput: `5913.18` msg/sec with 95% CI [5710.58, 6115.78]
Best node CPU: `42.87` % with 95% CI [41.38, 44.36]
Best total CPU: `183.94` % with 95% CI [181.59, 186.28]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5913.18 | [5710.58, 6115.78] | 5913.18 | [5710.58, 6115.78] | 240901.11 | 0.00 | true | 42.87 | [41.38, 44.36] | 183.94 | [181.59, 186.28] | 0 |
| 2 | coarse | 144000 | 25 | 5982.80 | [5901.98, 6063.62] | 5982.80 | [5901.98, 6063.62] | 38338.06 | 1.18 | false | 43.62 | [43.05, 44.18] | 185.05 | [184.11, 185.99] | 0 |
| 3 | coarse | 162000 | 25 | 6046.60 | [6030.40, 6062.81] | 6046.60 | [6030.40, 6062.81] | 1540.88 | 2.26 | false | 44.09 | [43.96, 44.22] | 185.63 | [185.35, 185.91] | 0 |
| 4 | refine | 145000 | 25 | 6018.96 | [6003.41, 6034.51] | 6018.96 | [6003.41, 6034.51] | 1419.12 | 1.79 | false | 43.78 | [43.69, 43.87] | 185.21 | [184.85, 185.56] | 0 |
| 5 | refine | 136500 | 25 | 5949.41 | [5798.00, 6100.82] | 5949.41 | [5798.00, 6100.82] | 134547.97 | 0.61 | false | 43.66 | [42.55, 44.77] | 185.06 | [183.38, 186.73] | 0 |
| 6 | refine | 132250 | 25 | 6008.00 | [5991.26, 6024.74] | 6008.00 | [5991.26, 6024.74] | 1644.55 | 1.60 | false | 44.05 | [43.89, 44.20] | 185.50 | [185.18, 185.82] | 0 |
| 7 | refine | 130125 | 25 | 6016.57 | [5996.47, 6036.68] | 6016.57 | [5996.47, 6036.68] | 2372.76 | 1.75 | false | 43.88 | [43.73, 44.04] | 185.37 | [184.96, 185.77] | 0 |
