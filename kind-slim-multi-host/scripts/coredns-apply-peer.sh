#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Patch kube-system/coredns to add csit.peer stub zone pointing at the other Kind node.
# Usage: coredns-apply-peer.sh <kubectl-context> <a|b>
# Requires: coredns/.gen/peers.env (from discover-peer-ips.sh), jq
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT}/coredns/.gen/peers.env"
if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: missing $ENV_FILE — run task coredns:discover first" >&2
  exit 1
fi
# shellcheck source=/dev/null
source "$ENV_FILE"

CTX="${1:?context}"
ROLE="${2:?role a or b}"

command -v jq >/dev/null 2>&1 || { echo "ERROR: jq is required" >&2; exit 1; }

if [[ "$ROLE" == "a" ]]; then
  PEER_FQDN="node-b.csit.peer"
  PEER_IP="$PEER_B_IP"
elif [[ "$ROLE" == "b" ]]; then
  PEER_FQDN="node-a.csit.peer"
  PEER_IP="$PEER_A_IP"
else
  echo "ERROR: role must be a or b" >&2
  exit 1
fi

BLOCK="# BEGIN csit-peer
csit.peer:53 {
    errors
    cache 30
    hosts {
        ${PEER_IP} ${PEER_FQDN}
        fallthrough
    }
    forward . /etc/resolv.conf {
        max_concurrent 1000
    }
}
# END csit-peer"

CURRENT=$(kubectl --context "$CTX" get configmap coredns -n kube-system -o jsonpath='{.data.Corefile}')
STRIPPED=$(printf '%s' "$CURRENT" | sed '/^# BEGIN csit-peer$/,/^# END csit-peer$/d')
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

echo "CoreDNS updated on $CTX: ${PEER_FQDN} -> ${PEER_IP}"
