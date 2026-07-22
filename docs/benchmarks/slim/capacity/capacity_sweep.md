# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-22 09:51:34

This CI report combines the sink-backed capacity sweeps and the write capacity sweep into one markdown artifact.

## Sink-Backed Modes

#### Fire-And-Forget Clients=1 Payload=16384B

Best offered aggregate rate: `162000` msg/sec
Estimated capacity offered-rate interval: `[162000, 162875]` msg/sec
Best observed node throughput: `12293.09` msg/sec with 95% CI [12162.70, 12423.48]
Best sender-completed throughput: `12247.99` msg/sec with 95% CI [12181.42, 12314.57]
Best node CPU: `43.12` % with 95% CI [42.57, 43.67]
Best total CPU: `239.88` % with 95% CI [238.23, 241.53]
Stop reason: refinement narrowed the estimated capacity to offered rates 162000 through 162875

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 11929.68 | [11765.58, 12093.77] | 11916.16 | [11681.60, 12150.72] | 322902.13 | 0.00 | true | 43.32 | [42.42, 44.22] | 240.24 | [237.48, 243.01] | 0 |
| 2 | coarse | 144000 | 25 | 11802.23 | [11722.37, 11882.09] | 11940.22 | [11855.43, 12025.00] | 42191.00 | 0.20 | false | 43.21 | [43.07, 43.35] | 240.49 | [240.10, 240.89] | 0 |
| 3 | coarse | 162000 | 25 | 12247.99 | [12181.42, 12314.57] | 12293.09 | [12162.70, 12423.48] | 99787.26 | 3.16 | true | 43.12 | [42.57, 43.67] | 239.88 | [238.23, 241.53] | 0 |
| 4 | coarse | 176000 | 25 | 12144.39 | [11987.69, 12301.09] | 12242.63 | [12088.98, 12396.27] | 138543.40 | -0.41 | false | 43.74 | [43.53, 43.96] | 241.08 | [240.51, 241.66] | 0 |
| 5 | refine | 169000 | 25 | 12235.33 | [12147.17, 12323.50] | 12367.25 | [12281.06, 12453.45] | 43606.47 | 0.60 | false | 43.64 | [43.49, 43.80] | 240.42 | [239.86, 240.98] | 0 |
| 6 | refine | 165500 | 25 | 12030.93 | [11829.19, 12232.67] | 12151.64 | [11944.33, 12358.95] | 252238.13 | -1.15 | false | 43.49 | [43.29, 43.70] | 240.83 | [240.44, 241.23] | 0 |
| 7 | refine | 163750 | 25 | 12385.66 | [12338.82, 12432.50] | 12508.87 | [12456.70, 12561.05] | 15977.51 | 1.76 | false | 43.59 | [43.49, 43.69] | 240.59 | [240.22, 240.95] | 0 |
| 8 | refine | 162875 | 25 | 12181.21 | [12087.89, 12274.52] | 11772.35 | [10751.75, 12792.94] | 6113203.90 | -4.24 | false | 43.29 | [42.96, 43.61] | 240.27 | [239.39, 241.16] | 0 |

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.39` msg/sec with 95% CI [12.38, 12.39]
Best sender-completed throughput: `12.17` msg/sec with 95% CI [12.16, 12.18]
Best node CPU: `0.42` % with 95% CI [0.37, 0.46]
Best total CPU: `2.90` % with 95% CI [2.78, 3.01]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.17 | [12.16, 12.18] | 12.39 | [12.38, 12.39] | 0.00 | 0.00 | true | 0.42 | [0.37, 0.46] | 2.90 | [2.78, 3.01] | 0 |
| 2 | coarse | 2000 | 25 | 12.18 | [12.17, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | 0.02 | false | 0.37 | [0.32, 0.42] | 2.68 | [2.57, 2.79] | 0 |
| 3 | refine | 1500 | 25 | 12.17 | [12.16, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | 0.02 | false | 0.33 | [0.27, 0.39] | 2.56 | [2.41, 2.72] | 0 |
| 4 | refine | 1250 | 25 | 12.18 | [12.18, 12.18] | 12.39 | [12.39, 12.39] | 0.00 | 0.03 | false | 0.31 | [0.26, 0.35] | 2.40 | [2.32, 2.49] | 0 |

## Write Mode

#### Write Clients=1 Payload=16384B

Best offered aggregate rate: `128000` msg/sec
Estimated capacity offered-rate interval: `[128000, 130125]` msg/sec
Best sender write throughput: `12384.84` msg/sec with 95% CI [12307.99, 12461.69]
Best sender-completed throughput: `12384.84` msg/sec with 95% CI [12307.99, 12461.69]
Best node CPU: `43.43` % with 95% CI [43.28, 43.58]
Best total CPU: `178.46` % with 95% CI [178.15, 178.77]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Sender Write Throughput | Sender Write Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 12384.84 | [12307.99, 12461.69] | 12384.84 | [12307.99, 12461.69] | 34660.65 | 0.00 | true | 43.43 | [43.28, 43.58] | 178.46 | [178.15, 178.77] | 0 |
| 2 | coarse | 144000 | 25 | 12353.80 | [12250.45, 12457.16] | 12353.80 | [12250.45, 12457.16] | 62694.69 | -0.25 | false | 43.04 | [42.37, 43.71] | 177.77 | [176.63, 178.92] | 0 |
| 3 | coarse | 162000 | 25 | 12470.05 | [12396.38, 12543.73] | 12470.05 | [12396.38, 12543.73] | 31856.97 | 0.69 | false | 43.56 | [43.42, 43.71] | 178.67 | [178.25, 179.08] | 0 |
| 4 | refine | 145000 | 25 | 11963.84 | [11823.52, 12104.17] | 11963.84 | [11823.52, 12104.17] | 115566.58 | -3.40 | false | 43.20 | [43.03, 43.37] | 178.67 | [178.24, 179.11] | 0 |
| 5 | refine | 136500 | 25 | 12373.14 | [12285.93, 12460.34] | 12373.14 | [12285.93, 12460.34] | 44634.77 | -0.09 | false | 43.57 | [43.39, 43.74] | 178.56 | [178.15, 178.98] | 0 |
| 6 | refine | 132250 | 25 | 12351.56 | [12284.83, 12418.29] | 12351.56 | [12284.83, 12418.29] | 26131.97 | -0.27 | false | 43.67 | [43.44, 43.90] | 178.68 | [178.14, 179.22] | 0 |
| 7 | refine | 130125 | 25 | 12377.27 | [12244.09, 12510.46] | 12377.27 | [12244.09, 12510.46] | 104104.84 | -0.06 | false | 43.68 | [43.49, 43.87] | 178.86 | [178.47, 179.26] | 0 |

