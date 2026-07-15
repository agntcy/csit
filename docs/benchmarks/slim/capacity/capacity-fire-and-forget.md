# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-15 08:33:57

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
Best observed node throughput: `5850.11` msg/sec with 95% CI [5695.13, 6005.10]
Best sender-completed throughput: `5784.93` msg/sec with 95% CI [5633.10, 5936.75]
Best node CPU: `42.67` % with 95% CI [41.46, 43.89]
Best total CPU: `249.82` % with 95% CI [245.76, 253.89]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5784.93 | [5633.10, 5936.75] | 5850.11 | [5695.13, 6005.10] | 140973.11 | 0.00 | true | 42.67 | [41.46, 43.89] | 249.82 | [245.76, 253.89] | 0 |
| 2 | coarse | 144000 | 25 | 5821.43 | [5651.99, 5990.88] | 5887.22 | [5715.52, 6058.92] | 173030.12 | 0.63 | false | 43.33 | [42.04, 44.61] | 253.01 | [249.13, 256.88] | 0 |
| 3 | coarse | 162000 | 25 | 5896.46 | [5885.39, 5907.54] | 5959.07 | [5948.54, 5969.60] | 650.75 | 1.86 | false | 44.01 | [43.89, 44.12] | 255.09 | [254.66, 255.51] | 0 |
| 4 | refine | 145000 | 25 | 5898.26 | [5882.19, 5914.33] | 5970.32 | [5955.16, 5985.47] | 1348.59 | 2.05 | false | 43.93 | [43.79, 44.06] | 254.58 | [254.01, 255.16] | 0 |
| 5 | refine | 136500 | 25 | 5931.84 | [5910.65, 5953.03] | 6000.29 | [5976.63, 6023.95] | 3286.22 | 2.57 | false | 43.99 | [43.84, 44.15] | 254.85 | [254.37, 255.33] | 0 |
| 6 | refine | 132250 | 25 | 5917.73 | [5786.12, 6049.35] | 5987.79 | [5853.78, 6121.80] | 105397.35 | 2.35 | false | 43.77 | [42.79, 44.76] | 253.72 | [250.78, 256.66] | 0 |
| 7 | refine | 130125 | 25 | 5928.12 | [5814.73, 6041.50] | 5990.40 | [5875.43, 6105.38] | 77585.60 | 2.40 | false | 43.85 | [42.98, 44.71] | 253.83 | [251.27, 256.38] | 0 |
