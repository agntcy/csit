# AGNTCY AGP deploy action

A GitHub action to craate a kind cluster and deploy the agntcy/agp into it.

## Inputs:

- `gateway-image-tag`: Agntcy AGP gateway image tag (default: `0.3.11`)
- `gateway-chart-tag`: Agntcy AGP chart version (default: `v0.1.2`)
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

      - name: Deploy AGNTCY components
        uses: agntcy/csit/.github/actions/deploy-agp@feat/add-gw-deploy-action
        with:
          gateway-image-tag: '0.3.11'
          gateway-chart-tag: 'v0.1.2'
          kind-cluster-name: 'agntcy-test'
          kind-cluster-namespace: 'default'
```