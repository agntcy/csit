# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-15 09:07:25

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

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

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.39` msg/sec with 95% CI [12.38, 12.39]
Best sender-completed throughput: `12.17` msg/sec with 95% CI [12.16, 12.18]
Best node CPU: `0.57` % with 95% CI [0.53, 0.61]
Best total CPU: `4.39` % with 95% CI [4.28, 4.50]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.17 | [12.16, 12.18] | 12.39 | [12.38, 12.39] | 0.00 | 0.00 | true | 0.57 | [0.53, 0.61] | 4.39 | [4.28, 4.50] | 0 |
| 2 | coarse | 2000 | 25 | 12.16 | [12.14, 12.17] | 12.38 | [12.38, 12.39] | 0.00 | -0.01 | false | 0.56 | [0.50, 0.61] | 4.38 | [4.30, 4.46] | 0 |
| 3 | refine | 1500 | 25 | 12.16 | [12.15, 12.17] | 12.39 | [12.38, 12.39] | 0.00 | -0.00 | false | 0.56 | [0.53, 0.60] | 4.42 | [4.35, 4.48] | 0 |
| 4 | refine | 1250 | 25 | 12.17 | [12.16, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | 0.01 | false | 0.56 | [0.52, 0.61] | 4.37 | [4.30, 4.44] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `5937.01` msg/sec with 95% CI [5818.60, 6055.41]
Best sender-completed throughput: `5937.01` msg/sec with 95% CI [5818.60, 6055.41]
Best node CPU: `43.74` % with 95% CI [42.85, 44.62]
Best total CPU: `184.97` % with 95% CI [183.52, 186.41]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5937.01 | [5818.60, 6055.41] | 5937.01 | [5818.60, 6055.41] | 82283.13 | 0.00 | true | 43.74 | [42.85, 44.62] | 184.97 | [183.52, 186.41] | 0 |
| 2 | coarse | 144000 | 25 | 6006.92 | [5993.68, 6020.16] | 6006.92 | [5993.68, 6020.16] | 1029.04 | 1.18 | false | 44.16 | [44.04, 44.28] | 185.10 | [184.64, 185.55] | 0 |
| 3 | coarse | 162000 | 25 | 5980.18 | [5964.09, 5996.28] | 5980.18 | [5964.09, 5996.28] | 1520.82 | 0.73 | false | 44.16 | [44.03, 44.29] | 185.27 | [184.83, 185.71] | 0 |
| 4 | refine | 145000 | 25 | 5994.62 | [5981.53, 6007.70] | 5994.62 | [5981.53, 6007.70] | 1004.71 | 0.97 | false | 44.39 | [44.27, 44.52] | 185.43 | [185.03, 185.83] | 0 |
| 5 | refine | 136500 | 25 | 5963.70 | [5948.98, 5978.43] | 5963.70 | [5948.98, 5978.43] | 1272.35 | 0.45 | false | 44.23 | [44.12, 44.34] | 185.69 | [185.30, 186.08] | 0 |
| 6 | refine | 132250 | 25 | 5993.76 | [5979.29, 6008.22] | 5993.76 | [5979.29, 6008.22] | 1228.60 | 0.96 | false | 44.15 | [44.02, 44.28] | 185.01 | [184.53, 185.50] | 0 |
| 7 | refine | 130125 | 25 | 6020.98 | [6006.21, 6035.75] | 6020.98 | [6006.21, 6035.75] | 1281.10 | 1.41 | false | 44.34 | [44.18, 44.49] | 185.84 | [185.38, 186.30] | 0 |

