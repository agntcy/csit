# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-14 14:19:48

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

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

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.37` msg/sec with 95% CI [12.37, 12.38]
Best sender-completed throughput: `12.16` msg/sec with 95% CI [12.15, 12.16]
Best node CPU: `0.60` % with 95% CI [0.55, 0.65]
Best total CPU: `3.80` % with 95% CI [3.73, 3.87]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.16 | [12.15, 12.16] | 12.37 | [12.37, 12.38] | 0.00 | 0.00 | true | 0.60 | [0.55, 0.65] | 3.80 | [3.73, 3.87] | 0 |
| 2 | coarse | 2000 | 25 | 12.15 | [12.14, 12.16] | 12.37 | [12.37, 12.37] | 0.00 | -0.04 | false | 0.59 | [0.55, 0.64] | 3.80 | [3.74, 3.86] | 0 |
| 3 | refine | 1500 | 25 | 12.14 | [12.12, 12.15] | 12.36 | [12.36, 12.37] | 0.00 | -0.09 | false | 0.62 | [0.57, 0.67] | 3.85 | [3.79, 3.90] | 0 |
| 4 | refine | 1250 | 25 | 12.15 | [12.14, 12.16] | 12.37 | [12.36, 12.37] | 0.00 | -0.06 | false | 0.60 | [0.56, 0.64] | 3.82 | [3.77, 3.87] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `7361.18` msg/sec with 95% CI [7340.68, 7381.69]
Best sender-completed throughput: `7361.18` msg/sec with 95% CI [7340.68, 7381.69]
Best node CPU: `41.46` % with 95% CI [41.35, 41.56]
Best total CPU: `181.55` % with 95% CI [181.23, 181.87]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7361.18 | [7340.68, 7381.69] | 7361.18 | [7340.68, 7381.69] | 2467.76 | 0.00 | true | 41.46 | [41.35, 41.56] | 181.55 | [181.23, 181.87] | 0 |
| 2 | coarse | 144000 | 25 | 7357.45 | [7335.40, 7379.50] | 7357.45 | [7335.40, 7379.50] | 2854.49 | -0.05 | false | 41.26 | [41.14, 41.37] | 181.03 | [180.55, 181.50] | 0 |
| 3 | coarse | 162000 | 25 | 7336.79 | [7317.80, 7355.78] | 7336.79 | [7317.80, 7355.78] | 2116.64 | -0.33 | false | 41.19 | [41.05, 41.33] | 180.53 | [180.12, 180.93] | 0 |
| 4 | refine | 145000 | 25 | 7382.07 | [7360.22, 7403.91] | 7382.07 | [7360.22, 7403.91] | 2801.04 | 0.28 | false | 41.51 | [41.39, 41.64] | 181.87 | [181.48, 182.26] | 0 |
| 5 | refine | 136500 | 25 | 7384.32 | [7363.55, 7405.08] | 7384.32 | [7363.55, 7405.08] | 2530.62 | 0.31 | false | 41.48 | [41.33, 41.63] | 181.81 | [181.29, 182.32] | 0 |
| 6 | refine | 132250 | 25 | 7373.66 | [7354.16, 7393.15] | 7373.66 | [7354.16, 7393.15] | 2230.08 | 0.17 | false | 41.56 | [41.46, 41.66] | 181.99 | [181.65, 182.33] | 0 |
| 7 | refine | 130125 | 25 | 7343.36 | [7323.43, 7363.28] | 7343.36 | [7323.43, 7363.28] | 2329.99 | -0.24 | false | 41.48 | [41.34, 41.61] | 181.64 | [181.14, 182.14] | 0 |

