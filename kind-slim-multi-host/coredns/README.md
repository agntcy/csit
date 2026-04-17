# CoreDNS peer stub zone (`csit.peer`)

Each Kind cluster’s **in-cluster CoreDNS** only resolves that cluster’s `*.svc.cluster.local`. Cross-cluster Kubernetes Service DNS is **not** provided here.

This bundle adds a **dedicated stub zone** `csit.peer` so pods can resolve stable names for the **other** cluster’s Kind node (Docker network IP):

- On cluster **A**: `node-b.csit.peer` → IPv4 of `${CLUSTER_B}-control-plane`
- On cluster **B**: `node-a.csit.peer` → IPv4 of `${CLUSTER_A}-control-plane`

Implementation:

1. `scripts/discover-peer-ips.sh` writes `coredns/.gen/peers.env` (gitignored).
2. `scripts/coredns-apply-peer.sh` merges a marked block (`# BEGIN csit-peer` … `# END csit-peer`) into `kube-system/coredns` ConfigMap `Corefile`, then restarts `deployment/coredns` when present.

Pods that call the peer must still use a **port reachable across Docker** (for example **NodePort**, **hostPort**, or a listener on that node IP)—same as any simple multi-network test rig.
