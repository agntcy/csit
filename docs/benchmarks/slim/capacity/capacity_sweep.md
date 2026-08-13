# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-08-13 13:12:15

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `5901.35` msg/sec with 95% CI [5854.62, 5948.07]
Best sender-completed throughput: `5835.86` msg/sec with 95% CI [5787.74, 5883.98]
Best node CPU: `43.11` % with 95% CI [42.58, 43.63]
Best total CPU: `251.13` % with 95% CI [248.80, 253.46]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5835.86 | [5787.74, 5883.98] | 5901.35 | [5854.62, 5948.07] | 12812.29 | 0.00 | true | 43.11 | [42.58, 43.63] | 251.13 | [248.80, 253.46] | 0 |
| 2 | coarse | 144000 | 25 | 5894.79 | [5815.63, 5973.94] | 5962.59 | [5881.34, 6043.83] | 38737.87 | 1.04 | false | 43.42 | [42.79, 44.05] | 252.79 | [250.95, 254.62] | 0 |
| 3 | coarse | 162000 | 25 | 5909.57 | [5837.85, 5981.30] | 5975.65 | [5902.26, 6049.04] | 31612.43 | 1.26 | false | 43.47 | [42.91, 44.02] | 252.85 | [251.18, 254.51] | 0 |
| 4 | refine | 145000 | 25 | 5855.75 | [5840.33, 5871.16] | 5917.92 | [5902.64, 5933.19] | 1369.85 | 0.28 | false | 43.44 | [43.31, 43.56] | 253.36 | [252.90, 253.82] | 0 |
| 5 | refine | 136500 | 25 | 5917.50 | [5801.00, 6034.00] | 5973.95 | [5856.16, 6091.74] | 81423.43 | 1.23 | false | 43.39 | [42.52, 44.26] | 252.70 | [250.00, 255.40] | 0 |
| 6 | refine | 132250 | 25 | 5838.53 | [5671.38, 6005.69] | 5898.30 | [5729.04, 6067.56] | 168147.97 | -0.05 | false | 43.03 | [41.75, 44.30] | 251.88 | [248.07, 255.68] | 0 |
| 7 | refine | 130125 | 25 | 5821.50 | [5671.39, 5971.60] | 5881.35 | [5729.36, 6033.34] | 135578.60 | -0.34 | false | 42.99 | [41.88, 44.09] | 252.00 | [248.67, 255.33] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.36` msg/sec with 95% CI [12.36, 12.37]
Best sender-completed throughput: `12.15` msg/sec with 95% CI [12.14, 12.15]
Best node CPU: `0.70` % with 95% CI [0.66, 0.74]
Best total CPU: `5.06` % with 95% CI [5.00, 5.11]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.15 | [12.14, 12.15] | 12.36 | [12.36, 12.37] | 0.00 | 0.00 | true | 0.70 | [0.66, 0.74] | 5.06 | [5.00, 5.11] | 0 |
| 2 | coarse | 2000 | 25 | 12.13 | [12.12, 12.14] | 12.35 | [12.35, 12.36] | 0.00 | -0.09 | false | 0.73 | [0.68, 0.78] | 5.10 | [5.04, 5.17] | 0 |
| 3 | refine | 1500 | 25 | 12.14 | [12.13, 12.15] | 12.36 | [12.36, 12.36] | 0.00 | -0.02 | false | 0.73 | [0.70, 0.77] | 5.09 | [5.04, 5.15] | 0 |
| 4 | refine | 1250 | 25 | 12.13 | [12.11, 12.14] | 12.35 | [12.35, 12.36] | 0.00 | -0.08 | false | 0.73 | [0.69, 0.78] | 5.13 | [5.08, 5.17] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `5915.38` msg/sec with 95% CI [5893.07, 5937.70]
Best sender-completed throughput: `5915.38` msg/sec with 95% CI [5893.07, 5937.70]
Best node CPU: `43.66` % with 95% CI [43.52, 43.80]
Best total CPU: `184.34` % with 95% CI [183.94, 184.73]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5915.38 | [5893.07, 5937.70] | 5915.38 | [5893.07, 5937.70] | 2922.22 | 0.00 | true | 43.66 | [43.52, 43.80] | 184.34 | [183.94, 184.73] | 0 |
| 2 | coarse | 144000 | 25 | 5873.88 | [5850.75, 5897.02] | 5873.88 | [5850.75, 5897.02] | 3141.17 | -0.70 | false | 43.66 | [43.51, 43.82] | 184.48 | [184.02, 184.95] | 0 |
| 3 | coarse | 162000 | 25 | 5864.26 | [5842.18, 5886.33] | 5864.26 | [5842.18, 5886.33] | 2859.89 | -0.86 | false | 43.67 | [43.57, 43.77] | 184.77 | [184.48, 185.05] | 0 |
| 4 | refine | 145000 | 25 | 5883.18 | [5809.64, 5956.71] | 5883.18 | [5809.64, 5956.71] | 31735.06 | -0.54 | false | 43.48 | [42.91, 44.05] | 184.14 | [183.21, 185.07] | 0 |
| 5 | refine | 136500 | 25 | 5839.96 | [5707.94, 5971.99] | 5839.96 | [5707.94, 5971.99] | 102296.91 | -1.27 | false | 43.18 | [42.16, 44.20] | 183.60 | [181.98, 185.23] | 0 |
| 6 | refine | 132250 | 25 | 5834.85 | [5719.72, 5949.98] | 5834.85 | [5719.72, 5949.98] | 77796.14 | -1.36 | false | 43.19 | [42.31, 44.07] | 183.78 | [182.36, 185.21] | 0 |
| 7 | refine | 130125 | 25 | 5923.29 | [5903.61, 5942.96] | 5923.29 | [5903.61, 5942.96] | 2271.99 | 0.13 | false | 43.76 | [43.61, 43.91] | 184.66 | [184.28, 185.03] | 0 |

