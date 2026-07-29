# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-29 09:54:55

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
Best sender write throughput: `8230.47` msg/sec with 95% CI [8150.78, 8310.16]
Best sender-completed throughput: `8230.47` msg/sec with 95% CI [8150.78, 8310.16]
Best node CPU: `44.22` % with 95% CI [43.75, 44.69]
Best total CPU: `185.22` % with 95% CI [184.53, 185.90]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 8230.47 | [8150.78, 8310.16] | 8230.47 | [8150.78, 8310.16] | 37268.72 | 0.00 | true | 44.22 | [43.75, 44.69] | 185.22 | [184.53, 185.90] | 0 |
| 2 | coarse | 144000 | 25 | 8272.66 | [8253.64, 8291.68] | 8272.66 | [8253.64, 8291.68] | 2123.01 | 0.51 | false | 44.33 | [44.21, 44.46] | 184.91 | [184.45, 185.37] | 0 |
| 3 | coarse | 162000 | 25 | 8293.42 | [8271.92, 8314.92] | 8293.42 | [8271.92, 8314.92] | 2713.14 | 0.76 | false | 44.57 | [44.45, 44.68] | 185.97 | [185.67, 186.26] | 0 |
| 4 | refine | 145000 | 25 | 8276.13 | [8238.33, 8313.93] | 8276.13 | [8238.33, 8313.93] | 8385.09 | 0.55 | false | 44.47 | [44.27, 44.67] | 185.80 | [185.41, 186.19] | 0 |
| 5 | refine | 136500 | 25 | 8316.77 | [8303.16, 8330.39] | 8316.77 | [8303.16, 8330.39] | 1088.16 | 1.05 | false | 44.62 | [44.52, 44.73] | 186.11 | [185.80, 186.42] | 0 |
| 6 | refine | 132250 | 25 | 8305.85 | [8291.44, 8320.25] | 8305.85 | [8291.44, 8320.25] | 1218.36 | 0.92 | false | 44.46 | [44.31, 44.60] | 185.75 | [185.40, 186.10] | 0 |
| 7 | refine | 130125 | 25 | 8274.86 | [8217.12, 8332.61] | 8274.86 | [8217.12, 8332.61] | 19569.18 | 0.54 | false | 44.27 | [43.95, 44.60] | 185.70 | [185.18, 186.23] | 0 |
