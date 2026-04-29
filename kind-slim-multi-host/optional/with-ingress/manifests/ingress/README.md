# Ingress

gRPC Ingress for slim dataplane and control plane is defined in Helm values (upstream charts), not here:

- [`helm/values/controller.yaml`](../../../helm/values/controller.yaml) — `ingressNorth` (northbound gRPC, `control.cluster-a.csit.test`)
- [`helm/values/slim-cluster-a.yaml`](../../../helm/values/slim-cluster-a.yaml) — `slim.ingresses` (`slim.cluster-a.csit.test`)
- [`helm/values/slim-cluster-b.yaml`](../../../helm/values/slim-cluster-b.yaml) — `slim.ingresses` (`slim.cluster-b.csit.test`)

See slim charts: `charts/slim` (`templates/ingress.yaml`, values under `slim.ingresses`) and `charts/slim-control-plane` (`templates/ingress-north.yaml`, values under `ingressNorth`).
