#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Start cloud-provider-kind (https://github.com/kubernetes-sigs/cloud-provider-kind) so KinD
# Services can get LoadBalancer IPs. Darwin: refreshes sudo first (interactive password ok),
# then runs the controller under sudo in the background and waits until it is running.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${ROOT}/.gen"
STDERR_LOG="${LOG_DIR}/cloud-provider-kind.stderr"

mkdir -p "$LOG_DIR"
: >"$STDERR_LOG"

command -v go >/dev/null || {
  echo "ERROR: go is required for cloud-provider-kind" >&2
  exit 1
}

echo "Starting cloud-provider-kind for KinD LoadBalancer support..."
OS="$(uname -s)"

if [[ "$OS" == "Darwin" ]]; then
  echo "macOS: you may be prompted for your password so sudo can access Docker." >&2
  sudo -v
  sudo go run sigs.k8s.io/cloud-provider-kind@latest >>"$STDERR_LOG" 2>&1 &
elif [[ "$OS" == "Linux" ]]; then
  go run sigs.k8s.io/cloud-provider-kind@latest >>"$STDERR_LOG" 2>&1 &
else
  echo "ERROR: unsupported OS: $OS" >&2
  exit 1
fi

# Wait until controller shows up (first `go run` can compile for a while).
max_attempts=120
for i in $(seq 1 "$max_attempts"); do
  if pgrep -f 'cloud-provider-kind|provider-kind' >/dev/null 2>&1; then
    echo "cloud-provider-kind is running (stderr log: ${STDERR_LOG})."
    exit 0
  fi
  sleep 2
done

echo "ERROR: timed out waiting for cloud-provider-kind to start after ${max_attempts} attempts (~$((max_attempts * 2))s)." >&2
echo "Last lines from ${STDERR_LOG}:" >&2
tail -50 "$STDERR_LOG" >&2 || true
exit 1
