# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-17 09:50:02

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `5745.40` msg/sec with 95% CI [5699.31, 5791.49]
Best sender-completed throughput: `5681.49` msg/sec with 95% CI [5632.79, 5730.18]
Best node CPU: `43.30` % with 95% CI [42.75, 43.86]
Best total CPU: `251.57` % with 95% CI [249.11, 254.04]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5681.49 | [5632.79, 5730.18] | 5745.40 | [5699.31, 5791.49] | 12467.32 | 0.00 | true | 43.30 | [42.75, 43.86] | 251.57 | [249.11, 254.04] | 0 |
| 2 | coarse | 144000 | 25 | 5666.25 | [5646.11, 5686.39] | 5733.89 | [5715.86, 5751.92] | 1907.68 | -0.20 | false | 43.37 | [43.18, 43.55] | 250.90 | [250.16, 251.65] | 0 |
| 3 | coarse | 162000 | 25 | 5547.03 | [5430.21, 5663.85] | 5613.50 | [5493.95, 5733.04] | 83877.96 | -2.30 | false | 42.54 | [41.62, 43.46] | 247.84 | [245.20, 250.47] | 0 |
| 4 | refine | 145000 | 25 | 5630.32 | [5526.75, 5733.90] | 5690.50 | [5585.58, 5795.41] | 64596.72 | -0.96 | false | 43.22 | [42.41, 44.03] | 250.85 | [248.45, 253.25] | 0 |
| 5 | refine | 136500 | 25 | 5627.16 | [5473.30, 5781.02] | 5692.22 | [5537.75, 5846.70] | 140050.58 | -0.93 | false | 42.98 | [41.78, 44.17] | 251.07 | [247.52, 254.63] | 0 |
| 6 | refine | 132250 | 25 | 5594.41 | [5463.44, 5725.37] | 5659.56 | [5526.60, 5792.53] | 103760.63 | -1.49 | false | 43.12 | [42.17, 44.08] | 251.57 | [248.64, 254.49] | 0 |
| 7 | refine | 130125 | 25 | 5555.03 | [5410.85, 5699.21] | 5614.43 | [5468.07, 5760.79] | 125723.97 | -2.28 | false | 42.74 | [41.63, 43.86] | 249.69 | [246.40, 252.99] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.35` msg/sec with 95% CI [12.34, 12.36]
Best sender-completed throughput: `12.12` msg/sec with 95% CI [12.11, 12.14]
Best node CPU: `0.73` % with 95% CI [0.69, 0.78]
Best total CPU: `5.16` % with 95% CI [5.08, 5.23]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.12 | [12.11, 12.14] | 12.35 | [12.34, 12.36] | 0.00 | 0.00 | true | 0.73 | [0.69, 0.78] | 5.16 | [5.08, 5.23] | 0 |
| 2 | coarse | 2000 | 25 | 12.13 | [12.11, 12.14] | 12.35 | [12.35, 12.36] | 0.00 | 0.03 | false | 0.72 | [0.67, 0.77] | 5.13 | [5.05, 5.20] | 0 |
| 3 | refine | 1500 | 25 | 12.12 | [12.11, 12.13] | 12.34 | [12.34, 12.35] | 0.00 | -0.06 | false | 0.77 | [0.73, 0.82] | 5.28 | [5.20, 5.35] | 0 |
| 4 | refine | 1250 | 25 | 12.11 | [12.09, 12.12] | 12.34 | [12.33, 12.34] | 0.00 | -0.10 | false | 0.77 | [0.72, 0.82] | 5.31 | [5.23, 5.40] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `5763.59` msg/sec with 95% CI [5747.57, 5779.60]
Best sender-completed throughput: `5763.59` msg/sec with 95% CI [5747.57, 5779.60]
Best node CPU: `43.56` % with 95% CI [43.41, 43.70]
Best total CPU: `184.32` % with 95% CI [183.96, 184.68]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5763.59 | [5747.57, 5779.60] | 5763.59 | [5747.57, 5779.60] | 1504.98 | 0.00 | true | 43.56 | [43.41, 43.70] | 184.32 | [183.96, 184.68] | 0 |
| 2 | coarse | 144000 | 25 | 5697.58 | [5678.82, 5716.33] | 5697.58 | [5678.82, 5716.33] | 2063.47 | -1.15 | false | 43.32 | [43.17, 43.47] | 183.47 | [183.05, 183.88] | 0 |
| 3 | coarse | 162000 | 25 | 5614.98 | [5446.09, 5783.86] | 5614.98 | [5446.09, 5783.86] | 167390.32 | -2.58 | false | 42.57 | [41.29, 43.84] | 182.20 | [180.34, 184.07] | 0 |
| 4 | refine | 145000 | 25 | 5650.34 | [5627.81, 5672.88] | 5650.34 | [5627.81, 5672.88] | 2979.85 | -1.96 | false | 42.83 | [42.67, 42.98] | 181.05 | [180.55, 181.54] | 0 |
| 5 | refine | 136500 | 25 | 5602.55 | [5569.38, 5635.71] | 5602.55 | [5569.38, 5635.71] | 6455.81 | -2.79 | false | 42.61 | [42.39, 42.83] | 180.53 | [179.78, 181.27] | 0 |
| 6 | refine | 132250 | 25 | 5595.30 | [5553.89, 5636.70] | 5595.30 | [5553.89, 5636.70] | 10061.59 | -2.92 | false | 42.31 | [41.92, 42.69] | 179.64 | [178.21, 181.07] | 0 |
| 7 | refine | 130125 | 25 | 5573.51 | [5440.51, 5706.52] | 5573.51 | [5440.51, 5706.52] | 103828.66 | -3.30 | false | 42.31 | [41.30, 43.31] | 180.94 | [179.43, 182.44] | 0 |

