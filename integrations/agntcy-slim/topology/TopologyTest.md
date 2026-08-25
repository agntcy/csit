# TopologyTest

TopologyTest is a testing framework for validating control-plane managed SLIM topologies. It deploys a SLIM control plane in API mode together with a set of SLIM data-plane clusters, then drives the inter-cluster topology at runtime with `slimctl` and validates message routing between point-to-point (p2p) clients.

## Overview

The TopologyTest framework enables you to:

- Define the SLIM data-plane clusters declaratively in YAML (`config/clusters.yaml`).
- Deploy a control plane in API mode (topology is managed at runtime, not baked into the chart values) with SQLite persistence, exposed via a LoadBalancer service so `slimctl` can reach the northbound API from the host.
- Manage the inter-cluster topology (segments and links) at runtime with `slimctl`.
- Deploy p2p clients (alice/bob/carol) and validate reachability through SLIM client log assertions.
- Validate recovery behavior after data-plane node restarts and control-plane restarts.

## Quick Start

### Steps to Run Topology Test

1. **Navigate to the integration directory:**
   ```bash
   cd integrations/agntcy-slim/topology
   ```

2. **Download slimctl (used by the test to drive the topology):**
   ```bash
   task deps:slimctl-download
   ```

3. **Start a LoadBalancer provider (KinD only):**
   The controller Service is `type: LoadBalancer`, which stays `<pending>` on a
   plain KinD cluster. Start [cloud-provider-kind](https://github.com/kubernetes-sigs/cloud-provider-kind)
   so the Service gets an external IP that `slimctl` can reach from the host:
   ```bash
   task kind:cloud-provider-kind:up
   ```
   > Requires Go. On macOS you may be prompted for your password (sudo is needed
   > for Docker access). Leave it running for the duration of the test; stop it
   > later with `task kind:cloud-provider-kind:down`. On a real cluster with a
   > cloud LoadBalancer, skip this step.

4. **Deploy the SLIM controller (API mode + persistence + LoadBalancer):**
   ```bash
   task test:topology:deploy:controller
   ```

5. **Deploy the generated cluster topology:**
   ```bash
   task test:topology:deploy:clusters
   ```

6. **Run the topology test:**
   ```bash
   task test:topology:run
   ```

7. **Render the HTML report (optional):**
   ```bash
   task test:topology:report
   ```

### Tear Down

1. **Clean up generated resources:**
   ```bash
   task test:topology:cleanup:clusters
   ```

2. **Clean up controller:**
   ```bash
   task test:topology:cleanup:controller
   ```

3. **Stop the LoadBalancer provider (KinD only):**
   ```bash
   task kind:cloud-provider-kind:down
   ```

## Cluster Configuration Format

The clusters are described in `config/clusters.yaml`:

```yaml
topology:
    clusters:
        "cluster-a":
            spireMtls: false
            replicaCount: 1
        "cluster-b":
            spireMtls: false
            replicaCount: 2
        "cluster-c":
            spireMtls: false
            replicaCount: 2
```

Each cluster entry supports:

- `spireMtls` (bool, default false): enable SPIRE mTLS for the data-plane nodes.
- `replicaCount` (int, default 1): number of SLIM data-plane node replicas in the cluster.
- `deployAsDaemonSet` (bool, default false): deploy nodes as a DaemonSet instead of a Deployment.

The default topology uses `cluster-a` (1 node), `cluster-b` (2 nodes) and `cluster-c` (2 nodes). Each cluster registers with the control plane as a group named after the cluster.

## Control plane (API mode)

The controller is deployed from `config/controller-values.yaml` in API mode: `topology: {}` leaves the control plane as the source of truth but with no pre-wired links, so the test creates segments and links at runtime with `slimctl`. SQLite persistence is enabled so topology survives control-plane restarts. The controller Service is of type `LoadBalancer` so the northbound API (`50051`) is reachable from the host running the test.

## Test Scenarios

The Ginkgo suite runs as an ordered set of `When`/`Then` scenarios:

1. **When the control plane and clusters are deployed / Then all clusters and nodes are joined:** every node reports `Connected` in `slimctl controller node list` and all three groups appear in `slimctl controller group list`.
2. **When cluster-a is linked to cluster-b and cluster-a to cluster-c in separate segments and the p2p clients are deployed / Then alice reaches bob and carol, but bob cannot reach carol:** alice sends to bob and to carol (distinct messages), and the b->c path is isolated, so carol never receives bob's message.
3. **When the topology is modified so cluster-b and cluster-c are linked / Then alice still reaches both and bob can now reach carol.**
4. **When the gateway node handling the b<->c link is stopped and restarted / Then the link is restored and bob can still reach carol** (a continuous sender keeps producing traffic across the restart).
5. **When the control plane is restarted / Then the links remain intact** (persistence via the SQLite PVC).

## Test Execution

The test framework performs the following steps:

1. **Parse `config/clusters.yaml`** and generate SLIM helm values/configs for each cluster.
2. **Deploy SLIM controller & SLIM cluster nodes** using the helm charts.
3. **Discover the controller northbound endpoint** (LoadBalancer) and drive topology with `slimctl`.
4. **Deploy p2p client pods** and watch their logs for assertion strings (presence and, for isolation, absence).
5. **Validate** reachability, isolation, and recovery behavior within timeout periods.
6. **Clean up** client resources after test completion.
