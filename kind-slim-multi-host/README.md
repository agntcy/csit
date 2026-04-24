# Two KinD clusters + CoreDNS peer names (slim)

Default path: **two minimal Kind clusters**, discover each control-plane **IPv4 on the Docker `kind` network**, and patch **CoreDNS** in each cluster with a small **`csit.peer`** stub zone so workloads can resolve the other cluster’s node as **`node-a.csit.peer`** / **`node-b.csit.peer`**.

**What this does not do:** in-cluster DNS still cannot resolve the **other** cluster’s `*.svc.cluster.local` without multicluster plumbing. Use **`csit.peer`** names for the **node IP**, then expose services with **NodePort**, **hostPort**, or similar.

An **optional** subtree [`optional/with-ingress/`](optional/with-ingress/) keeps the older **Ingress + host edge nginx + dnsmasq** workflow for laptop-friendly hostnames; it is **not** run by default.

No changes are required elsewhere in the csit repository; run all tasks from this directory.

## GitHub Actions

Workflow: [`../.github/workflows/kind-slim-multicluster.yml`](../.github/workflows/kind-slim-multicluster.yml)

| Trigger | Behavior |
|---------|----------|
| `push` to `main` | Installs tooling via [`.github/actions/setup-k8s`](../.github/actions/setup-k8s), then `task prereq` → `task up` → checks → `task teardown`. Path filters: `kind-slim-multi-host/**`, this workflow, and `setup-k8s` action. |
| `pull_request` | Same as push, with the same path filters. |
| `workflow_dispatch` | Same baseline, plus optional boolean inputs below. |

**`workflow_dispatch` inputs**

| Input | Effect |
|-------|--------|
| `with_ingress_install` | After `task up`, run `task optional:with-ingress:ingress:install:all`. |
| `with_apps` | After `task up`, install ingress-nginx (unless `with_ingress_full`), then `task apps:install` (Helm + Ingress manifests). Helm is already on the runner from `setup-k8s`. |
| `with_ingress_full` | Runs `task optional:with-ingress:full` (ingress, edge, dnsmasq, apps). Often unsuitable on shared runners if host ports **80** or **53** are unavailable. |

PR and push runs **do** install Helm (and kind/kubectl/ct) via `setup-k8s`, but they **do not** run optional ingress or `apps:install` unless you use `workflow_dispatch` with the inputs above.

### Test the workflow locally

**Option A (recommended):** run the same steps CI uses. From the repo root, with Docker running and `kind`, `kubectl`, `jq`, and `task` installed (any recent versions):

```bash
cd kind-slim-multi-host
task prereq
task up
kubectl --context kind-csit-a get nodes
kubectl --context kind-csit-b get nodes
kubectl --context kind-csit-a get configmap coredns -n kube-system -o jsonpath='{.data.Corefile}' | grep -q "BEGIN csit-peer"
kubectl --context kind-csit-a run dns-verify -n default --rm --attach --restart=Never \
  --image=docker.io/library/busybox:1.36 --command -- nslookup node-b.csit.peer
task teardown
```

To approximate **`workflow_dispatch`** with optional flags, run the same extra `task` targets by hand after `task up` (for example `task optional:with-ingress:ingress:install:all`, `task apps:install`, or `task optional:with-ingress:full`) before the verify lines and `task teardown`.

**Option B ([act](https://github.com/nektos/act)):** run Actions in Docker from the **repository root** (the directory that contains `kind-slim-multi-host/` and `.github/`). KinD needs nested Docker and often a **privileged** runner image; this is brittle on some hosts (especially Apple Silicon). Example dry run:

```bash
act -n -W .github/workflows/kind-slim-multicluster.yml workflow_dispatch
```

Pass `workflow_dispatch` inputs with a JSON file per [act event payloads](https://nektosact.com/usage/index.html#passing-inputs-to-manual-workflows). Full job parity is usually easier with **Option A** on your laptop.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/)
- [kubectl](https://kubernetes.io/docs/tasks/)
- [jq](https://jqlang.org/)
- [Task](https://taskfile.dev/) (recommended)

Helm is only needed for optional app installs (`task apps:install`).

## Quick start

From `kind-slim-multi-host/`:

```bash
task prereq
task up
```

This creates clusters **csit-a** and **csit-b** (contexts `kind-csit-a`, `kind-csit-b`), writes `coredns/.gen/peers.env`, patches CoreDNS on both clusters, and restarts the CoreDNS Deployment when it exists.

### Verify

From cluster A:

```bash
kubectl --context kind-csit-a run -n default --rm -it --restart=Never dns-test --image=busybox:1.36 -- nslookup node-b.csit.peer
```

You should see the Docker IP of `csit-b-control-plane`.

### Tear down

```bash
task teardown
```

Deletes both Kind clusters. Generated files under `coredns/.gen/` can be removed manually; they are gitignored.

## Task reference

| Task | Purpose |
|------|---------|
| `task up` | `prereq` → create both clusters → `coredns:discover` → `coredns:apply:all` |
| `task coredns:discover` | Refresh `coredns/.gen/peers.env` from Docker |
| `task coredns:apply:a` / `coredns:apply:b` | Patch CoreDNS on one context |
| `task coredns:apply:all` | Both clusters |
| `task teardown` | `kind:delete:all` |

Environment overrides: `CLUSTER_A`, `CLUSTER_B`, `KIND_DOCKER_NETWORK` (default `kind`) — see `scripts/discover-peer-ips.sh`.

## Helm values (optional)

After `task up`, you can install charts with `task apps:install` (requires `task prereq:apps` and the optional Ingress path if you rely on Ingress hostnames).

| File | Role |
|------|------|
| [`helm/values/controller.yaml`](helm/values/controller.yaml) | slim-control-plane on A |
| [`helm/values/slim-cluster-a.yaml`](helm/values/slim-cluster-a.yaml) | Slim on A |
| [`helm/values/slim-cluster-b.yaml`](helm/values/slim-cluster-b.yaml) | Slim on B; example controller client → `node-a.csit.peer:50051` (adjust port after you expose northbound on node A) |

## Optional: Ingress, edge, dnsmasq

See [`optional/with-ingress/README.md`](optional/with-ingress/README.md). Example KinD configs with **host port mappings** for that stack live under `optional/with-ingress/kind/*.portmap.example.yaml`; copy or merge into `kind/cluster-*.yaml` before recreating clusters if you use the edge proxy.

Composite task:

```bash
task optional:with-ingress:full
```

(Order: ingress-nginx on both clusters → edge → dnsmasq → Helm + Ingress manifests.)

## Limitations

- **Stub zone only:** `csit.peer` maps to **Kind node container IPs**, not remote Service DNS.
- **Reachability:** pod → peer node IP must use a port that routing allows (often NodePort or a published port on that node).
- **Kind/CoreDNS versions:** the apply script expects a `deployment/coredns` in `kube-system`; if your Kind version differs, adjust the rollout target to match your cluster.
