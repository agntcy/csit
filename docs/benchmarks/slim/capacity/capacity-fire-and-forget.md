# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-22 09:18:10

**Modes:** fire-and-forget
**Clients:** 1
**Sizes:** 16384
**Start Rate:** 128000
**Max Rate:** 176000
**Growth Factor:** 1.12
**Plateau Threshold:** 3.00%
**Plateau Steps:** 2
**Max Steps:** 6
**Repeats Per Sweep Step:** 25

## Adaptive Capacity Sweep

This sweep first increases the configured send rate geometrically to find the saturation region, then performs midpoint refinement to narrow the offered-rate interval that saturates the node.
Results are reported separately for each fixed `(mode, clients, payload)` case. The reported rate is the aggregate offered load across all clients in that case. For request-reply and fire-and-forget, effective throughput is sink-observed total node throughput. For write mode, effective throughput is sender-completed write throughput because no responder is running.

- Modes: `fire-and-forget`
- Clients: `1`
- Sizes: `16384` bytes
- Start rate: `128000` msg/sec
- Max rate: `176000` msg/sec (0 means unbounded by rate cap)
- Growth factor: `1.12`
- Plateau threshold: `3.00%` effective throughput gain
- Plateau steps: `2`
- Max steps: `6`
- Repeats per sweep step: `25`
- Refinement steps after coarse sweep: `4`
- Minimum offered-rate interval after refinement: `1000` msg/sec

### Sink-Backed Modes

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
