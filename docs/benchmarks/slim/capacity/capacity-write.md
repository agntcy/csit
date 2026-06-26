# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-06-26 08:53:08

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
Best sender write throughput: `7255.44` msg/sec with 95% CI [7234.73, 7276.16]
Best sender-completed throughput: `7255.44` msg/sec with 95% CI [7234.73, 7276.16]
Best node CPU: `41.43` % with 95% CI [41.33, 41.54]
Best total CPU: `181.30` % with 95% CI [180.89, 181.70]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7255.44 | [7234.73, 7276.16] | 7255.44 | [7234.73, 7276.16] | 2517.47 | 0.00 | true | 41.43 | [41.33, 41.54] | 181.30 | [180.89, 181.70] | 0 |
| 2 | coarse | 144000 | 25 | 7286.01 | [7259.07, 7312.94] | 7286.01 | [7259.07, 7312.94] | 4257.16 | 0.42 | false | 41.63 | [41.51, 41.74] | 181.94 | [181.53, 182.35] | 0 |
| 3 | coarse | 162000 | 25 | 7171.80 | [7145.03, 7198.56] | 7171.80 | [7145.03, 7198.56] | 4204.32 | -1.15 | false | 41.36 | [41.22, 41.50] | 181.69 | [181.21, 182.16] | 0 |
| 4 | refine | 145000 | 25 | 7189.72 | [7172.24, 7207.19] | 7189.72 | [7172.24, 7207.19] | 1793.20 | -0.91 | false | 41.37 | [41.30, 41.44] | 181.58 | [181.31, 181.84] | 0 |
| 5 | refine | 136500 | 25 | 7214.24 | [7190.76, 7237.72] | 7214.24 | [7190.76, 7237.72] | 3235.67 | -0.57 | false | 41.56 | [41.47, 41.65] | 182.03 | [181.67, 182.38] | 0 |
| 6 | refine | 132250 | 25 | 7166.72 | [7130.80, 7202.64] | 7166.72 | [7130.80, 7202.64] | 7572.95 | -1.22 | false | 41.46 | [41.35, 41.57] | 181.87 | [181.44, 182.30] | 0 |
| 7 | refine | 130125 | 25 | 7244.22 | [7222.52, 7265.91] | 7244.22 | [7222.52, 7265.91] | 2763.23 | -0.15 | false | 41.56 | [41.44, 41.68] | 181.87 | [181.42, 182.31] | 0 |
