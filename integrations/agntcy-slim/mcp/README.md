# Slim MCP KinD Integration Test

End-to-end test for [slim-mcp-rust](https://github.com/agntcy/slim-mcp-rust) MCP proxy and
[kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server) on a simple SLIM
data plane in KinD. No LLM is involved.

## Source of truth

`deps:fetch-repo` shallow-clones [slim-mcp-rust](https://github.com/agntcy/slim-mcp-rust) at a
pinned ref into `.cache/slim-mcp-rust`. Chart, Helm values, KinD config, and test client all come
from that checkout:

| What | Path in cloned repo |
|------|---------------------|
| Helm chart | `charts/mcp-proxy/` |
| Helm values (mcp-proxy) | `mcp-proxy/examples/k8s-test/mcp-proxy-config/mcp-proxy-values.yaml` |
| Helm values (k8s MCP server) | `mcp-proxy/examples/k8s-test/mcp-server-config/kubernetes-mcp-server-values.yaml` |
| KinD config | `mcp-proxy/examples/k8s-test/kind-config/kind-cluster-config.yaml` |
| Test client | `mcp-proxy/examples/k8s-test/mcp-client/` |

## Quick start

```bash
task integrations:slim-mcp:test
```

## Overrides

| Variable | Default |
|----------|---------|
| `SLIM_MCP_RUST_REF` | `helm-mcp-proxy-v0.1.4` |
| `MCP_PROXY_IMAGE_TAG` | `0.2.5` |
| `SLIM_CHART_TAG` | `v1.4.0` |
| `SLIM_IMAGE_TAG` | `1.4.0` |
