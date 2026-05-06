#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Patch kube-system/coredns on cluster B so control.cluster-a.<zone>, slim.cluster-a.<zone>,
# spire.cluster-a.<zone>, and spire-bundle.cluster-a.<zone> resolve to cluster A ingress-nginx
# LoadBalancer IP (HTTPS / gRPC on :443).
# Usage: coredns-apply-cluster-b-ingress-alias.sh <kubectl-context-cluster-b>
# Requires: .gen/ingress-a.env (from discover-ingress-lb-ip-a.sh), jq
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT}/.gen/ingress-a.env"
if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: missing $ENV_FILE — run discover-ingress-lb-ip-a.sh first" >&2
  exit 1
fi
# shellcheck source=/dev/null
source "$ENV_FILE"

CTX="${1:?kubectl context for cluster B}"
ZONE="${CSIT_DNS_ZONE:-csit.test}"
CONTROL_HOST="control.cluster-a.${ZONE}"
SLIM_HOST="slim.cluster-a.${ZONE}"
SPIRE_HOST="spire.cluster-a.${ZONE}"
SPIRE_BUNDLE_HOST="spire-bundle.cluster-a.${ZONE}"
IP="${INGRESS_A_LB_IP:?missing INGRESS_A_LB_IP in ingress-a.env}"

command -v jq >/dev/null 2>&1 || { echo "ERROR: jq is required" >&2; exit 1; }

BLOCK="# BEGIN csit-cross-cluster
${ZONE}:53 {
    errors
    cache 30
    hosts {
        ${IP} ${CONTROL_HOST}
        ${IP} ${SLIM_HOST}
        ${IP} ${SPIRE_HOST}
        ${IP} ${SPIRE_BUNDLE_HOST}
        fallthrough
    }
    forward . /etc/resolv.conf {
        max_concurrent 1000
    }
}
# END csit-cross-cluster"

CURRENT=$(kubectl --context "$CTX" get configmap coredns -n kube-system -o jsonpath='{.data.Corefile}')
STRIPPED=$(printf '%s' "$CURRENT" | sed '/^# BEGIN csit-cross-cluster$/,/^# END csit-cross-cluster$/d')
MERGED="${STRIPPED}
${BLOCK}
"

TMP=$(mktemp)
printf '%s' "$MERGED" >"$TMP"
kubectl --context "$CTX" get configmap coredns -n kube-system -o json | jq --rawfile cf "$TMP" '.data.Corefile = $cf' | kubectl --context "$CTX" apply -f -
rm -f "$TMP"

if kubectl --context "$CTX" get deployment coredns -n kube-system >/dev/null 2>&1; then
  kubectl --context "$CTX" rollout restart deployment/coredns -n kube-system
  kubectl --context "$CTX" rollout status deployment/coredns -n kube-system --timeout=120s
else
  echo "WARN: no deployment/coredns; restart CoreDNS pods manually if needed" >&2
fi

echo "CoreDNS updated on $CTX: ${CONTROL_HOST} ${SLIM_HOST} ${SPIRE_HOST} ${SPIRE_BUNDLE_HOST} -> ${IP}"
