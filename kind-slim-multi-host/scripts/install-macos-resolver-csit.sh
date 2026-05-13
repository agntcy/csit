#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Installs macOS split-DNS for *.csit.test → CoreDNS in docker (kind-slim-multi-host-local-dns).
# Requires: Docker DNS helper listening on CSIT_LOCAL_DNS_HOST_PORT (default 8053).
# Writes /etc/resolver/csit.test (root). See README "macOS host DNS".
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "install-macos-resolver-csit.sh: only macOS is supported (this script writes /etc/resolver/)." >&2
  exit 1
fi

PORT="${CSIT_LOCAL_DNS_HOST_PORT:-8053}"
RESOLVER="/etc/resolver/csit.test"

echo "Installing ${RESOLVER} (nameserver 127.0.0.1 port ${PORT}) — sudo required."
sudo mkdir -p /etc/resolver
sudo tee "${RESOLVER}" >/dev/null <<EOF
nameserver 127.0.0.1
port ${PORT}
EOF
sudo chmod 644 "${RESOLVER}"

echo "Installed ${RESOLVER}"
echo "Flush macOS DNS cache: sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder"
echo "Verify DNS helper: dig @127.0.0.1 -p ${PORT} +short control.nb.cluster-a.csit.test A"
echo "Verify OS resolver: scutil --dns (look for domain csit.test → 127.0.0.1 port ${PORT}); plain dig may still be empty on macOS"
