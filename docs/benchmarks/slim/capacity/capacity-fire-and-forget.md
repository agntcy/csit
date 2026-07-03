# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-03 12:20:48

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
Best observed node throughput: `6446.67` msg/sec with 95% CI [6388.54, 6504.80]
Best sender-completed throughput: `6377.02` msg/sec with 95% CI [6319.71, 6434.33]
Best node CPU: `43.76` % with 95% CI [43.22, 44.30]
Best total CPU: `248.20` % with 95% CI [245.89, 250.51]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6377.02 | [6319.71, 6434.33] | 6446.67 | [6388.54, 6504.80] | 19829.14 | 0.00 | true | 43.76 | [43.22, 44.30] | 248.20 | [245.89, 250.51] | 0 |
| 2 | coarse | 144000 | 25 | 6437.33 | [6420.45, 6454.21] | 6499.25 | [6484.20, 6514.29] | 1328.60 | 0.82 | false | 44.36 | [44.22, 44.50] | 250.71 | [250.15, 251.26] | 0 |
| 3 | coarse | 162000 | 25 | 6452.23 | [6414.35, 6490.12] | 6522.89 | [6482.66, 6563.12] | 9499.85 | 1.18 | false | 44.56 | [44.32, 44.80] | 251.34 | [250.63, 252.04] | 0 |
| 4 | refine | 145000 | 25 | 6468.88 | [6425.63, 6512.13] | 6531.24 | [6487.21, 6575.27] | 11377.86 | 1.31 | false | 44.56 | [44.25, 44.88] | 251.23 | [250.45, 252.02] | 0 |
| 5 | refine | 136500 | 25 | 6509.62 | [6493.30, 6525.94] | 6582.75 | [6571.33, 6594.17] | 765.42 | 2.11 | false | 44.74 | [44.60, 44.88] | 251.48 | [250.90, 252.06] | 0 |
| 6 | refine | 132250 | 25 | 6502.32 | [6472.50, 6532.14] | 6573.29 | [6544.26, 6602.32] | 4946.87 | 1.96 | false | 44.60 | [44.40, 44.81] | 251.34 | [250.65, 252.04] | 0 |
| 7 | refine | 130125 | 25 | 6514.35 | [6503.56, 6525.15] | 6585.41 | [6575.94, 6594.88] | 526.15 | 2.15 | false | 44.75 | [44.66, 44.85] | 251.87 | [251.45, 252.28] | 0 |
