# C1 Evidence Report

| Field | Value |
|-------|-------|
| Schema version | `1` |
| Generated at | 2026-07-14T13:45:55Z |
| Source | `analitics/tests/c1_evidence_test.go` |
| Machine-readable | [c1-evidence.json](./c1-evidence.json) |
| HTML | [c1-evidence.html](./c1-evidence.html) |

## Use cases

### c1-request-reply — verified

- **Mode:** `request-reply`
- **Use case:** Agent A calls B and waits for a reply
- **Sender messages:** 20
- **Sender errors:** 0
- **Sink received:** 20
- **Sink replies:** 20
- **Mean latency (ms):** 1.0696539999999999

**Assertions:**
- sender completed 20 messages with 0 runtime errors
- round-trip mean latency 1.070 ms
- sink received 20 messages and replied 20 times with 0 errors

### c1-fire-and-forget — verified

- **Mode:** `fire-and-forget`
- **Use case:** Agent fires an event; consumer handles async
- **Sender messages:** 20
- **Sender errors:** 0
- **Sink received:** 20
- **Mean latency (ms):** 0.110386

**Assertions:**
- sender completed 20 messages with 0 runtime errors
- sink received 20 async messages with 0 errors

### c1-write — verified

- **Mode:** `write`
- **Use case:** Publish into the mesh without a paired responder
- **Sender messages:** 20
- **Sender errors:** 0
- **Mean latency (ms):** 0.11468099999999999

**Assertions:**
- sender completed 20 messages with 0 runtime errors
- sender wrote at 9.78 msg/sec without a bound responder

