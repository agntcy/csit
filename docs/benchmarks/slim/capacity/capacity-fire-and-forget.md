# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-29 09:21:29

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
Best observed node throughput: `8320.86` msg/sec with 95% CI [8293.06, 8348.65]
Best sender-completed throughput: `8235.18` msg/sec with 95% CI [8206.09, 8264.27]
Best node CPU: `44.04` % with 95% CI [43.82, 44.26]
Best total CPU: `247.33` % with 95% CI [246.38, 248.28]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 8235.18 | [8206.09, 8264.27] | 8320.86 | [8293.06, 8348.65] | 4534.14 | 0.00 | true | 44.04 | [43.82, 44.26] | 247.33 | [246.38, 248.28] | 0 |
| 2 | coarse | 144000 | 25 | 8256.29 | [8212.69, 8299.89] | 8344.79 | [8300.60, 8388.98] | 11461.02 | 0.29 | false | 44.29 | [44.05, 44.53] | 248.75 | [247.98, 249.53] | 0 |
| 3 | coarse | 162000 | 25 | 8272.73 | [8252.97, 8292.48] | 8358.58 | [8342.34, 8374.82] | 1548.10 | 0.45 | false | 44.48 | [44.34, 44.62] | 248.89 | [248.34, 249.44] | 0 |
| 4 | refine | 145000 | 25 | 8282.17 | [8265.06, 8299.27] | 8373.53 | [8359.89, 8387.16] | 1091.33 | 0.63 | false | 44.43 | [44.33, 44.54] | 249.06 | [248.57, 249.56] | 0 |
| 5 | refine | 136500 | 25 | 8253.25 | [8203.32, 8303.18] | 8336.39 | [8285.72, 8387.06] | 15067.60 | 0.19 | false | 44.18 | [43.88, 44.48] | 248.49 | [247.69, 249.29] | 0 |
| 6 | refine | 132250 | 25 | 8263.84 | [8247.55, 8280.13] | 8362.82 | [8351.10, 8374.55] | 807.13 | 0.50 | false | 44.21 | [44.10, 44.33] | 248.17 | [247.64, 248.69] | 0 |
| 7 | refine | 130125 | 25 | 8257.89 | [8240.81, 8274.98] | 8351.32 | [8336.48, 8366.17] | 1293.17 | 0.37 | false | 44.28 | [44.17, 44.39] | 248.35 | [247.79, 248.90] | 0 |
