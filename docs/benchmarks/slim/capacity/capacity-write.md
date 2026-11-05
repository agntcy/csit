# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-11-05 19:01:28

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
Best sender write throughput: `7541.90` msg/sec with 95% CI [7513.97, 7569.83]
Best sender-completed throughput: `7541.90` msg/sec with 95% CI [7513.97, 7569.83]
Best node CPU: `43.86` % with 95% CI [43.69, 44.02]
Best total CPU: `179.04` % with 95% CI [178.43, 179.65]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7541.90 | [7513.97, 7569.83] | 7541.90 | [7513.97, 7569.83] | 4579.02 | 0.00 | true | 43.86 | [43.69, 44.02] | 179.04 | [178.43, 179.65] | 0 |
| 2 | coarse | 144000 | 25 | 7461.07 | [7426.17, 7495.97] | 7461.07 | [7426.17, 7495.97] | 7148.26 | -1.07 | false | 43.93 | [43.73, 44.12] | 179.67 | [179.13, 180.21] | 0 |
| 3 | coarse | 162000 | 25 | 7438.10 | [7404.95, 7471.25] | 7438.10 | [7404.95, 7471.25] | 6449.87 | -1.38 | false | 43.72 | [43.53, 43.91] | 178.93 | [178.31, 179.55] | 0 |
| 4 | refine | 145000 | 25 | 7431.06 | [7408.81, 7453.30] | 7431.06 | [7408.81, 7453.30] | 2904.73 | -1.47 | false | 43.99 | [43.87, 44.11] | 179.92 | [179.51, 180.32] | 0 |
| 5 | refine | 136500 | 25 | 7484.59 | [7459.17, 7510.00] | 7484.59 | [7459.17, 7510.00] | 3790.75 | -0.76 | false | 44.02 | [43.84, 44.20] | 179.85 | [179.28, 180.42] | 0 |
| 6 | refine | 132250 | 25 | 7458.90 | [7427.99, 7489.80] | 7458.90 | [7427.99, 7489.80] | 5606.12 | -1.10 | false | 43.78 | [43.65, 43.91] | 179.09 | [178.64, 179.54] | 0 |
| 7 | refine | 130125 | 25 | 7489.17 | [7457.91, 7520.42] | 7489.17 | [7457.91, 7520.42] | 5733.07 | -0.70 | false | 43.88 | [43.72, 44.04] | 179.24 | [178.74, 179.74] | 0 |
