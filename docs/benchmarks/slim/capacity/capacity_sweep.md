# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-08-25 09:41:17

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

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

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.38` msg/sec with 95% CI [12.38, 12.39]
Best sender-completed throughput: `12.16` msg/sec with 95% CI [12.15, 12.17]
Best node CPU: `0.56` % with 95% CI [0.52, 0.61]
Best total CPU: `4.39` % with 95% CI [4.31, 4.46]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.16 | [12.15, 12.17] | 12.38 | [12.38, 12.39] | 0.00 | 0.00 | true | 0.56 | [0.52, 0.61] | 4.39 | [4.31, 4.46] | 0 |
| 2 | coarse | 2000 | 25 | 12.16 | [12.14, 12.17] | 12.39 | [12.38, 12.39] | 0.00 | 0.03 | false | 0.57 | [0.53, 0.61] | 4.37 | [4.32, 4.42] | 0 |
| 3 | refine | 1500 | 25 | 12.16 | [12.15, 12.17] | 12.38 | [12.38, 12.39] | 0.00 | 0.02 | false | 0.56 | [0.54, 0.59] | 4.41 | [4.35, 4.46] | 0 |
| 4 | refine | 1250 | 25 | 12.17 | [12.16, 12.18] | 12.39 | [12.38, 12.39] | 0.00 | 0.02 | false | 0.57 | [0.53, 0.62] | 4.41 | [4.35, 4.48] | 0 |

## Write Mode

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

