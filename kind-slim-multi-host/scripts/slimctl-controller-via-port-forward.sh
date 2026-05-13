#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Port-forward slim-control northbound gRPC to localhost, run one slimctl controller
# subcommand, then stop the forward. Invoked from Task (real bash): go-task uses mvdan/sh
# for inline cmds, where `set -u` + `$!` after `kubectl ... &` can error with `!: unbound variable`.
#
# Env: SLIMCTL_PATH, CTX_A, CTRL_RELEASE, NS_ADMIN, SLIMCTL_PF_LOCAL_PORT, SLIM_CONTROLLER_HTTP_URL
# Arg: link | route
set -euo pipefail

ACTION="${1:?usage: $0 link|route}"

SLIMCTL="${SLIMCTL_PATH:?}"
case "$SLIMCTL" in
  /*) ;;
  *) SLIMCTL="$(pwd)/$SLIMCTL" ;;
esac

kubectl port-forward --context "${CTX_A:?}" "svc/${CTRL_RELEASE:?}" -n "${NS_ADMIN:?}" \
  "${SLIMCTL_PF_LOCAL_PORT:?}:50051" &
PF_PID=$!
cleanup() { kill "$PF_PID" 2>/dev/null || true; }
trap cleanup EXIT
sleep 2

# Plain http:// controller URL conflicts with global TLS client flags from env/config;
# slimctl then errors after printing output. Clear only the common env overrides for this hop.
unset SLIMCTL_COMMON_OPTS_TLS_INSECURE_SKIP_VERIFY SLIMCTL_COMMON_OPTS_TLS_INSECURE 2>/dev/null || true

case "$ACTION" in
  link)
    "$SLIMCTL" -s "${SLIM_CONTROLLER_HTTP_URL:?}" controller link outline
    ;;
  route)
    "$SLIMCTL" -s "${SLIM_CONTROLLER_HTTP_URL:?}" controller route outline
    ;;
  *)
    echo "$0: unknown action: $ACTION (expected link or route)" >&2
    exit 1
    ;;
esac
