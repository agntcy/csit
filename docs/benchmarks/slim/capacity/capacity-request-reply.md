# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-11-23 13:01:41

**Modes:** request-reply
**Clients:** 1
**Sizes:** 16384
**Start Rate:** 1000
**Max Rate:** 8000
**Growth Factor:** 2.00
**Plateau Threshold:** 5.00%
**Plateau Steps:** 1
**Max Steps:** 6
**Repeats Per Sweep Step:** 25

## Adaptive Capacity Sweep

This sweep first increases the configured send rate geometrically to find the saturation region, then performs midpoint refinement to narrow the offered-rate interval that saturates the node.
Results are reported separately for each fixed `(mode, clients, payload)` case. The reported rate is the aggregate offered load across all clients in that case. For request-reply and fire-and-forget, effective throughput is sink-observed total node throughput. For write mode, effective throughput is sender-completed write throughput because no responder is running.

- Modes: `request-reply`
- Clients: `1`
- Sizes: `16384` bytes
- Start rate: `1000` msg/sec
- Max rate: `8000` msg/sec (0 means unbounded by rate cap)
- Growth factor: `2.00`
- Plateau threshold: `5.00%` effective throughput gain
- Plateau steps: `1`
- Max steps: `6`
- Repeats per sweep step: `25`
- Refinement steps after coarse sweep: `4`
- Minimum offered-rate interval after refinement: `250` msg/sec

### Sink-Backed Modes

#### Request-Reply Clients=1 Payload=16384B

Best offered aggregate rate: `1000` msg/sec
Estimated capacity offered-rate interval: `[1000, 1250]` msg/sec
Best observed node throughput: `12.37` msg/sec with 95% CI [12.37, 12.37]
Best sender-completed throughput: `12.15` msg/sec with 95% CI [12.14, 12.16]
Best node CPU: `0.61` % with 95% CI [0.56, 0.66]
Best total CPU: `4.03` % with 95% CI [3.94, 4.12]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.15 | [12.14, 12.16] | 12.37 | [12.37, 12.37] | 0.00 | 0.00 | true | 0.61 | [0.56, 0.66] | 4.03 | [3.94, 4.12] | 0 |
| 2 | coarse | 2000 | 25 | 12.15 | [12.14, 12.16] | 12.37 | [12.36, 12.37] | 0.00 | -0.02 | false | 0.65 | [0.59, 0.70] | 4.14 | [4.04, 4.24] | 0 |
| 3 | refine | 1500 | 25 | 12.14 | [12.14, 12.15] | 12.36 | [12.36, 12.37] | 0.00 | -0.06 | false | 0.68 | [0.62, 0.74] | 4.26 | [4.16, 4.36] | 0 |
| 4 | refine | 1250 | 25 | 12.13 | [12.13, 12.14] | 12.35 | [12.34, 12.35] | 0.00 | -0.17 | false | 0.68 | [0.61, 0.75] | 4.29 | [4.15, 4.43] | 0 |
