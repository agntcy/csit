# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-14 14:39:55

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
Best sender write throughput: `9006.38` msg/sec with 95% CI [8972.17, 9040.60]
Best sender-completed throughput: `9006.38` msg/sec with 95% CI [8972.17, 9040.60]
Best node CPU: `44.80` % with 95% CI [44.68, 44.91]
Best total CPU: `180.17` % with 95% CI [179.79, 180.55]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 9006.38 | [8972.17, 9040.60] | 9006.38 | [8972.17, 9040.60] | 6869.90 | 0.00 | true | 44.80 | [44.68, 44.91] | 180.17 | [179.79, 180.55] | 0 |
| 2 | coarse | 144000 | 25 | 9032.38 | [9008.49, 9056.27] | 9032.38 | [9008.49, 9056.27] | 3349.75 | 0.29 | false | 44.63 | [44.51, 44.76] | 179.52 | [179.03, 180.00] | 0 |
| 3 | coarse | 162000 | 25 | 8992.72 | [8962.32, 9023.12] | 8992.72 | [8962.32, 9023.12] | 5423.25 | -0.15 | false | 44.81 | [44.65, 44.97] | 180.09 | [179.57, 180.60] | 0 |
| 4 | refine | 145000 | 25 | 8843.55 | [8785.67, 8901.44] | 8843.55 | [8785.67, 8901.44] | 19665.25 | -1.81 | false | 44.89 | [44.75, 45.02] | 180.10 | [179.63, 180.58] | 0 |
| 5 | refine | 136500 | 25 | 8962.63 | [8913.45, 9011.82] | 8962.63 | [8913.45, 9011.82] | 14198.72 | -0.49 | false | 44.99 | [44.85, 45.12] | 180.28 | [179.88, 180.67] | 0 |
| 6 | refine | 132250 | 25 | 9070.21 | [9053.23, 9087.19] | 9070.21 | [9053.23, 9087.19] | 1691.79 | 0.71 | false | 44.94 | [44.77, 45.11] | 180.19 | [179.69, 180.69] | 0 |
| 7 | refine | 130125 | 25 | 9014.86 | [8987.73, 9041.98] | 9014.86 | [8987.73, 9041.98] | 4318.67 | 0.09 | false | 44.68 | [44.50, 44.87] | 179.15 | [178.56, 179.73] | 0 |
