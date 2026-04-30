#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Wait for ingress-nginx LoadBalancer EXTERNAL-IP on cluster A and write .gen/ingress-a.env.
# Usage: discover-ingress-lb-ip-a.sh <kubectl-context>
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CTX="${1:?kubectl context for cluster A}"
NS="${INGRESS_NS:-ingress-nginx}"
SVC="${INGRESS_SVC:-ingress-nginx-controller}"
OUT_DIR="${ROOT}/.gen"
OUT="${OUT_DIR}/ingress-a.env"

mkdir -p "$OUT_DIR"

echo "Waiting for LoadBalancer IP on ${CTX} Service ${NS}/${SVC}..."
for _ in $(seq 1 180); do
  IP=$(kubectl --context "$CTX" get svc "$SVC" -n "$NS" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)
  if [[ -n "$IP" ]]; then
    printf 'INGRESS_A_LB_IP=%s\n' "$IP" >"$OUT"
    echo "Wrote ${OUT}: INGRESS_A_LB_IP=${IP}"
    exit 0
  fi
  sleep 2
done

echo "ERROR: timed out waiting for ${SVC} LoadBalancer IP (is cloud-provider-kind running? See .gen/cloud-provider-kind.stderr)" >&2
exit 1
