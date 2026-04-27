# Optional: local DNS helper + edge nginx + Ingress + Helm (slim)

This subtree provides the **Ingress-based** workflow (edge proxy, local DNS helper, Kubernetes Ingress manifests, Helm values for slim).

It is **not** part of the default `task up` path. Use it when you want hostnames on port 80/443 and OCI chart installs.

- `edge/` — nginx front proxy (Docker Desktop–friendly port split)
- `dns/` — CoreDNS-based local DNS helper for `*.csit.test` (default **host `127.0.0.1:8053`** → container 53; set matching `port` in `/etc/resolver/csit.test`. Override with **`CSIT_DNSMASQ_HOST_PORT`**.)
- `manifests/ingress/` — gRPC Ingress for slim / control plane

Example (from `kind-slim-multi-host/`):

```bash
task up:with-ingress-apps
```

Or, if clusters are already up: `task optional:with-ingress:full`.

**KinD port maps:** the parent [`kind/cluster-*.yaml`](../kind/cluster-a.yaml) files already include `extraPortMappings` for edge → ingress. The `kind/*.portmap.example.yaml` files here are copies for reference only.
