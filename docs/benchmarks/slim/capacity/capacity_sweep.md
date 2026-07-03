# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-03 12:54:17

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `6446.67` msg/sec with 95% CI [6388.54, 6504.80]
Best sender-completed throughput: `6377.02` msg/sec with 95% CI [6319.71, 6434.33]
Best node CPU: `43.76` % with 95% CI [43.22, 44.30]
Best total CPU: `248.20` % with 95% CI [245.89, 250.51]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6377.02 | [6319.71, 6434.33] | 6446.67 | [6388.54, 6504.80] | 19829.14 | 0.00 | true | 43.76 | [43.22, 44.30] | 248.20 | [245.89, 250.51] | 0 |
| 2 | coarse | 144000 | 25 | 6437.33 | [6420.45, 6454.21] | 6499.25 | [6484.20, 6514.29] | 1328.60 | 0.82 | false | 44.36 | [44.22, 44.50] | 250.71 | [250.15, 251.26] | 0 |
| 3 | coarse | 162000 | 25 | 6452.23 | [6414.35, 6490.12] | 6522.89 | [6482.66, 6563.12] | 9499.85 | 1.18 | false | 44.56 | [44.32, 44.80] | 251.34 | [250.63, 252.04] | 0 |
| 4 | refine | 145000 | 25 | 6468.88 | [6425.63, 6512.13] | 6531.24 | [6487.21, 6575.27] | 11377.86 | 1.31 | false | 44.56 | [44.25, 44.88] | 251.23 | [250.45, 252.02] | 0 |
| 5 | refine | 136500 | 25 | 6509.62 | [6493.30, 6525.94] | 6582.75 | [6571.33, 6594.17] | 765.42 | 2.11 | false | 44.74 | [44.60, 44.88] | 251.48 | [250.90, 252.06] | 0 |
| 6 | refine | 132250 | 25 | 6502.32 | [6472.50, 6532.14] | 6573.29 | [6544.26, 6602.32] | 4946.87 | 1.96 | false | 44.60 | [44.40, 44.81] | 251.34 | [250.65, 252.04] | 0 |
| 7 | refine | 130125 | 25 | 6514.35 | [6503.56, 6525.15] | 6585.41 | [6575.94, 6594.88] | 526.15 | 2.15 | false | 44.75 | [44.66, 44.85] | 251.87 | [251.45, 252.28] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.38` msg/sec with 95% CI [12.38, 12.38]
Best sender-completed throughput: `12.17` msg/sec with 95% CI [12.17, 12.17]
Best node CPU: `0.59` % with 95% CI [0.55, 0.62]
Best total CPU: `4.15` % with 95% CI [4.10, 4.20]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.17 | [12.17, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | 0.00 | true | 0.59 | [0.55, 0.62] | 4.15 | [4.10, 4.20] | 0 |
| 2 | coarse | 2000 | 25 | 12.17 | [12.17, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | 0.01 | false | 0.61 | [0.56, 0.66] | 4.20 | [4.14, 4.25] | 0 |
| 3 | refine | 1500 | 25 | 12.17 | [12.16, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | 0.01 | false | 0.62 | [0.57, 0.66] | 4.20 | [4.14, 4.27] | 0 |
| 4 | refine | 1250 | 25 | 12.16 | [12.15, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | -0.01 | false | 0.63 | [0.58, 0.67] | 4.24 | [4.19, 4.30] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `6468.05` msg/sec with 95% CI [6449.91, 6486.18]
Best sender-completed throughput: `6468.05` msg/sec with 95% CI [6449.91, 6486.18]
Best node CPU: `44.35` % with 95% CI [44.21, 44.50]
Best total CPU: `186.49` % with 95% CI [186.01, 186.98]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6468.05 | [6449.91, 6486.18] | 6468.05 | [6449.91, 6486.18] | 1931.17 | 0.00 | true | 44.35 | [44.21, 44.50] | 186.49 | [186.01, 186.98] | 0 |
| 2 | coarse | 144000 | 25 | 6442.04 | [6424.09, 6459.99] | 6442.04 | [6424.09, 6459.99] | 1891.50 | -0.40 | false | 44.09 | [43.95, 44.23] | 185.53 | [185.06, 185.99] | 0 |
| 3 | coarse | 162000 | 25 | 6466.02 | [6453.40, 6478.64] | 6466.02 | [6453.40, 6478.64] | 935.12 | -0.03 | false | 44.49 | [44.38, 44.59] | 186.97 | [186.61, 187.33] | 0 |
| 4 | refine | 145000 | 25 | 6466.84 | [6445.72, 6487.96] | 6466.84 | [6445.72, 6487.96] | 2617.94 | -0.02 | false | 44.48 | [44.33, 44.62] | 187.01 | [186.55, 187.47] | 0 |
| 5 | refine | 136500 | 25 | 6478.42 | [6458.10, 6498.74] | 6478.42 | [6458.10, 6498.74] | 2423.41 | 0.16 | false | 44.57 | [44.37, 44.76] | 186.96 | [186.39, 187.54] | 0 |
| 6 | refine | 132250 | 25 | 6493.29 | [6464.25, 6522.32] | 6493.29 | [6464.25, 6522.32] | 4949.18 | 0.39 | false | 44.66 | [44.42, 44.90] | 187.49 | [187.10, 187.87] | 0 |
| 7 | refine | 130125 | 25 | 6501.71 | [6487.49, 6515.93] | 6501.71 | [6487.49, 6515.93] | 1186.78 | 0.52 | false | 44.60 | [44.49, 44.72] | 187.35 | [186.98, 187.72] | 0 |

