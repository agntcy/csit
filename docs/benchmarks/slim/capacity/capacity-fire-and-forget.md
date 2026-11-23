# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-11-23 12:48:44

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
