# SLIM Adaptive Capacity Sweep Report

**Generated:** 2026-07-14 13:40:53

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
Best observed node throughput: `6431.80` msg/sec with 95% CI [6378.66, 6484.93]
Best sender-completed throughput: `6362.59` msg/sec with 95% CI [6306.52, 6418.66]
Best node CPU: `43.88` % with 95% CI [43.36, 44.41]
Best total CPU: `248.25` % with 95% CI [246.03, 250.47]
Stop reason: refinement narrowed the estimated capacity to offered rates 128000 through 130125

| Step | Phase | Offered Aggregate Rate | Repeats | Sender Mean msg/sec | Sender 95% CI | Observed Node Throughput | Observed Node Throughput 95% CI | Observed Variance | Observed Gain % | Improved | Node CPU % | Node CPU 95% CI | Total CPU % | Total CPU 95% CI | Errors |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | coarse | 128000 | 25 | 6362.59 | [6306.52, 6418.66] | 6431.80 | [6378.66, 6484.93] | 16570.57 | 0.00 | true | 43.88 | [43.36, 44.41] | 248.25 | [246.03, 250.47] | 0 |
| 2 | coarse | 144000 | 25 | 6404.47 | [6384.56, 6424.38] | 6482.46 | [6469.52, 6495.40] | 982.79 | 0.79 | false | 44.32 | [44.15, 44.49] | 249.94 | [249.31, 250.57] | 0 |
| 3 | coarse | 162000 | 25 | 6416.07 | [6402.20, 6429.94] | 6492.10 | [6478.15, 6506.04] | 1141.59 | 0.94 | false | 44.37 | [44.22, 44.52] | 250.20 | [249.53, 250.88] | 0 |
| 4 | refine | 145000 | 25 | 6422.48 | [6408.93, 6436.04] | 6498.77 | [6488.36, 6509.19] | 636.64 | 1.04 | false | 44.51 | [44.38, 44.64] | 250.83 | [250.40, 251.27] | 0 |
| 5 | refine | 136500 | 25 | 6414.00 | [6397.75, 6430.25] | 6492.26 | [6478.45, 6506.07] | 1119.42 | 0.94 | false | 44.49 | [44.38, 44.61] | 250.60 | [250.10, 251.10] | 0 |
| 6 | refine | 132250 | 25 | 6424.40 | [6410.32, 6438.49] | 6498.48 | [6487.13, 6509.83] | 755.81 | 1.04 | false | 44.60 | [44.48, 44.73] | 251.16 | [250.58, 251.73] | 0 |
| 7 | refine | 130125 | 25 | 6429.56 | [6417.61, 6441.51] | 6491.37 | [6479.89, 6502.86] | 773.53 | 0.93 | false | 44.63 | [44.53, 44.73] | 251.40 | [251.06, 251.74] | 0 |
