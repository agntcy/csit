# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-03 10:50:08

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
Best observed node throughput: `6133.17` msg/sec with 95% CI [6103.20, 6163.14]
Best sender-completed throughput: `6066.99` msg/sec with 95% CI [6034.65, 6099.33]
Best node CPU: `43.74` % with 95% CI [43.41, 44.07]
Best total CPU: `253.84` % with 95% CI [252.37, 255.30]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6066.99 | [6034.65, 6099.33] | 6133.17 | [6103.20, 6163.14] | 5270.40 | 0.00 | true | 43.74 | [43.41, 44.07] | 253.84 | [252.37, 255.30] | 0 |
| 2 | coarse | 144000 | 25 | 6108.78 | [6088.92, 6128.65] | 6173.37 | [6154.03, 6192.71] | 2194.33 | 0.66 | false | 43.94 | [43.83, 44.04] | 254.59 | [254.12, 255.06] | 0 |
| 3 | coarse | 162000 | 25 | 6058.14 | [5980.10, 6136.18] | 6127.75 | [6049.64, 6205.86] | 35807.85 | -0.09 | false | 43.51 | [42.90, 44.12] | 252.46 | [250.58, 254.34] | 0 |
| 4 | refine | 145000 | 25 | 6041.10 | [6013.35, 6068.86] | 6114.96 | [6085.38, 6144.53] | 5132.29 | -0.30 | false | 43.84 | [43.69, 43.99] | 254.24 | [253.60, 254.88] | 0 |
| 5 | refine | 136500 | 25 | 6132.26 | [6107.43, 6157.08] | 6196.88 | [6172.45, 6221.30] | 3500.91 | 1.04 | false | 44.09 | [43.96, 44.22] | 254.89 | [254.37, 255.40] | 0 |
| 6 | refine | 132250 | 25 | 6135.30 | [6113.72, 6156.88] | 6201.81 | [6178.39, 6225.23] | 3219.04 | 1.12 | false | 44.10 | [43.99, 44.21] | 254.47 | [254.05, 254.89] | 0 |
| 7 | refine | 130125 | 25 | 6037.81 | [6016.02, 6059.60] | 6099.97 | [6079.60, 6120.33] | 2433.59 | -0.54 | false | 44.09 | [43.97, 44.22] | 254.78 | [254.32, 255.25] | 0 |
