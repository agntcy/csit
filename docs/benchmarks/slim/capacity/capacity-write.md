# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-17 09:50:02

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
Best sender write throughput: `5763.59` msg/sec with 95% CI [5747.57, 5779.60]
Best sender-completed throughput: `5763.59` msg/sec with 95% CI [5747.57, 5779.60]
Best node CPU: `43.56` % with 95% CI [43.41, 43.70]
Best total CPU: `184.32` % with 95% CI [183.96, 184.68]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5763.59 | [5747.57, 5779.60] | 5763.59 | [5747.57, 5779.60] | 1504.98 | 0.00 | true | 43.56 | [43.41, 43.70] | 184.32 | [183.96, 184.68] | 0 |
| 2 | coarse | 144000 | 25 | 5697.58 | [5678.82, 5716.33] | 5697.58 | [5678.82, 5716.33] | 2063.47 | -1.15 | false | 43.32 | [43.17, 43.47] | 183.47 | [183.05, 183.88] | 0 |
| 3 | coarse | 162000 | 25 | 5614.98 | [5446.09, 5783.86] | 5614.98 | [5446.09, 5783.86] | 167390.32 | -2.58 | false | 42.57 | [41.29, 43.84] | 182.20 | [180.34, 184.07] | 0 |
| 4 | refine | 145000 | 25 | 5650.34 | [5627.81, 5672.88] | 5650.34 | [5627.81, 5672.88] | 2979.85 | -1.96 | false | 42.83 | [42.67, 42.98] | 181.05 | [180.55, 181.54] | 0 |
| 5 | refine | 136500 | 25 | 5602.55 | [5569.38, 5635.71] | 5602.55 | [5569.38, 5635.71] | 6455.81 | -2.79 | false | 42.61 | [42.39, 42.83] | 180.53 | [179.78, 181.27] | 0 |
| 6 | refine | 132250 | 25 | 5595.30 | [5553.89, 5636.70] | 5595.30 | [5553.89, 5636.70] | 10061.59 | -2.92 | false | 42.31 | [41.92, 42.69] | 179.64 | [178.21, 181.07] | 0 |
| 7 | refine | 130125 | 25 | 5573.51 | [5440.51, 5706.52] | 5573.51 | [5440.51, 5706.52] | 103828.66 | -3.30 | false | 42.31 | [41.30, 43.31] | 180.94 | [179.43, 182.44] | 0 |
