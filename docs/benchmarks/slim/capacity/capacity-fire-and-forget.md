# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-11-05 18:27:58

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

Best offered aggregate rate: `144000` msg/sec
Estimated capacity offered-rate interval: `[144000, 146000]` msg/sec
Best observed node throughput: `7667.46` msg/sec with 95% CI [7645.15, 7689.77]
Best sender-completed throughput: `7588.82` msg/sec with 95% CI [7563.41, 7614.23]
Best node CPU: `44.04` % with 95% CI [43.89, 44.19]
Best total CPU: `253.05` % with 95% CI [252.37, 253.73]
Stop reason: refinement narrowed the estimated capacity to offered rates 144000 through 146000

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7564.30 | [7511.13, 7617.48] | 7346.06 | [6712.37, 7979.75] | 2356772.38 | 0.00 | true | 43.57 | [43.10, 44.03] | 250.58 | [248.57, 252.59] | 0 |
| 2 | coarse | 144000 | 25 | 7588.82 | [7563.41, 7614.23] | 7667.46 | [7645.15, 7689.77] | 2922.28 | 4.38 | true | 44.04 | [43.89, 44.19] | 253.05 | [252.37, 253.73] | 0 |
| 3 | coarse | 162000 | 25 | 7603.04 | [7580.42, 7625.66] | 7686.66 | [7662.14, 7711.18] | 3528.95 | 0.25 | false | 44.06 | [43.95, 44.18] | 252.60 | [252.12, 253.08] | 0 |
| 4 | coarse | 176000 | 25 | 7546.05 | [7517.50, 7574.59] | 7633.73 | [7608.81, 7658.66] | 3646.19 | -0.44 | false | 43.69 | [43.51, 43.88] | 251.17 | [250.40, 251.94] | 0 |
| 5 | refine | 160000 | 25 | 7550.40 | [7528.43, 7572.37] | 7628.58 | [7606.56, 7650.59] | 2844.45 | -0.51 | false | 43.65 | [43.50, 43.80] | 251.12 | [250.48, 251.76] | 0 |
| 6 | refine | 152000 | 25 | 7541.26 | [7502.28, 7580.24] | 7616.97 | [7577.78, 7656.16] | 9015.32 | -0.66 | false | 43.67 | [43.53, 43.81] | 251.00 | [250.34, 251.67] | 0 |
| 7 | refine | 148000 | 25 | 7572.36 | [7549.94, 7594.77] | 7665.37 | [7648.34, 7682.41] | 1702.76 | -0.03 | false | 43.69 | [43.56, 43.83] | 251.23 | [250.64, 251.82] | 0 |
| 8 | refine | 146000 | 25 | 7578.05 | [7550.23, 7605.87] | 7663.71 | [7637.67, 7689.75] | 3980.05 | -0.05 | false | 43.85 | [43.69, 44.00] | 251.80 | [251.09, 252.50] | 0 |
