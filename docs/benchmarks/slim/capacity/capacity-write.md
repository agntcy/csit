# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-22 09:51:34

**Modes:** write
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

- Modes: `write`
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

### Write Mode

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
