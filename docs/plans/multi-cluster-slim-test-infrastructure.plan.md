# Multi-cluster slim test infrastructure (plan + current implementation)

## Implemented: `kind-slim-multi-host/`

This is the **authoritative** local multicluster testbed in this repository.

| Piece | Role |
|--------|------|
| **KinD** | Two clusters **csit-a** / **csit-b** (`kind/cluster-*.yaml`): ingress on host **127.0.0.1** at **10080**/10443 (A) and **9080**/9443 (B); cluster **A** **ingress-nginx** uses **LoadBalancer** (cloud-provider-kind); CoreDNS on **B** maps **`control.cluster-a.csit.test`** → LB IP (`scripts/coredns-apply-cluster-b-ingress-alias.sh`). |
| **ingress-nginx** | Installed on **cluster A** only (LoadBalancer); receives traffic from the host edge proxy. |
| **Edge nginx** | Host **:80** routes by `Host` to the correct KinD ingress port (`kind-slim-multi-host/compose/edge/`). |
| **Local DNS helper** | Docker **CoreDNS** + generated **`Corefile`** (`scripts/render-local-dns-corefile.sh`), published at **`127.0.0.1:${CSIT_LOCAL_DNS_HOST_PORT:-8053}`** → `*.csit.test` → **127.0.0.1** (macOS: `/etc/resolver/csit.test` + matching `port`). |
| **Helm + Ingress YAML** | **`task stack:install`** installs ingress (A), slim + controller, LB wait, CoreDNS patch on B, edge + local DNS helper. |
| **CI** | [`../../.github/workflows/kind-slim-multicluster.yml`](../../.github/workflows/kind-slim-multicluster.yml) runs **`task cluster:up`** → **`task stack:install`** / **`task stack:down`** and verifies clusters, LoadBalancer, CoreDNS alias on B, host DNS, and Helm releases. |

**Tasks:** **`task cluster:up`** = KinD only. **`task stack:up`** = full stack. See [`kind-slim-multi-host/README.md`](../../kind-slim-multi-host/README.md).

**Generated (not committed):** `kind-slim-multi-host/.gen/*`, `kind-slim-multi-host/compose/dns/Corefile` (see `kind-slim-multi-host/.gitignore`).

---

## Longer-term / alternate patterns (not required for `kind-slim-multi-host`)

These remain useful for **integrations/** or custom topologies; they differ from the default **single loopback IP + edge routing** approach above.

### Optional: two loopback IPs (A vs B on 127.0.0.1 vs 127.0.0.2)

If each KinD mapping binds ingress to a **different** host address (e.g. **127.0.0.2** for B), DNS can return **127.0.0.1** vs **127.0.0.2** per hostname instead of relying on one edge proxy. That needs extra host setup (loopback aliases; Docker Desktop nuances). The repo **does not** ship example KinD YAML for that path anymore; document here if you reintroduce it.

### integrations/ topology automation (future)

- [`integrations/agntcy-slim/config/peer-to-peer.yaml`](../../integrations/agntcy-slim/config/peer-to-peer.yaml) — multiple clusters in topology YAML.
- [`integrations/agntcy-slim/tests/config/main/generate_configs.go`](../../integrations/agntcy-slim/tests/config/main/generate_configs.go) — today may still use a **single** `SLIM_CONTROLLER_ENDPOINT`; multicluster needs **per-cluster** controller client endpoints (ingress URL on B, in-cluster on A).
- Topology tests may still assume **one** kube client — multi-cluster runs need two contexts or merged kubeconfig.

| Gap | Direction |
|-----|-----------|
| Per-cluster controller endpoint | Extend generator / topology schema with per-cluster `controllerEndpoint` (or equivalent). |
| Tests | Parameterize kubecontext per cluster in topology tests. |

## Summary

| Layer | Default in `kind-slim-multi-host` |
|-------|----------------------------------|
| Clusters | 2× KinD, contexts `kind-csit-a` / `kind-csit-b` |
| Ingress collision | Host **127.0.0.1** + distinct **hostPort** maps + **edge nginx** by `Host` |
| DNS | **`control.cluster-a.csit.test`** → ingress LB IP on cluster **B** CoreDNS; **`*.csit.test`** on host via CoreDNS helper + split DNS |
| Slim → controller from B | **`http://control.cluster-a.csit.test`** through ingress **LoadBalancer**; Docker Desktop fallback: [`slim-cluster-b.docker-desktop.yaml`](../../kind-slim-multi-host/helm/values/slim-cluster-b.docker-desktop.yaml) |
