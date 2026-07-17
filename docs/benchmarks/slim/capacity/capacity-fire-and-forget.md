# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-17 09:16:30

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
Best observed node throughput: `5745.40` msg/sec with 95% CI [5699.31, 5791.49]
Best sender-completed throughput: `5681.49` msg/sec with 95% CI [5632.79, 5730.18]
Best node CPU: `43.30` % with 95% CI [42.75, 43.86]
Best total CPU: `251.57` % with 95% CI [249.11, 254.04]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5681.49 | [5632.79, 5730.18] | 5745.40 | [5699.31, 5791.49] | 12467.32 | 0.00 | true | 43.30 | [42.75, 43.86] | 251.57 | [249.11, 254.04] | 0 |
| 2 | coarse | 144000 | 25 | 5666.25 | [5646.11, 5686.39] | 5733.89 | [5715.86, 5751.92] | 1907.68 | -0.20 | false | 43.37 | [43.18, 43.55] | 250.90 | [250.16, 251.65] | 0 |
| 3 | coarse | 162000 | 25 | 5547.03 | [5430.21, 5663.85] | 5613.50 | [5493.95, 5733.04] | 83877.96 | -2.30 | false | 42.54 | [41.62, 43.46] | 247.84 | [245.20, 250.47] | 0 |
| 4 | refine | 145000 | 25 | 5630.32 | [5526.75, 5733.90] | 5690.50 | [5585.58, 5795.41] | 64596.72 | -0.96 | false | 43.22 | [42.41, 44.03] | 250.85 | [248.45, 253.25] | 0 |
| 5 | refine | 136500 | 25 | 5627.16 | [5473.30, 5781.02] | 5692.22 | [5537.75, 5846.70] | 140050.58 | -0.93 | false | 42.98 | [41.78, 44.17] | 251.07 | [247.52, 254.63] | 0 |
| 6 | refine | 132250 | 25 | 5594.41 | [5463.44, 5725.37] | 5659.56 | [5526.60, 5792.53] | 103760.63 | -1.49 | false | 43.12 | [42.17, 44.08] | 251.57 | [248.64, 254.49] | 0 |
| 7 | refine | 130125 | 25 | 5555.03 | [5410.85, 5699.21] | 5614.43 | [5468.07, 5760.79] | 125723.97 | -2.28 | false | 42.74 | [41.63, 43.86] | 249.69 | [246.40, 252.99] | 0 |
