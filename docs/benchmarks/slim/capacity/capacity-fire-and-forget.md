# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-14 14:06:29

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
