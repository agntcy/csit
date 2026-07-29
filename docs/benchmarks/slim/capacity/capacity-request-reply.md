# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-29 09:34:21

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
