#!/bin/sh
set -eu

SPIRE_SERVER_PRIVATE_DIR="${SPIRE_SERVER_PRIVATE_DIR}" # e.g. /tmp/spire-server/private
SPIRE_SERVER_SOCKET_PATH="$SPIRE_SERVER_PRIVATE_DIR/api.sock" # e.g. /tmp/spire-server/private/api.sock

# dir-client (host) entry
SPIRE_AGENT_HOST_SPIFFE_ID="${SPIRE_AGENT_HOST_SPIFFE_ID}" # e.g. spiffe://example.org/nodes/host-agent
UNIX_UID="${UNIX_UID}" # e.g. 501

/opt/spire/bin/spire-server entry create \
  -socketPath "$SPIRE_SERVER_SOCKET_PATH" \
  -spiffeID "spiffe://example.org/dir-client" \
  -parentID "$SPIRE_AGENT_HOST_SPIFFE_ID" \
  -selector "unix:uid:${UNIX_UID}"

# dir-daemon (Docker) entry
COMPOSE_PROJECT="${COMPOSE_PROJECT}" # e.g. v130-spire-dir
SPIRE_AGENT_DOCKER_SPIFFE_ID="${SPIRE_AGENT_DOCKER_SPIFFE_ID}" # e.g. spiffe://example.org/nodes/docker-agent

/opt/spire/bin/spire-server entry create \
  -socketPath "$SPIRE_SERVER_SOCKET_PATH" \
  -spiffeID "spiffe://example.org/dir-daemon" \
  -parentID "$SPIRE_AGENT_DOCKER_SPIFFE_ID" \
  -selector "docker:label:com.docker.compose.project:${COMPOSE_PROJECT}" \
  -selector docker:label:com.docker.compose.service:daemon
