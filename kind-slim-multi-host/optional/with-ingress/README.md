# Optional: host dnsmasq + edge nginx + Ingress + Helm (slim)

This subtree preserved the previous **Ingress-based** workflow (edge proxy, host dnsmasq, Kubernetes Ingress manifests, Helm values for slim).

It is **not** part of the default `task up` path. Use it when you want hostnames on port 80/443 and OCI chart installs.

- `edge/` — nginx front proxy (Docker Desktop–friendly port split)
- `dns/` — dnsmasq for `*.csit.test` on the laptop
- `manifests/ingress/` — gRPC Ingress for slim / control plane

Example (from `kind-slim-multi-host/` after `task up` or with clusters already created):

```bash
task optional:with-ingress:full
```

**KinD port maps:** the default `kind/cluster-*.yaml` in the parent directory is minimal (no host port mappings). For edge nginx to reach each cluster’s ingress, merge or replace with the examples in `optional/with-ingress/kind/cluster-a.portmap.example.yaml` and `cluster-b.portmap.example.yaml`, then recreate the clusters.
