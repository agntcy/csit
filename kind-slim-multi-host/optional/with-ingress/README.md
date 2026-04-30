# Optional: local DNS helper + edge nginx + Ingress + Helm (slim)

Ingress-based stack: **cloud-provider-kind** (LoadBalancer on cluster **A**), **CoreDNS on cluster B** for `control.cluster-a.csit.test`, **edge nginx**, **CoreDNS DNS helper** for host `*.csit.test`, Helm values. Used by **`task optional:with-ingress:*`** / **`task up:with-ingress-apps`** (not plain **`task up`**).

- `edge/` — nginx front proxy (Docker Desktop–friendly port split)
- `dns/` — CoreDNS helper: host **`127.0.0.1:<port>`** → container `:53` (default **8053** via **`CSIT_LOCAL_DNS_HOST_PORT`**; same number in `/etc/resolver/csit.test` **`port`**)
- gRPC Ingress for slim dataplane and control plane — configured in [`helm/values/`](../helm/values/) via chart values (`slim.ingresses`, `ingressNorth`), not standalone manifests

Example (from `kind-slim-multi-host/`):

```bash
task up:with-ingress-apps
```

Or, if clusters are already up: `task optional:with-ingress:full`.

**KinD port maps:** [`kind/cluster-*.yaml`](../kind/cluster-a.yaml) lists `extraPortMappings` (e.g. host **10080** / **9080**) so edge nginx can reach each cluster’s ingress-nginx.
