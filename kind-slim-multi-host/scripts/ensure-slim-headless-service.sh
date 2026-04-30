#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# The upstream slim chart StatefulSet uses serviceName: <release>-slim-headless, but the chart
# only emits a Service named <release>-slim (see charts/slim/templates/{service,statefulset}.yaml).
# Without this headless Service, the StatefulSet never becomes ready and helm --wait hangs.
#
# Usage: ensure-slim-headless-service.sh <kubectl-context> <namespace> <helm-release-name>
# Ports/names match slim chart defaults (slim.service.data/control).
set -euo pipefail

CTX="${1:?context}"
NS="${2:?namespace}"
REL="${3:?helm release name}"

NAME="${REL}-slim-headless"

kubectl --context "$CTX" create namespace "$NS" --dry-run=client -o yaml | kubectl --context "$CTX" apply -f -

kubectl --context "$CTX" apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: ${NAME}
  namespace: ${NS}
  labels:
    app.kubernetes.io/name: slim
    app.kubernetes.io/instance: ${REL}
spec:
  clusterIP: None
  publishNotReadyAddresses: true
  selector:
    app.kubernetes.io/name: slim
    app.kubernetes.io/instance: ${REL}
  ports:
    - port: 46357
      targetPort: data-plane-0
      protocol: TCP
      name: data-plane-0
    - port: 46358
      targetPort: controller-0
      protocol: TCP
      name: controller-0
EOF

echo "Ensured headless Service ${NS}/${NAME} for StatefulSet serviceName"
