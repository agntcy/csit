# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-08-25 09:41:16

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
Best sender write throughput: `5863.05` msg/sec with 95% CI [5717.36, 6008.75]
Best sender-completed throughput: `5863.05` msg/sec with 95% CI [5717.36, 6008.75]
Best node CPU: `43.84` % with 95% CI [42.73, 44.96]
Best total CPU: `185.38` % with 95% CI [183.67, 187.08]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5863.05 | [5717.36, 6008.75] | 5863.05 | [5717.36, 6008.75] | 124584.47 | 0.00 | true | 43.84 | [42.73, 44.96] | 185.38 | [183.67, 187.08] | 0 |
| 2 | coarse | 144000 | 25 | 5900.58 | [5884.52, 5916.65] | 5900.58 | [5884.52, 5916.65] | 1514.70 | 0.64 | false | 44.38 | [44.25, 44.52] | 186.12 | [185.74, 186.51] | 0 |
| 3 | coarse | 162000 | 25 | 5926.28 | [5907.94, 5944.62] | 5926.28 | [5907.94, 5944.62] | 1973.76 | 1.08 | false | 44.45 | [44.33, 44.58] | 186.39 | [186.10, 186.69] | 0 |
| 4 | refine | 145000 | 25 | 5946.42 | [5927.24, 5965.60] | 5946.42 | [5927.24, 5965.60] | 2158.81 | 1.42 | false | 44.52 | [44.38, 44.66] | 186.41 | [185.99, 186.83] | 0 |
| 5 | refine | 136500 | 25 | 5835.29 | [5681.56, 5989.03] | 5835.29 | [5681.56, 5989.03] | 138716.06 | -0.47 | false | 43.72 | [42.52, 44.92] | 185.27 | [183.40, 187.15] | 0 |
| 6 | refine | 132250 | 25 | 5916.55 | [5890.87, 5942.24] | 5916.55 | [5890.87, 5942.24] | 3872.53 | 0.91 | false | 44.26 | [44.13, 44.39] | 186.03 | [185.58, 186.47] | 0 |
| 7 | refine | 130125 | 25 | 5866.36 | [5794.49, 5938.22] | 5866.36 | [5794.49, 5938.22] | 30310.92 | 0.06 | false | 43.93 | [43.38, 44.48] | 185.72 | [184.82, 186.61] | 0 |
