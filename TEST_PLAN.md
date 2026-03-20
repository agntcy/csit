# SLIM Integration & Benchmark Test Plan

## Objective
Develop a comprehensive set of integration and performance tests for the [SLIM](https://github.com/agntcy/slim) project within the `csit` repository. The goal is to produce proof points for SLIM's value proposition:
- **Low Latency** (Messaging)
- **Security** (MLS, mTLS, Identity)
- **Scalable Group Communication**

## Methodology
We will leverage **Behavior-Driven Development (BDD)** using the [Ginkgo](https://onsi.github.io/ginkgo/) framework and [Gomega](https://onsi.github.io/gomega/) matchers.

To enable **Black Box Testing**, we will develop and utilize external tools that interact with the System Under Test (SUT) from the outside, treating SLIM nodes as opaque boxes.

### 1. External Tools Strategy

We will build/integrate the following tools:

#### A. Performance: `slim-bench`
A custom High-Performance Load Generator ensuring we measure the protocol's limits, not the test runner's.
- **Location**: `benchmarks/agntcy-slim/tools/slim-bench`
- **Language**: Go (using SLIM Go bindings)
- **Features**:
  - Configurable message payload size.
  - Configurable message rate (MPS).
  - Mode: `ping-pong` (1-to-1) and `fan-out` (1-to-N).

#### B. Network Impairment: `toxiproxy`
To simulate real-world conditions (jitter, latency, packet loss).
- **Tool**: [Shopify/toxiproxy](https://github.com/Shopify/toxiproxy)
- **3. Se**: Run as a sidecar or gateway container. The test runner uses the API to inject faults.

#### C. Security: `security-probe`
A tool specifically designed to violate protocol specs and attempt unauthorized access.
- **Location**: `integrations/agntcy-slim/tools/security-probe`
- **Features**:
  - `bad-tls`: Attempts handshakes with invalid connections.
  - `mitm-tamper`: Acts as a TCP proxy that modifies payloads.

### 2. Performance Benchmarks (BDD)
Located in: `benchmarks/agntcy-slim/tests/`

**Key Scenarios:**
- **Point-to-Point Latency**: Measure round-trip time (RTT) between two SLIM nodes/clients.
  - *Given* two connected agents.
  - *When* Agent A sends a message to Agent B.
  - *Then* the latency should be within `X` milliseconds.
- **Group Broadcast Latency**: Measure fan-out time for group messages.
  - *Given* a group of `N` agents.
  - *When* the leader multicasts a message.
  - *Then* all `N` agents should receive it within `Y` milliseconds.
- **Throughput**: Measure messages per second (MPS) under load.

**Implementation Details:**
- Use `gmeasure` (Ginkgo's measurement package) to record and report statistics.
- Use the existing `Taskfile.yml` to run these benchmarks locally and in CI.

### 2. security & Adversarial Tests (BDD)
Located in: `integrations/agntcy-slim/tests/`

**Key Scenarios (Adversarial):**
- **Unauthorized Access**:
  - *Given* a secure SLIM group.
  - *When* an agent without valid credentials (bad TLS cert or missing key) attempts to connect/join.
  - *Then* the connection is rejected.
- **Traffic Isolation**:
  - *Given* two distinct SLIM namespaces/groups.
  - *When* Agent A in Group 1 tries to send to Group 2.
  - *Then* the message is dropped or rejected.
- **Man-in-the-Middle (MitM) Resistance**:
  - *Given* a strictly mTLS-enforced environment.
  - *When* a proxy tries to intercept traffic without a valid CA signature.
  - *Then* the handshake fails.

### 4. Adversarial Robustness & Security Performance
Located in: `integrations/agntcy-slim/tests/adversarial_performance_test.go`

**Objective**: Validate system stability and performance under active attack.

**Key Scenarios:**
- **Connection Flooding (DoS)**:
  - *Given* a SLIM node ensuring availability.
  - *When* `security-probe --mode=flood` opens 10,000 concurrent, idle TCP connections.
  - *Then* the node should still process legitimate traffic (though possibly with increased latency), and memory usage should be bounded.
- **Handshake Stalling (Resource Exhaustion)**:
  - *Given* a running node.
  - *When* `security-probe --mode=slow-handshake` initiates 500 TLS handshakes but never completes them.
  - *Then* the node should timeout these connections efficiently without crashing or blocking new valid handshakes.
- **CPU Exhaustion (Crypto-Dos)**:
  - *Given* a node with limited CPU.
  - *When* `security-probe --mode=bad-crypto` spams ClientHellos with expensive cipher suites or renegotiation requests.
  - *Then* the node maintains responsiveness for existing sessions.

## Next Steps
1.  **Scaffold Performance Tests**: Create `benchmarks/agntcy-slim/tests/performance_test.go`.
2.  **Scaffold Security Tests**: Create `integrations/agntcy-slim/tests/security_adversarial_test.go`.
3.  **Implement Setup Logic**: helper functions to spin up SLIM nodes (using `testutils/k8shelper` or Docker runners).
