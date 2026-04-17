#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Optional: KinD configs that publish ingress on 127.0.0.2:80/443 need that address
# on the Docker host namespace. The default kind-slim-multi-host Kind configs use
# 127.0.0.1 + edge/nginx instead — you usually do not need this script.
#
# When used: Docker binds that address
# in its own namespace. On Docker Desktop (macOS/Windows), `ifconfig lo0 alias` on
# the Mac does NOT create 127.0.0.2 where Docker listens — use this script first.
#
# On Linux with Docker Engine, either `sudo ip addr add 127.0.0.2/8 dev lo` on the
# host or this script (uses a privileged --network host container).
set -euo pipefail

SECOND="${KIND_SECOND_LOOPBACK:-127.0.0.2}"

have_second() {
  if command -v ip >/dev/null 2>&1; then
    ip -4 addr show dev lo 2>/dev/null | grep -q "inet ${SECOND}/"
    return $?
  fi
  return 1
}

if have_second; then
  echo "${SECOND} already present on lo (checked with ip)."
  exit 0
fi

if command -v ip >/dev/null 2>&1 && ip addr add "${SECOND}/8" dev lo 2>/dev/null; then
  echo "Added ${SECOND} to lo on this host (ip addr)."
  exit 0
fi

echo "Adding ${SECOND} in Docker's host network namespace (pulls alpine:3.20 once)..."
docker run --rm --privileged --network host alpine:3.20 sh -ceu '
  apk add --no-cache iproute2 >/dev/null
  ip addr add 127.0.0.2/8 dev lo 2>/dev/null || true
  ip -4 addr show dev lo | grep -q "inet 127.0.0.2/" || { echo "ERROR: 127.0.0.2 not on lo after add." >&2; exit 1; }
  echo "127.0.0.2 is available for Kind port publishing."
'
