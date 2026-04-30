#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Remove legacy # BEGIN csit-peer … # END csit-peer block from kube-system/coredns Corefile.
# Usage: coredns-strip-legacy-peer.sh <kubectl-context> [<kubectl-context> ...]
set -euo pipefail

command -v jq >/dev/null 2>&1 || { echo "ERROR: jq is required" >&2; exit 1; }

[[ "${#}" -ge 1 ]] || { echo "ERROR: pass at least one kubectl context" >&2; exit 1; }

for CTX in "$@"; do
  CURRENT=$(kubectl --context "$CTX" get configmap coredns -n kube-system -o jsonpath='{.data.Corefile}')
  STRIPPED=$(printf '%s' "$CURRENT" | sed '/^# BEGIN csit-peer$/,/^# END csit-peer$/d')
  if [[ "$CURRENT" == "$STRIPPED" ]]; then
    echo "No csit-peer block on $CTX; skipping"
    continue
  fi
  TMP=$(mktemp)
  printf '%s' "$STRIPPED" >"$TMP"
  kubectl --context "$CTX" get configmap coredns -n kube-system -o json | jq --rawfile cf "$TMP" '.data.Corefile = $cf' | kubectl --context "$CTX" apply -f -
  rm -f "$TMP"
  if kubectl --context "$CTX" get deployment coredns -n kube-system >/dev/null 2>&1; then
    kubectl --context "$CTX" rollout restart deployment/coredns -n kube-system
    kubectl --context "$CTX" rollout status deployment/coredns -n kube-system --timeout=120s
  fi
  echo "Stripped csit-peer block from $CTX"
done
