# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-14 13:46:21

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
Best observed node throughput: `7362.25` msg/sec with 95% CI [7300.92, 7423.58]
Best sender-completed throughput: `7271.47` msg/sec with 95% CI [7208.26, 7334.69]
Best node CPU: `41.17` % with 95% CI [40.70, 41.63]
Best total CPU: `246.52` % with 95% CI [244.42, 248.63]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7271.47 | [7208.26, 7334.69] | 7362.25 | [7300.92, 7423.58] | 22075.99 | 0.00 | true | 41.17 | [40.70, 41.63] | 246.52 | [244.42, 248.63] | 0 |
| 2 | coarse | 144000 | 25 | 7315.59 | [7291.80, 7339.39] | 7390.25 | [7368.38, 7412.12] | 2807.28 | 0.38 | false | 41.55 | [41.40, 41.69] | 248.15 | [247.46, 248.85] | 0 |
| 3 | coarse | 162000 | 25 | 7318.04 | [7288.98, 7347.10] | 7400.03 | [7373.75, 7426.30] | 4051.56 | 0.51 | false | 41.48 | [41.34, 41.63] | 247.68 | [247.05, 248.31] | 0 |
| 4 | refine | 145000 | 25 | 7354.85 | [7337.76, 7371.95] | 7429.91 | [7414.50, 7445.33] | 1395.19 | 0.92 | false | 41.69 | [41.57, 41.82] | 248.79 | [248.37, 249.20] | 0 |
| 5 | refine | 136500 | 25 | 7333.13 | [7307.37, 7358.89] | 7419.81 | [7397.49, 7442.12] | 2923.05 | 0.78 | false | 41.55 | [41.40, 41.70] | 247.96 | [247.30, 248.62] | 0 |
| 6 | refine | 132250 | 25 | 7362.15 | [7334.73, 7389.58] | 7437.37 | [7413.86, 7460.89] | 3245.83 | 1.02 | false | 41.50 | [41.36, 41.64] | 247.51 | [246.76, 248.27] | 0 |
| 7 | refine | 130125 | 25 | 7361.87 | [7340.05, 7383.70] | 7448.49 | [7430.17, 7466.81] | 1970.16 | 1.17 | false | 41.51 | [41.36, 41.66] | 248.02 | [247.33, 248.72] | 0 |
