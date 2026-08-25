# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-08-25 09:07:49

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
Best observed node throughput: `5962.83` msg/sec with 95% CI [5920.99, 6004.68]
Best sender-completed throughput: `5904.41` msg/sec with 95% CI [5863.75, 5945.07]
Best node CPU: `43.80` % with 95% CI [43.35, 44.24]
Best total CPU: `254.03` % with 95% CI [251.88, 256.19]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5904.41 | [5863.75, 5945.07] | 5962.83 | [5920.99, 6004.68] | 10274.49 | 0.00 | true | 43.80 | [43.35, 44.24] | 254.03 | [251.88, 256.19] | 0 |
| 2 | coarse | 144000 | 25 | 5921.78 | [5909.58, 5933.98] | 5986.10 | [5976.27, 5995.94] | 567.91 | 0.39 | false | 44.00 | [43.90, 44.09] | 255.33 | [254.92, 255.75] | 0 |
| 3 | coarse | 162000 | 25 | 5810.24 | [5650.33, 5970.14] | 5871.34 | [5711.99, 6030.69] | 149026.92 | -1.53 | false | 43.37 | [42.14, 44.59] | 253.82 | [250.06, 257.57] | 0 |
| 4 | refine | 145000 | 25 | 5910.99 | [5891.41, 5930.57] | 5975.19 | [5957.52, 5992.86] | 1833.32 | 0.21 | false | 43.99 | [43.87, 44.11] | 255.39 | [254.90, 255.89] | 0 |
| 5 | refine | 136500 | 25 | 5891.65 | [5867.88, 5915.42] | 5953.81 | [5933.92, 5973.70] | 2321.53 | -0.15 | false | 43.96 | [43.81, 44.11] | 255.18 | [254.63, 255.74] | 0 |
| 6 | refine | 132250 | 25 | 5833.90 | [5814.48, 5853.31] | 5907.01 | [5888.03, 5925.99] | 2114.06 | -0.94 | false | 43.86 | [43.70, 44.02] | 254.54 | [253.91, 255.17] | 0 |
| 7 | refine | 130125 | 25 | 5812.83 | [5682.18, 5943.48] | 5868.75 | [5736.63, 6000.87] | 102452.63 | -1.58 | false | 43.52 | [42.54, 44.51] | 253.81 | [250.87, 256.74] | 0 |
