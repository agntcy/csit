# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-03 11:23:36

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `6133.17` msg/sec with 95% CI [6103.20, 6163.14]
Best sender-completed throughput: `6066.99` msg/sec with 95% CI [6034.65, 6099.33]
Best node CPU: `43.74` % with 95% CI [43.41, 44.07]
Best total CPU: `253.84` % with 95% CI [252.37, 255.30]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6066.99 | [6034.65, 6099.33] | 6133.17 | [6103.20, 6163.14] | 5270.40 | 0.00 | true | 43.74 | [43.41, 44.07] | 253.84 | [252.37, 255.30] | 0 |
| 2 | coarse | 144000 | 25 | 6108.78 | [6088.92, 6128.65] | 6173.37 | [6154.03, 6192.71] | 2194.33 | 0.66 | false | 43.94 | [43.83, 44.04] | 254.59 | [254.12, 255.06] | 0 |
| 3 | coarse | 162000 | 25 | 6058.14 | [5980.10, 6136.18] | 6127.75 | [6049.64, 6205.86] | 35807.85 | -0.09 | false | 43.51 | [42.90, 44.12] | 252.46 | [250.58, 254.34] | 0 |
| 4 | refine | 145000 | 25 | 6041.10 | [6013.35, 6068.86] | 6114.96 | [6085.38, 6144.53] | 5132.29 | -0.30 | false | 43.84 | [43.69, 43.99] | 254.24 | [253.60, 254.88] | 0 |
| 5 | refine | 136500 | 25 | 6132.26 | [6107.43, 6157.08] | 6196.88 | [6172.45, 6221.30] | 3500.91 | 1.04 | false | 44.09 | [43.96, 44.22] | 254.89 | [254.37, 255.40] | 0 |
| 6 | refine | 132250 | 25 | 6135.30 | [6113.72, 6156.88] | 6201.81 | [6178.39, 6225.23] | 3219.04 | 1.12 | false | 44.10 | [43.99, 44.21] | 254.47 | [254.05, 254.89] | 0 |
| 7 | refine | 130125 | 25 | 6037.81 | [6016.02, 6059.60] | 6099.97 | [6079.60, 6120.33] | 2433.59 | -0.54 | false | 44.09 | [43.97, 44.22] | 254.78 | [254.32, 255.25] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.38` msg/sec with 95% CI [12.37, 12.38]
Best sender-completed throughput: `12.15` msg/sec with 95% CI [12.14, 12.16]
Best node CPU: `0.63` % with 95% CI [0.59, 0.68]
Best total CPU: `4.69` % with 95% CI [4.57, 4.80]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.15 | [12.14, 12.16] | 12.38 | [12.37, 12.38] | 0.00 | 0.00 | true | 0.63 | [0.59, 0.68] | 4.69 | [4.57, 4.80] | 0 |
| 2 | coarse | 2000 | 25 | 12.16 | [12.14, 12.17] | 12.38 | [12.37, 12.39] | 0.00 | 0.02 | false | 0.58 | [0.54, 0.62] | 4.47 | [4.35, 4.58] | 0 |
| 3 | refine | 1500 | 25 | 12.15 | [12.15, 12.15] | 12.36 | [12.36, 12.37] | 0.00 | -0.11 | false | 0.70 | [0.65, 0.74] | 4.96 | [4.89, 5.03] | 0 |
| 4 | refine | 1250 | 25 | 12.15 | [12.15, 12.16] | 12.37 | [12.36, 12.37] | 0.00 | -0.08 | false | 0.67 | [0.63, 0.72] | 4.88 | [4.81, 4.95] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `5913.18` msg/sec with 95% CI [5710.58, 6115.78]
Best sender-completed throughput: `5913.18` msg/sec with 95% CI [5710.58, 6115.78]
Best node CPU: `42.87` % with 95% CI [41.38, 44.36]
Best total CPU: `183.94` % with 95% CI [181.59, 186.28]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 5913.18 | [5710.58, 6115.78] | 5913.18 | [5710.58, 6115.78] | 240901.11 | 0.00 | true | 42.87 | [41.38, 44.36] | 183.94 | [181.59, 186.28] | 0 |
| 2 | coarse | 144000 | 25 | 5982.80 | [5901.98, 6063.62] | 5982.80 | [5901.98, 6063.62] | 38338.06 | 1.18 | false | 43.62 | [43.05, 44.18] | 185.05 | [184.11, 185.99] | 0 |
| 3 | coarse | 162000 | 25 | 6046.60 | [6030.40, 6062.81] | 6046.60 | [6030.40, 6062.81] | 1540.88 | 2.26 | false | 44.09 | [43.96, 44.22] | 185.63 | [185.35, 185.91] | 0 |
| 4 | refine | 145000 | 25 | 6018.96 | [6003.41, 6034.51] | 6018.96 | [6003.41, 6034.51] | 1419.12 | 1.79 | false | 43.78 | [43.69, 43.87] | 185.21 | [184.85, 185.56] | 0 |
| 5 | refine | 136500 | 25 | 5949.41 | [5798.00, 6100.82] | 5949.41 | [5798.00, 6100.82] | 134547.97 | 0.61 | false | 43.66 | [42.55, 44.77] | 185.06 | [183.38, 186.73] | 0 |
| 6 | refine | 132250 | 25 | 6008.00 | [5991.26, 6024.74] | 6008.00 | [5991.26, 6024.74] | 1644.55 | 1.60 | false | 44.05 | [43.89, 44.20] | 185.50 | [185.18, 185.82] | 0 |
| 7 | refine | 130125 | 25 | 6016.57 | [5996.47, 6036.68] | 6016.57 | [5996.47, 6036.68] | 2372.76 | 1.75 | false | 43.88 | [43.73, 44.04] | 185.37 | [184.96, 185.77] | 0 |

