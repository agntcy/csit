# CoreDNS cross-cluster controller hostname (`csit.test`)

Cluster **B** pods resolve **`control.cluster-a.csit.test`** to cluster **A**’s **ingress-nginx LoadBalancer IP**, assigned by [cloud-provider-kind](https://github.com/kubernetes-sigs/cloud-provider-kind) on the host.

Implementation:

1. [`scripts/discover-ingress-lb-ip-a.sh`](../scripts/discover-ingress-lb-ip-a.sh) waits for `ingress-nginx-controller` **LoadBalancer** on cluster A and writes **`coredns/.gen/ingress-a.env`** (`INGRESS_A_LB_IP=…`, gitignored).
2. [`scripts/coredns-apply-cluster-b-ingress-alias.sh`](../scripts/coredns-apply-cluster-b-ingress-alias.sh) merges a **`# BEGIN csit-cross-cluster`** … **`# END csit-cross-cluster`** block into **cluster B** only (`kube-system/coredns` `Corefile`), defining zone **`csit.test`** with a **`hosts`** entry for **`control.cluster-a.csit.test`**, then restarts CoreDNS.

Legacy **`csit.peer`** stubs (**`node-a.csit.peer`**) are removed from this repo; upgrades can run **`task coredns:strip:legacy-peer`** once to drop old CoreDNS blocks.

In-cluster **`*.svc.cluster.local`** for the remote cluster is still **not** provided—only this explicit **`*.csit.test`** hostname for the controller ingress edge.
