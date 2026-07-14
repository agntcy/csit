# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-14 14:14:23

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `6431.80` msg/sec with 95% CI [6378.66, 6484.93]
Best sender-completed throughput: `6362.59` msg/sec with 95% CI [6306.52, 6418.66]
Best node CPU: `43.88` % with 95% CI [43.36, 44.41]
Best total CPU: `248.25` % with 95% CI [246.03, 250.47]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6362.59 | [6306.52, 6418.66] | 6431.80 | [6378.66, 6484.93] | 16570.57 | 0.00 | true | 43.88 | [43.36, 44.41] | 248.25 | [246.03, 250.47] | 0 |
| 2 | coarse | 144000 | 25 | 6404.47 | [6384.56, 6424.38] | 6482.46 | [6469.52, 6495.40] | 982.79 | 0.79 | false | 44.32 | [44.15, 44.49] | 249.94 | [249.31, 250.57] | 0 |
| 3 | coarse | 162000 | 25 | 6416.07 | [6402.20, 6429.94] | 6492.10 | [6478.15, 6506.04] | 1141.59 | 0.94 | false | 44.37 | [44.22, 44.52] | 250.20 | [249.53, 250.88] | 0 |
| 4 | refine | 145000 | 25 | 6422.48 | [6408.93, 6436.04] | 6498.77 | [6488.36, 6509.19] | 636.64 | 1.04 | false | 44.51 | [44.38, 44.64] | 250.83 | [250.40, 251.27] | 0 |
| 5 | refine | 136500 | 25 | 6414.00 | [6397.75, 6430.25] | 6492.26 | [6478.45, 6506.07] | 1119.42 | 0.94 | false | 44.49 | [44.38, 44.61] | 250.60 | [250.10, 251.10] | 0 |
| 6 | refine | 132250 | 25 | 6424.40 | [6410.32, 6438.49] | 6498.48 | [6487.13, 6509.83] | 755.81 | 1.04 | false | 44.60 | [44.48, 44.73] | 251.16 | [250.58, 251.73] | 0 |
| 7 | refine | 130125 | 25 | 6429.56 | [6417.61, 6441.51] | 6491.37 | [6479.89, 6502.86] | 773.53 | 0.93 | false | 44.63 | [44.53, 44.73] | 251.40 | [251.06, 251.74] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.38` msg/sec with 95% CI [12.38, 12.38]
Best sender-completed throughput: `12.16` msg/sec with 95% CI [12.14, 12.17]
Best node CPU: `0.58` % with 95% CI [0.52, 0.63]
Best total CPU: `4.19` % with 95% CI [4.14, 4.25]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.16 | [12.14, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | 0.00 | true | 0.58 | [0.52, 0.63] | 4.19 | [4.14, 4.25] | 0 |
| 2 | coarse | 2000 | 25 | 12.16 | [12.15, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | -0.01 | false | 0.60 | [0.54, 0.65] | 4.20 | [4.14, 4.26] | 0 |
| 3 | refine | 1500 | 25 | 12.16 | [12.16, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | -0.00 | false | 0.61 | [0.57, 0.65] | 4.18 | [4.13, 4.23] | 0 |
| 4 | refine | 1250 | 25 | 12.16 | [12.15, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | -0.01 | false | 0.61 | [0.56, 0.66] | 4.25 | [4.19, 4.30] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `6401.74` msg/sec with 95% CI [6383.82, 6419.65]
Best sender-completed throughput: `6401.74` msg/sec with 95% CI [6383.82, 6419.65]
Best node CPU: `44.58` % with 95% CI [44.44, 44.72]
Best total CPU: `187.01` % with 95% CI [186.68, 187.34]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6401.74 | [6383.82, 6419.65] | 6401.74 | [6383.82, 6419.65] | 1884.58 | 0.00 | true | 44.58 | [44.44, 44.72] | 187.01 | [186.68, 187.34] | 0 |
| 2 | coarse | 144000 | 25 | 6392.71 | [6377.31, 6408.12] | 6392.71 | [6377.31, 6408.12] | 1393.08 | -0.14 | false | 44.47 | [44.35, 44.59] | 186.74 | [186.36, 187.12] | 0 |
| 3 | coarse | 162000 | 25 | 6408.67 | [6395.93, 6421.42] | 6408.67 | [6395.93, 6421.42] | 952.70 | 0.11 | false | 44.49 | [44.37, 44.61] | 187.05 | [186.75, 187.36] | 0 |
| 4 | refine | 145000 | 25 | 6399.10 | [6383.35, 6414.86] | 6399.10 | [6383.35, 6414.86] | 1456.81 | -0.04 | false | 44.45 | [44.32, 44.58] | 186.81 | [186.42, 187.21] | 0 |
| 5 | refine | 136500 | 25 | 6406.97 | [6387.95, 6425.98] | 6406.97 | [6387.95, 6425.98] | 2122.48 | 0.08 | false | 44.51 | [44.38, 44.63] | 186.91 | [186.55, 187.28] | 0 |
| 6 | refine | 132250 | 25 | 6432.66 | [6421.62, 6443.69] | 6432.66 | [6421.62, 6443.69] | 714.69 | 0.48 | false | 44.58 | [44.49, 44.68] | 187.46 | [187.14, 187.78] | 0 |
| 7 | refine | 130125 | 25 | 6421.58 | [6410.78, 6432.38] | 6421.58 | [6410.78, 6432.38] | 684.59 | 0.31 | false | 44.60 | [44.48, 44.71] | 187.29 | [187.00, 187.59] | 0 |

