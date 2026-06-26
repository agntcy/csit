# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-06-26 08:53:08

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `7210.38` msg/sec with 95% CI [7174.95, 7245.80]
Best sender-completed throughput: `7130.47` msg/sec with 95% CI [7093.50, 7167.44]
Best node CPU: `41.15` % with 95% CI [40.91, 41.39]
Best total CPU: `246.58` % with 95% CI [245.57, 247.60]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7130.47 | [7093.50, 7167.44] | 7210.38 | [7174.95, 7245.80] | 7364.00 | 0.00 | true | 41.15 | [40.91, 41.39] | 246.58 | [245.57, 247.60] | 0 |
| 2 | coarse | 144000 | 25 | 7163.89 | [7134.92, 7192.86] | 7241.88 | [7215.27, 7268.48] | 4154.58 | 0.44 | false | 41.41 | [41.31, 41.51] | 247.78 | [247.29, 248.27] | 0 |
| 3 | coarse | 162000 | 25 | 7144.52 | [7124.39, 7164.65] | 7215.58 | [7194.94, 7236.22] | 2500.06 | 0.07 | false | 41.29 | [41.19, 41.38] | 247.37 | [246.80, 247.93] | 0 |
| 4 | refine | 145000 | 25 | 7118.52 | [7087.43, 7149.60] | 7209.75 | [7181.02, 7238.48] | 4844.33 | -0.01 | false | 41.34 | [41.25, 41.43] | 247.62 | [247.21, 248.04] | 0 |
| 5 | refine | 136500 | 25 | 7156.42 | [7133.80, 7179.05] | 7248.20 | [7226.31, 7270.10] | 2812.80 | 0.52 | false | 41.19 | [41.07, 41.30] | 246.55 | [245.88, 247.22] | 0 |
| 6 | refine | 132250 | 25 | 7141.96 | [7109.99, 7173.94] | 7210.67 | [7180.83, 7240.52] | 5226.64 | 0.00 | false | 41.14 | [41.00, 41.28] | 246.51 | [245.89, 247.14] | 0 |
| 7 | refine | 130125 | 25 | 7191.67 | [7162.79, 7220.56] | 7274.59 | [7241.60, 7307.58] | 6386.30 | 0.89 | false | 41.45 | [41.36, 41.53] | 247.96 | [247.53, 248.40] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.37` msg/sec with 95% CI [12.36, 12.37]
Best sender-completed throughput: `12.13` msg/sec with 95% CI [12.12, 12.15]
Best node CPU: `0.61` % with 95% CI [0.56, 0.66]
Best total CPU: `3.85` % with 95% CI [3.79, 3.91]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.13 | [12.12, 12.15] | 12.37 | [12.36, 12.37] | 0.00 | 0.00 | true | 0.61 | [0.56, 0.66] | 3.85 | [3.79, 3.91] | 0 |
| 2 | coarse | 2000 | 25 | 12.13 | [12.11, 12.15] | 12.37 | [12.36, 12.38] | 0.00 | -0.01 | false | 0.60 | [0.55, 0.65] | 3.81 | [3.74, 3.87] | 0 |
| 3 | refine | 1500 | 25 | 12.14 | [12.13, 12.16] | 12.37 | [12.37, 12.37] | 0.00 | 0.03 | false | 0.60 | [0.56, 0.65] | 3.85 | [3.79, 3.90] | 0 |
| 4 | refine | 1250 | 25 | 12.15 | [12.13, 12.16] | 12.37 | [12.37, 12.37] | 0.00 | 0.03 | false | 0.60 | [0.56, 0.64] | 3.86 | [3.82, 3.91] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `7255.44` msg/sec with 95% CI [7234.73, 7276.16]
Best sender-completed throughput: `7255.44` msg/sec with 95% CI [7234.73, 7276.16]
Best node CPU: `41.43` % with 95% CI [41.33, 41.54]
Best total CPU: `181.30` % with 95% CI [180.89, 181.70]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 7255.44 | [7234.73, 7276.16] | 7255.44 | [7234.73, 7276.16] | 2517.47 | 0.00 | true | 41.43 | [41.33, 41.54] | 181.30 | [180.89, 181.70] | 0 |
| 2 | coarse | 144000 | 25 | 7286.01 | [7259.07, 7312.94] | 7286.01 | [7259.07, 7312.94] | 4257.16 | 0.42 | false | 41.63 | [41.51, 41.74] | 181.94 | [181.53, 182.35] | 0 |
| 3 | coarse | 162000 | 25 | 7171.80 | [7145.03, 7198.56] | 7171.80 | [7145.03, 7198.56] | 4204.32 | -1.15 | false | 41.36 | [41.22, 41.50] | 181.69 | [181.21, 182.16] | 0 |
| 4 | refine | 145000 | 25 | 7189.72 | [7172.24, 7207.19] | 7189.72 | [7172.24, 7207.19] | 1793.20 | -0.91 | false | 41.37 | [41.30, 41.44] | 181.58 | [181.31, 181.84] | 0 |
| 5 | refine | 136500 | 25 | 7214.24 | [7190.76, 7237.72] | 7214.24 | [7190.76, 7237.72] | 3235.67 | -0.57 | false | 41.56 | [41.47, 41.65] | 182.03 | [181.67, 182.38] | 0 |
| 6 | refine | 132250 | 25 | 7166.72 | [7130.80, 7202.64] | 7166.72 | [7130.80, 7202.64] | 7572.95 | -1.22 | false | 41.46 | [41.35, 41.57] | 181.87 | [181.44, 182.30] | 0 |
| 7 | refine | 130125 | 25 | 7244.22 | [7222.52, 7265.91] | 7244.22 | [7222.52, 7265.91] | 2763.23 | -0.15 | false | 41.56 | [41.44, 41.68] | 181.87 | [181.42, 182.31] | 0 |

