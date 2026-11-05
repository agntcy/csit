# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-11-05 19:01:28

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `144000` msg/sec
Estimated capacity offered-rate interval: `[144000, 146000]` msg/sec
Best observed node throughput: `7667.46` msg/sec with 95% CI [7645.15, 7689.77]
Best sender-completed throughput: `7588.82` msg/sec with 95% CI [7563.41, 7614.23]
Best node CPU: `44.04` % with 95% CI [43.89, 44.19]
Best total CPU: `253.05` % with 95% CI [252.37, 253.73]
Stop reason: refinement narrowed the estimated capacity to offered rates 144000 through 146000

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7564.30 | [7511.13, 7617.48] | 7346.06 | [6712.37, 7979.75] | 2356772.38 | 0.00 | true | 43.57 | [43.10, 44.03] | 250.58 | [248.57, 252.59] | 0 |
| 2 | coarse | 144000 | 25 | 7588.82 | [7563.41, 7614.23] | 7667.46 | [7645.15, 7689.77] | 2922.28 | 4.38 | true | 44.04 | [43.89, 44.19] | 253.05 | [252.37, 253.73] | 0 |
| 3 | coarse | 162000 | 25 | 7603.04 | [7580.42, 7625.66] | 7686.66 | [7662.14, 7711.18] | 3528.95 | 0.25 | false | 44.06 | [43.95, 44.18] | 252.60 | [252.12, 253.08] | 0 |
| 4 | coarse | 176000 | 25 | 7546.05 | [7517.50, 7574.59] | 7633.73 | [7608.81, 7658.66] | 3646.19 | -0.44 | false | 43.69 | [43.51, 43.88] | 251.17 | [250.40, 251.94] | 0 |
| 5 | refine | 160000 | 25 | 7550.40 | [7528.43, 7572.37] | 7628.58 | [7606.56, 7650.59] | 2844.45 | -0.51 | false | 43.65 | [43.50, 43.80] | 251.12 | [250.48, 251.76] | 0 |
| 6 | refine | 152000 | 25 | 7541.26 | [7502.28, 7580.24] | 7616.97 | [7577.78, 7656.16] | 9015.32 | -0.66 | false | 43.67 | [43.53, 43.81] | 251.00 | [250.34, 251.67] | 0 |
| 7 | refine | 148000 | 25 | 7572.36 | [7549.94, 7594.77] | 7665.37 | [7648.34, 7682.41] | 1702.76 | -0.03 | false | 43.69 | [43.56, 43.83] | 251.23 | [250.64, 251.82] | 0 |
| 8 | refine | 146000 | 25 | 7578.05 | [7550.23, 7605.87] | 7663.71 | [7637.67, 7689.75] | 3980.05 | -0.05 | false | 43.85 | [43.69, 44.00] | 251.80 | [251.09, 252.50] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.36` msg/sec with 95% CI [12.35, 12.37]
Best sender-completed throughput: `12.14` msg/sec with 95% CI [12.12, 12.16]
Best node CPU: `0.62` % with 95% CI [0.59, 0.65]
Best total CPU: `3.82` % with 95% CI [3.79, 3.86]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.14 | [12.12, 12.16] | 12.36 | [12.35, 12.37] | 0.00 | 0.00 | true | 0.62 | [0.59, 0.65] | 3.82 | [3.79, 3.86] | 0 |
| 2 | coarse | 2000 | 25 | 12.15 | [12.15, 12.15] | 12.36 | [12.36, 12.37] | 0.00 | 0.03 | false | 0.62 | [0.58, 0.66] | 3.84 | [3.80, 3.89] | 0 |
| 3 | refine | 1500 | 25 | 12.14 | [12.13, 12.15] | 12.36 | [12.35, 12.37] | 0.00 | -0.02 | false | 0.62 | [0.57, 0.67] | 3.88 | [3.83, 3.94] | 0 |
| 4 | refine | 1250 | 25 | 12.14 | [12.13, 12.15] | 11.87 | [10.85, 12.89] | 6.11 | -3.97 | false | 0.61 | [0.58, 0.64] | 3.81 | [3.77, 3.85] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `7541.90` msg/sec with 95% CI [7513.97, 7569.83]
Best sender-completed throughput: `7541.90` msg/sec with 95% CI [7513.97, 7569.83]
Best node CPU: `43.86` % with 95% CI [43.69, 44.02]
Best total CPU: `179.04` % with 95% CI [178.43, 179.65]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7541.90 | [7513.97, 7569.83] | 7541.90 | [7513.97, 7569.83] | 4579.02 | 0.00 | true | 43.86 | [43.69, 44.02] | 179.04 | [178.43, 179.65] | 0 |
| 2 | coarse | 144000 | 25 | 7461.07 | [7426.17, 7495.97] | 7461.07 | [7426.17, 7495.97] | 7148.26 | -1.07 | false | 43.93 | [43.73, 44.12] | 179.67 | [179.13, 180.21] | 0 |
| 3 | coarse | 162000 | 25 | 7438.10 | [7404.95, 7471.25] | 7438.10 | [7404.95, 7471.25] | 6449.87 | -1.38 | false | 43.72 | [43.53, 43.91] | 178.93 | [178.31, 179.55] | 0 |
| 4 | refine | 145000 | 25 | 7431.06 | [7408.81, 7453.30] | 7431.06 | [7408.81, 7453.30] | 2904.73 | -1.47 | false | 43.99 | [43.87, 44.11] | 179.92 | [179.51, 180.32] | 0 |
| 5 | refine | 136500 | 25 | 7484.59 | [7459.17, 7510.00] | 7484.59 | [7459.17, 7510.00] | 3790.75 | -0.76 | false | 44.02 | [43.84, 44.20] | 179.85 | [179.28, 180.42] | 0 |
| 6 | refine | 132250 | 25 | 7458.90 | [7427.99, 7489.80] | 7458.90 | [7427.99, 7489.80] | 5606.12 | -1.10 | false | 43.78 | [43.65, 43.91] | 179.09 | [178.64, 179.54] | 0 |
| 7 | refine | 130125 | 25 | 7489.17 | [7457.91, 7520.42] | 7489.17 | [7457.91, 7520.42] | 5733.07 | -0.70 | false | 43.88 | [43.72, 44.04] | 179.24 | [178.74, 179.74] | 0 |

