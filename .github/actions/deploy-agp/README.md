# AGNTCY AGP deploy action

A GitHub action to craate a kind cluster and deploy the agntcy/agp into it.

## Inputs:

- `gateway-image-tag`: Agntcy AGP gateway image tag (default: `0.3.14`)
- `gateway-chart-tag`: Agntcy AGP chart version (default: `v0.1.4`)
- `mcp-proxy-image-tag`: Agntcy AGP MCP proxy image tag (default: `0.1.4`)
- `mcp-proxy-deploy`: Agntcy AGP MCP proxy deploy (default: `false`)
- `mcp-server-addr`: Set MCP server address (default: `http://mcp-server:8000/sse`)
- `kind-cluster-name`: KinD cluster name
- `kind-cluster-namespace`: Deployment namespace

## Example Workflow

```yaml
name: Deploy AGNTCY Components
on:
  push:
    branches:
      - main
  pull_request:
  workflow_dispatch:

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Deploy AGNTCY AGP components
        uses: agntcy/csit/.github/actions/deploy-agp
        with:
          gateway-image-tag: '0.3.14'
          gateway-chart-tag: 'v0.1.4'
          mcp-proxy-enabled: 'false'
          kind-cluster-name: 'agntcy-test'
          kind-cluster-namespace: 'default'
```