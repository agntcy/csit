#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# After stack:install / compose:dns:up, confirms the host DNS helper answers cluster-A
# ingress names with the same IP cloud-provider-kind assigned (.gen/ingress-a.env).
#
# Env: CSIT_DNS_ZONE (default csit.test), CSIT_LOCAL_DNS_HOST_PORT (default 8053),
#      CSIT_DNS_VERIFY_RETRIES (default 15), CSIT_DNS_VERIFY_SLEEP (default 1)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT}/.gen/ingress-a.env"
ZONE_FILE="${ROOT}/compose/dns/csittest.zone"
ZONE="${CSIT_DNS_ZONE:-csit.test}"
PORT="${CSIT_LOCAL_DNS_HOST_PORT:-8053}"
RETRIES="${CSIT_DNS_VERIFY_RETRIES:-15}"
SLEEP="${CSIT_DNS_VERIFY_SLEEP:-1}"

command -v dig >/dev/null 2>&1 || {
  echo "verify-local-dns-helper: ERROR: dig not found (install bind-dnsutils / bind)" >&2
  exit 1
}

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "verify-local-dns-helper: ERROR: missing ${ENV_FILE} — run task ingress:a:wait-lb-ip first" >&2
  exit 1
fi
# shellcheck source=/dev/null
source "${ENV_FILE}"
if [[ -z "${INGRESS_A_LB_IP:-}" ]]; then
  echo "verify-local-dns-helper: ERROR: INGRESS_A_LB_IP empty in ${ENV_FILE}" >&2
  exit 1
fi

# Must match render-local-dns-corefile.sh: CSIT_INGRESS_IP_A wins over discovered LB IP.
EXPECT_CLUSTER_A="${CSIT_INGRESS_IP_A:-${INGRESS_A_LB_IP}}"

if [[ ! -f "${ZONE_FILE}" ]]; then
  echo "verify-local-dns-helper: ERROR: missing ${ZONE_FILE} — run task compose:dns:render or compose:dns:up" >&2
  exit 1
fi
if ! grep -qF "${EXPECT_CLUSTER_A}" "${ZONE_FILE}"; then
  echo "verify-local-dns-helper: ERROR: ${ZONE_FILE} does not contain cluster-A IP ${EXPECT_CLUSTER_A}" >&2
  echo "Re-run: task compose:dns:render && docker compose -f compose/dns/docker-compose.yaml up -d (or task compose:dns:up)" >&2
  exit 1
fi

dig_a() {
  dig @127.0.0.1 -p "${PORT}" +short "$1" A 2>/dev/null | head -1 | tr -d '\r'
}

verify_all() {
  local h got want
  for h in "control.cluster-a.${ZONE}" "control.nb.cluster-a.${ZONE}" "slim.cluster-a.${ZONE}" "spire.cluster-a.${ZONE}" "spire-bundle.cluster-a.${ZONE}"; do
    got=$(dig_a "${h}")
    want="${EXPECT_CLUSTER_A}"
    if [[ "${got}" != "${want}" ]]; then
      echo "verify-local-dns-helper: ERROR: ${h} → '${got}' (expected '${want}')" >&2
      return 1
    fi
  done
  got=$(dig_a "slim.cluster-b.${ZONE}")
  if [[ "${got}" != "127.0.0.1" ]]; then
    echo "verify-local-dns-helper: ERROR: slim.cluster-b.${ZONE} → '${got}' (expected '127.0.0.1')" >&2
    return 1
  fi
  return 0
}

for ((i = 1; i <= RETRIES; i++)); do
  if verify_all; then
    echo "verify-local-dns-helper: OK — cluster A ingress + SPIRE names → ${EXPECT_CLUSTER_A}; slim.cluster-b.${ZONE} → 127.0.0.1 (dig @127.0.0.1:${PORT})"
    exit 0
  fi
  if [[ "${i}" -lt "${RETRIES}" ]]; then
    sleep "${SLEEP}"
  fi
done

echo "verify-local-dns-helper: ERROR: timed out after ${RETRIES} attempts — is the local-dns container up? (task compose:dns:up)" >&2
exit 1
