# CoreDNS cross-cluster controller hostname (`csit.test`)

Cluster **B** pods resolve **`control.cluster-a.csit.test`** and **`slim.cluster-a.csit.test`** to cluster **A**’s **ingress-nginx LoadBalancer IP**, assigned by [cloud-provider-kind](https://github.com/kubernetes-sigs/cloud-provider-kind) on the host (HTTPS / gRPC on **443** to that IP).

Implementation:

1. [`scripts/discover-ingress-lb-ip-a.sh`](../scripts/discover-ingress-lb-ip-a.sh) waits for `ingress-nginx-controller` **LoadBalancer** on cluster A and writes **`.gen/ingress-a.env`** at the repo root of `kind-slim-multi-host/` (`INGRESS_A_LB_IP=…`, gitignored).
2. [`scripts/coredns-apply-cluster-b-ingress-alias.sh`](../scripts/coredns-apply-cluster-b-ingress-alias.sh) merges a **`# BEGIN csit-cross-cluster`** … **`# END csit-cross-cluster`** block into **cluster B** only (`kube-system/coredns` `Corefile`), defining zone **`csit.test`** with **`hosts`** entries for **`control.cluster-a`**, **`slim.cluster-a`**, **`spire.cluster-a`**, and **`spire-bundle.cluster-a`** (under the chosen **`CSIT_DNS_ZONE`**), then restarts CoreDNS.

In-cluster **`*.svc.cluster.local`** for the remote cluster is **not** mirrored—only these **`*.csit.test`** names for cluster **A** ingress (controller, slim dataplane gRPC, and SPIRE federation hostnames).
