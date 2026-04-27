# Optional: local DNS helper + edge nginx + Ingress + Helm (slim)

Ingress-based stack: **edge nginx**, **CoreDNS DNS helper** for `*.csit.test`, Ingress YAML, Helm values. Used by **`task optional:with-ingress:*`** / **`task up:with-ingress-apps`** (not plain **`task up`**).

- `edge/` — nginx front proxy (Docker Desktop–friendly port split)
- `dns/` — CoreDNS helper: host **`127.0.0.1:<port>`** → container `:53` (default **8053** via **`CSIT_LOCAL_DNS_HOST_PORT`**; same number in `/etc/resolver/csit.test` **`port`**)
- `manifests/ingress/` — gRPC Ingress for slim / control plane

Example (from `kind-slim-multi-host/`):

```bash
task up:with-ingress-apps
```

Or, if clusters are already up: `task optional:with-ingress:full`.

**KinD port maps:** the parent [`kind/cluster-*.yaml`](../kind/cluster-a.yaml) files already include `extraPortMappings` for edge → ingress. The `kind/*.portmap.example.yaml` files here are copies for reference only.
