# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-14 14:39:55

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `9100.60` msg/sec with 95% CI [9052.67, 9148.54]
Best sender-completed throughput: `8997.85` msg/sec with 95% CI [8952.24, 9043.45]
Best node CPU: `44.76` % with 95% CI [44.39, 45.12]
Best total CPU: `250.60` % with 95% CI [249.13, 252.07]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 8997.85 | [8952.24, 9043.45] | 9100.60 | [9052.67, 9148.54] | 13484.93 | 0.00 | true | 44.76 | [44.39, 45.12] | 250.60 | [249.13, 252.07] | 0 |
| 2 | coarse | 144000 | 25 | 9010.99 | [8985.42, 9036.56] | 9116.75 | [9093.90, 9139.61] | 3065.61 | 0.18 | false | 44.83 | [44.65, 45.01] | 251.03 | [250.29, 251.76] | 0 |
| 3 | coarse | 162000 | 25 | 9021.74 | [8993.85, 9049.63] | 9127.51 | [9100.38, 9154.65] | 4320.09 | 0.30 | false | 45.15 | [45.02, 45.28] | 252.30 | [251.68, 252.91] | 0 |
| 4 | refine | 145000 | 25 | 8992.09 | [8969.41, 9014.78] | 9079.98 | [9058.03, 9101.93] | 2827.81 | -0.23 | false | 44.71 | [44.57, 44.84] | 250.39 | [249.74, 251.03] | 0 |
| 5 | refine | 136500 | 25 | 8922.67 | [8885.29, 8960.05] | 9041.30 | [9014.25, 9068.35] | 4293.57 | -0.65 | false | 44.83 | [44.64, 45.01] | 250.59 | [249.82, 251.36] | 0 |
| 6 | refine | 132250 | 25 | 8881.90 | [8837.79, 8926.00] | 8974.72 | [8929.18, 9020.26] | 12171.38 | -1.38 | false | 44.83 | [44.69, 44.97] | 250.64 | [249.97, 251.31] | 0 |
| 7 | refine | 130125 | 25 | 8947.11 | [8905.17, 8989.06] | 9040.43 | [9001.22, 9079.64] | 9024.00 | -0.66 | false | 45.11 | [44.97, 45.24] | 252.31 | [251.67, 252.96] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.39` msg/sec with 95% CI [12.39, 12.39]
Best sender-completed throughput: `12.16` msg/sec with 95% CI [12.14, 12.18]
Best node CPU: `0.37` % with 95% CI [0.31, 0.42]
Best total CPU: `2.62` % with 95% CI [2.55, 2.69]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.16 | [12.14, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | 0.00 | true | 0.37 | [0.31, 0.42] | 2.62 | [2.55, 2.69] | 0 |
| 2 | coarse | 2000 | 25 | 12.17 | [12.16, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | -0.00 | false | 0.36 | [0.31, 0.42] | 2.63 | [2.56, 2.71] | 0 |
| 3 | refine | 1500 | 25 | 12.18 | [12.17, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | -0.00 | false | 0.39 | [0.35, 0.44] | 2.75 | [2.69, 2.81] | 0 |
| 4 | refine | 1250 | 25 | 12.16 | [12.15, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | -0.00 | false | 0.42 | [0.39, 0.45] | 2.87 | [2.81, 2.92] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `9006.38` msg/sec with 95% CI [8972.17, 9040.60]
Best sender-completed throughput: `9006.38` msg/sec with 95% CI [8972.17, 9040.60]
Best node CPU: `44.80` % with 95% CI [44.68, 44.91]
Best total CPU: `180.17` % with 95% CI [179.79, 180.55]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 9006.38 | [8972.17, 9040.60] | 9006.38 | [8972.17, 9040.60] | 6869.90 | 0.00 | true | 44.80 | [44.68, 44.91] | 180.17 | [179.79, 180.55] | 0 |
| 2 | coarse | 144000 | 25 | 9032.38 | [9008.49, 9056.27] | 9032.38 | [9008.49, 9056.27] | 3349.75 | 0.29 | false | 44.63 | [44.51, 44.76] | 179.52 | [179.03, 180.00] | 0 |
| 3 | coarse | 162000 | 25 | 8992.72 | [8962.32, 9023.12] | 8992.72 | [8962.32, 9023.12] | 5423.25 | -0.15 | false | 44.81 | [44.65, 44.97] | 180.09 | [179.57, 180.60] | 0 |
| 4 | refine | 145000 | 25 | 8843.55 | [8785.67, 8901.44] | 8843.55 | [8785.67, 8901.44] | 19665.25 | -1.81 | false | 44.89 | [44.75, 45.02] | 180.10 | [179.63, 180.58] | 0 |
| 5 | refine | 136500 | 25 | 8962.63 | [8913.45, 9011.82] | 8962.63 | [8913.45, 9011.82] | 14198.72 | -0.49 | false | 44.99 | [44.85, 45.12] | 180.28 | [179.88, 180.67] | 0 |
| 6 | refine | 132250 | 25 | 9070.21 | [9053.23, 9087.19] | 9070.21 | [9053.23, 9087.19] | 1691.79 | 0.71 | false | 44.94 | [44.77, 45.11] | 180.19 | [179.69, 180.69] | 0 |
| 7 | refine | 130125 | 25 | 9014.86 | [8987.73, 9041.98] | 9014.86 | [8987.73, 9041.98] | 4318.67 | 0.09 | false | 44.68 | [44.50, 44.87] | 179.15 | [178.56, 179.73] | 0 |

