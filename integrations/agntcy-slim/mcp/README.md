# Slim MCP Integration Test

Slim MCP Integration Test is an end-to-end test for validating [slim-mcp-rust](https://github.com/agntcy/slim-mcp-rust) MCP proxy and [kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server) on a simple SLIM data plane in KinD. It deploys the stack with Helm, connects a Python MCP client over SLIM, and exercises Kubernetes MCP tools without an LLM.

## Overview

The Slim MCP integration test enables you to:

- Clone chart, Helm values, KinD config, and test client from a pinned [slim-mcp-rust](https://github.com/agntcy/slim-mcp-rust) ref
- Create a dedicated KinD cluster (`mcp-test`) and deploy `kubernetes-mcp-server`, `mcp-proxy`, and `SLIM`
- Port-forward the `SLIM` service and run the upstream Python test client locally
- List MCP tools and call `pods_list_in_namespace` against the in-cluster API server
- Capture client output under `reports/test-output.log` for CI and GitHub Pages

## Quick Start

### Prerequisites

- Docker
- [kind](https://kind.sigs.k8s.io/), `kubectl`, and `helm`
- [Task](https://taskfile.dev/)
- `git` and [uv](https://docs.astral.sh/uv/) (Python 3.13+)

### Steps to Run the MCP Test

1. **Navigate to the integration directory:**
   ```bash
   cd integrations/agntcy-slim/mcp
   ```

2. **Run the full test (create cluster, deploy stack, run client, tear down):**
   ```bash
   task test
   ```

   From the repository root you can also run:
   ```bash
   task integrations:slim-mcp:test
   ```

### Run Step by Step

1. **Create the KinD cluster and deploy the stack:**
   ```bash
   task up
   ```

2. **Run the MCP client test:**
   ```bash
   task test:run
   ```

3. **Inspect the report (optional):**
   ```bash
   cat reports/test-output.log
   ```

### Tear Down

1. **Uninstall Helm releases and delete the KinD cluster:**
   ```bash
   task down
   ```

## Configuration Source

Chart, Helm values, KinD config, and test client are not vendored in this repository. `deps:fetch-repo` shallow-clones slim-mcp-rust at a pinned ref into `.cache/slim-mcp-rust`:

| What | Path in cloned repo |
|------|---------------------|
| Helm chart | `charts/mcp-proxy/` |
| Helm values (mcp-proxy) | `mcp-proxy/examples/k8s-test/mcp-proxy-config/mcp-proxy-values.yaml` |
| Helm values (k8s MCP server) | `mcp-proxy/examples/k8s-test/mcp-server-config/kubernetes-mcp-server-values.yaml` |
| KinD config | `mcp-proxy/examples/k8s-test/kind-config/kind-cluster-config.yaml` |
| Test client | `mcp-proxy/examples/k8s-test/mcp-client/` |

The SLIM subchart version in the cloned chart is patched to match `SLIM_CHART_TAG` before `helm dependency update`.


## Test Execution

The test framework performs the following steps:

1. **Clone slim-mcp-rust** at the pinned ref and refresh Helm chart dependencies
2. **Create KinD cluster** `mcp-test` (or reuse it if already present) and fix kubeconfig for local API access
3. **Install Helm releases** `kubernetes-mcp-server` and `mcp-proxy` into namespace `mcp-system`
4. **Wait** for deployments and the SLIM StatefulSet to become ready
5. **Port-forward** `svc/slim` to `localhost:46357`
6. **Run the Python client** — list tools, call `pods_list_in_namespace` for `mcp-system`, write output to `reports/test-output.log`
7. **Clean up** Helm releases and the KinD cluster

## Overrides

| Variable | Default |
|----------|---------|
| `SLIM_MCP_RUST_REF` | `helm-mcp-proxy-v0.1.4` |
| `MCP_PROXY_IMAGE_TAG` | `0.2.5` |
| `SLIM_CHART_TAG` | `v1.4.0` |
| `SLIM_IMAGE_TAG` | `1.4.0` |

Example:

```bash
SLIM_MCP_RUST_REF=helm-mcp-proxy-v0.1.4 SLIM_IMAGE_TAG=1.4.0 task test
```

## CI

The `test-slim-mcp` workflow runs `task integrations:slim-mcp:test` on GitHub Actions, uploads `reports/` as an artifact, and publishes an HTML report to GitHub Pages on `main`.
