# AGNTCY Slim deploy action

A GitHub action to craate a kind cluster and deploy the agntcy/slim into it.

## Inputs:

- `slim-image-tag`: Agntcy Slim slim image tag (default: `0.3.14`)
- `slim-chart-tag`: Agntcy Slim chart version (default: `v0.1.4`)
- `mcp-proxy-image-tag`: Agntcy Slim MCP proxy image tag (default: `0.1.4`)
- `mcp-proxy-deploy`: Agntcy Slim MCP proxy deploy (default: `false`)
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

      - name: Deploy AGNTCY Slim components
        uses: agntcy/csit/.github/actions/deploy-slim
        with:
          slim-image-tag: '0.3.14'
          slim-chart-tag: 'v0.1.4'
          mcp-proxy-enabled: 'false'
          kind-cluster-name: 'agntcy-test'
          kind-cluster-namespace: 'default'
```