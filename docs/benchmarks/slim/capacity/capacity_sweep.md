# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-29 09:54:55

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best observed node throughput: `8320.86` msg/sec with 95% CI [8293.06, 8348.65]
Best sender-completed throughput: `8235.18` msg/sec with 95% CI [8206.09, 8264.27]
Best node CPU: `44.04` % with 95% CI [43.82, 44.26]
Best total CPU: `247.33` % with 95% CI [246.38, 248.28]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 8235.18 | [8206.09, 8264.27] | 8320.86 | [8293.06, 8348.65] | 4534.14 | 0.00 | true | 44.04 | [43.82, 44.26] | 247.33 | [246.38, 248.28] | 0 |
| 2 | coarse | 144000 | 25 | 8256.29 | [8212.69, 8299.89] | 8344.79 | [8300.60, 8388.98] | 11461.02 | 0.29 | false | 44.29 | [44.05, 44.53] | 248.75 | [247.98, 249.53] | 0 |
| 3 | coarse | 162000 | 25 | 8272.73 | [8252.97, 8292.48] | 8358.58 | [8342.34, 8374.82] | 1548.10 | 0.45 | false | 44.48 | [44.34, 44.62] | 248.89 | [248.34, 249.44] | 0 |
| 4 | refine | 145000 | 25 | 8282.17 | [8265.06, 8299.27] | 8373.53 | [8359.89, 8387.16] | 1091.33 | 0.63 | false | 44.43 | [44.33, 44.54] | 249.06 | [248.57, 249.56] | 0 |
| 5 | refine | 136500 | 25 | 8253.25 | [8203.32, 8303.18] | 8336.39 | [8285.72, 8387.06] | 15067.60 | 0.19 | false | 44.18 | [43.88, 44.48] | 248.49 | [247.69, 249.29] | 0 |
| 6 | refine | 132250 | 25 | 8263.84 | [8247.55, 8280.13] | 8362.82 | [8351.10, 8374.55] | 807.13 | 0.50 | false | 44.21 | [44.10, 44.33] | 248.17 | [247.64, 248.69] | 0 |
| 7 | refine | 130125 | 25 | 8257.89 | [8240.81, 8274.98] | 8351.32 | [8336.48, 8366.17] | 1293.17 | 0.37 | false | 44.28 | [44.17, 44.39] | 248.35 | [247.79, 248.90] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.39` msg/sec with 95% CI [12.39, 12.39]
Best sender-completed throughput: `12.17` msg/sec with 95% CI [12.16, 12.18]
Best node CPU: `0.50` % with 95% CI [0.45, 0.55]
Best total CPU: `3.37` % with 95% CI [3.31, 3.42]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.17 | [12.16, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | 0.00 | true | 0.50 | [0.45, 0.55] | 3.37 | [3.31, 3.42] | 0 |
| 2 | coarse | 2000 | 25 | 12.17 | [12.16, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | 0.01 | false | 0.49 | [0.45, 0.54] | 3.31 | [3.25, 3.37] | 0 |
| 3 | refine | 1500 | 25 | 12.18 | [12.18, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | 0.01 | false | 0.50 | [0.46, 0.54] | 3.34 | [3.29, 3.40] | 0 |
| 4 | refine | 1250 | 25 | 12.17 | [12.17, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | 0.01 | false | 0.51 | [0.47, 0.55] | 3.35 | [3.30, 3.40] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `8230.47` msg/sec with 95% CI [8150.78, 8310.16]
Best sender-completed throughput: `8230.47` msg/sec with 95% CI [8150.78, 8310.16]
Best node CPU: `44.22` % with 95% CI [43.75, 44.69]
Best total CPU: `185.22` % with 95% CI [184.53, 185.90]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 8230.47 | [8150.78, 8310.16] | 8230.47 | [8150.78, 8310.16] | 37268.72 | 0.00 | true | 44.22 | [43.75, 44.69] | 185.22 | [184.53, 185.90] | 0 |
| 2 | coarse | 144000 | 25 | 8272.66 | [8253.64, 8291.68] | 8272.66 | [8253.64, 8291.68] | 2123.01 | 0.51 | false | 44.33 | [44.21, 44.46] | 184.91 | [184.45, 185.37] | 0 |
| 3 | coarse | 162000 | 25 | 8293.42 | [8271.92, 8314.92] | 8293.42 | [8271.92, 8314.92] | 2713.14 | 0.76 | false | 44.57 | [44.45, 44.68] | 185.97 | [185.67, 186.26] | 0 |
| 4 | refine | 145000 | 25 | 8276.13 | [8238.33, 8313.93] | 8276.13 | [8238.33, 8313.93] | 8385.09 | 0.55 | false | 44.47 | [44.27, 44.67] | 185.80 | [185.41, 186.19] | 0 |
| 5 | refine | 136500 | 25 | 8316.77 | [8303.16, 8330.39] | 8316.77 | [8303.16, 8330.39] | 1088.16 | 1.05 | false | 44.62 | [44.52, 44.73] | 186.11 | [185.80, 186.42] | 0 |
| 6 | refine | 132250 | 25 | 8305.85 | [8291.44, 8320.25] | 8305.85 | [8291.44, 8320.25] | 1218.36 | 0.92 | false | 44.46 | [44.31, 44.60] | 185.75 | [185.40, 186.10] | 0 |
| 7 | refine | 130125 | 25 | 8274.86 | [8217.12, 8332.61] | 8274.86 | [8217.12, 8332.61] | 19569.18 | 0.54 | false | 44.27 | [43.95, 44.60] | 185.70 | [185.18, 186.23] | 0 |

