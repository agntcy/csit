# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-11-23 13:22:17

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `6404.47` msg/sec with 95% CI [6360.77, 6448.17]
Best sender-completed throughput: `6336.83` msg/sec with 95% CI [6291.38, 6382.29]
Best node CPU: `44.35` % with 95% CI [43.84, 44.87]
Best total CPU: `254.49` % with 95% CI [252.35, 256.64]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6336.83 | [6291.38, 6382.29] | 6404.47 | [6360.77, 6448.17] | 11209.37 | 0.00 | true | 44.35 | [43.84, 44.87] | 254.49 | [252.35, 256.64] | 0 |
| 2 | coarse | 144000 | 25 | 6385.77 | [6335.80, 6435.73] | 6456.03 | [6405.23, 6506.83] | 15146.69 | 0.81 | false | 44.62 | [44.21, 45.03] | 255.18 | [253.34, 257.01] | 0 |
| 3 | coarse | 162000 | 25 | 6405.74 | [6366.83, 6444.64] | 6473.93 | [6436.12, 6511.74] | 8391.12 | 1.08 | false | 44.65 | [44.30, 44.99] | 255.55 | [254.05, 257.04] | 0 |
| 4 | refine | 145000 | 25 | 6270.47 | [6216.08, 6324.86] | 6341.52 | [6286.93, 6396.11] | 17492.47 | -0.98 | false | 43.67 | [43.25, 44.08] | 251.09 | [249.19, 252.99] | 0 |
| 5 | refine | 136500 | 25 | 6392.65 | [6346.08, 6439.23] | 6460.75 | [6414.87, 6506.63] | 12354.68 | 0.88 | false | 44.50 | [44.08, 44.93] | 254.69 | [252.86, 256.51] | 0 |
| 6 | refine | 132250 | 25 | 6410.79 | [6378.16, 6443.42] | 6475.01 | [6443.35, 6506.67] | 5881.85 | 1.10 | false | 44.81 | [44.53, 45.09] | 256.28 | [255.06, 257.51] | 0 |
| 7 | refine | 130125 | 25 | 6405.73 | [6362.24, 6449.22] | 6474.14 | [6428.30, 6519.99] | 12337.58 | 1.09 | false | 44.68 | [44.30, 45.05] | 255.63 | [254.05, 257.22] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.37` msg/sec with 95% CI [12.37, 12.37]
Best sender-completed throughput: `12.15` msg/sec with 95% CI [12.14, 12.16]
Best node CPU: `0.61` % with 95% CI [0.56, 0.66]
Best total CPU: `4.03` % with 95% CI [3.94, 4.12]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.15 | [12.14, 12.16] | 12.37 | [12.37, 12.37] | 0.00 | 0.00 | true | 0.61 | [0.56, 0.66] | 4.03 | [3.94, 4.12] | 0 |
| 2 | coarse | 2000 | 25 | 12.15 | [12.14, 12.16] | 12.37 | [12.36, 12.37] | 0.00 | -0.02 | false | 0.65 | [0.59, 0.70] | 4.14 | [4.04, 4.24] | 0 |
| 3 | refine | 1500 | 25 | 12.14 | [12.14, 12.15] | 12.36 | [12.36, 12.37] | 0.00 | -0.06 | false | 0.68 | [0.62, 0.74] | 4.26 | [4.16, 4.36] | 0 |
| 4 | refine | 1250 | 25 | 12.13 | [12.13, 12.14] | 12.35 | [12.34, 12.35] | 0.00 | -0.17 | false | 0.68 | [0.61, 0.75] | 4.29 | [4.15, 4.43] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `6373.09` msg/sec with 95% CI [6331.23, 6414.96]
Best sender-completed throughput: `6373.09` msg/sec with 95% CI [6331.23, 6414.96]
Best node CPU: `44.79` % with 95% CI [44.42, 45.16]
Best total CPU: `187.84` % with 95% CI [186.48, 189.19]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6373.09 | [6331.23, 6414.96] | 6373.09 | [6331.23, 6414.96] | 10288.09 | 0.00 | true | 44.79 | [44.42, 45.16] | 187.84 | [186.48, 189.19] | 0 |
| 2 | coarse | 144000 | 25 | 6434.60 | [6403.13, 6466.08] | 6434.60 | [6403.13, 6466.08] | 5814.50 | 0.97 | false | 45.02 | [44.74, 45.31] | 188.40 | [187.49, 189.31] | 0 |
| 3 | coarse | 162000 | 25 | 6404.29 | [6355.02, 6453.57] | 6404.29 | [6355.02, 6453.57] | 14249.56 | 0.49 | false | 44.86 | [44.39, 45.33] | 187.58 | [185.99, 189.16] | 0 |
| 4 | refine | 145000 | 25 | 6397.26 | [6348.24, 6446.29] | 6397.26 | [6348.24, 6446.29] | 14107.39 | 0.38 | false | 44.85 | [44.42, 45.28] | 187.48 | [185.96, 189.00] | 0 |
| 5 | refine | 136500 | 25 | 6450.31 | [6411.42, 6489.21] | 6450.31 | [6411.42, 6489.21] | 8877.49 | 1.21 | false | 45.14 | [44.77, 45.52] | 188.08 | [186.86, 189.31] | 0 |
| 6 | refine | 132250 | 25 | 6434.66 | [6393.72, 6475.60] | 6434.66 | [6393.72, 6475.60] | 9837.80 | 0.97 | false | 45.05 | [44.72, 45.39] | 188.16 | [187.07, 189.26] | 0 |
| 7 | refine | 130125 | 25 | 6410.01 | [6364.68, 6455.35] | 6410.01 | [6364.68, 6455.35] | 12063.41 | 0.58 | false | 44.76 | [44.36, 45.16] | 187.33 | [186.03, 188.63] | 0 |

