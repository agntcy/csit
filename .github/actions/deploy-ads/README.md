# AGNTCY ADS deploy action

A GitHub action to craate a kind cluster and deploy the agntcy/slim into it.

## Inputs:

- `directory-image-tag`: Agntcy ADS image tag (default: `v0.2.5`)
- `directory-chart-tag`: Agntcy ADS chart version (default: `v0.2.5`)
- `kind-cluster-name`: KinD cluster name
- `kind-cluster-namespace`: Deployment namespace
- `deploy-dir-network`: Deploy directory network (default: `false`)
- `dirctl-bin-version`: Dirctl binary version (default: `v0.2.1`)
- `network-namespace-prefix`: Set cluster namespace where directory network are deployed (default: `network`)
- `network-size`: Number of directory network peers (default: `3`)

## Example Workflow

```yaml
name: Deploy AGNTCY ADS Components
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

      - name: Deploy AGNTCY ADS components
        uses: agntcy/csit/.github/actions/deploy-ads
        with:
          directory-image-tag: 'v0.2.5'
          directory-chart-tag: 'v0.2.5'
          kind-cluster-name: 'agntcy-test'
          kind-cluster-namespace: 'default'
```