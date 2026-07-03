# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-03 12:33:43

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
Best observed node throughput: `12.38` msg/sec with 95% CI [12.38, 12.38]
Best sender-completed throughput: `12.17` msg/sec with 95% CI [12.17, 12.17]
Best node CPU: `0.59` % with 95% CI [0.55, 0.62]
Best total CPU: `4.15` % with 95% CI [4.10, 4.20]
Stop reason: refinement narrowed the estimated capacity to offered rates 1000 through 1250

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 1000 | 25 | 12.17 | [12.17, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | 0.00 | true | 0.59 | [0.55, 0.62] | 4.15 | [4.10, 4.20] | 0 |
| 2 | coarse | 2000 | 25 | 12.17 | [12.17, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | 0.01 | false | 0.61 | [0.56, 0.66] | 4.20 | [4.14, 4.25] | 0 |
| 3 | refine | 1500 | 25 | 12.17 | [12.16, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | 0.01 | false | 0.62 | [0.57, 0.66] | 4.20 | [4.14, 4.27] | 0 |
| 4 | refine | 1250 | 25 | 12.16 | [12.15, 12.17] | 12.38 | [12.38, 12.38] | 0.00 | -0.01 | false | 0.63 | [0.58, 0.67] | 4.24 | [4.19, 4.30] | 0 |
