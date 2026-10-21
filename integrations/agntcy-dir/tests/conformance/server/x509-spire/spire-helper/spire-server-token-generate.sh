#!/bin/sh
set -eu

SPIRE_TOKEN_DIR="${SPIRE_TOKEN_DIR}" # e.g. /run/spire/token
SPIRE_SERVER_PRIVATE_DIR="${SPIRE_SERVER_PRIVATE_DIR}" # e.g. /tmp/spire-server/private
SPIRE_SERVER_SOCKET_PATH="$SPIRE_SERVER_PRIVATE_DIR/api.sock" # e.g. /tmp/spire-server/private/api.sock

# SPIRE agent (Docker) join token
SPIRE_AGENT_DOCKER_TOKEN_FILENAME="${SPIRE_AGENT_DOCKER_TOKEN_FILENAME}" # e.g. docker-agent-token.txt
SPIRE_AGENT_DOCKER_SPIFFE_ID="${SPIRE_AGENT_DOCKER_SPIFFE_ID}" # e.g. spiffe://example.org/nodes/docker-agent
SPIRE_AGENT_DOCKER_TOKEN="$SPIRE_TOKEN_DIR/$SPIRE_AGENT_DOCKER_TOKEN_FILENAME" # e.g. /run/spire/token/docker-agent-token.txt

/opt/spire/bin/spire-server token generate \
  -socketPath "$SPIRE_SERVER_SOCKET_PATH" \
  -spiffeID "$SPIRE_AGENT_DOCKER_SPIFFE_ID" \
  | awk '{print $2}' > "$SPIRE_AGENT_DOCKER_TOKEN"

if [ ! -s "$SPIRE_AGENT_DOCKER_TOKEN" ]; then
  echo "FAILED to generate SPIRE agent (Docker) join token" >&2
  exit 1
else
  echo "SPIRE agent (Docker) join token: $(cat $SPIRE_AGENT_DOCKER_TOKEN)"
fi

# SPIRE agent (host) join token
SPIRE_AGENT_HOST_TOKEN_FILENAME="${SPIRE_AGENT_HOST_TOKEN_FILENAME}" # e.g. host-agent-token.txt
SPIRE_AGENT_HOST_SPIFFE_ID="${SPIRE_AGENT_HOST_SPIFFE_ID}" # e.g. spiffe://example.org/nodes/host-agent
SPIRE_AGENT_HOST_TOKEN="$SPIRE_TOKEN_DIR/$SPIRE_AGENT_HOST_TOKEN_FILENAME" # e.g. /run/spire/token/host-agent-token.txt

/opt/spire/bin/spire-server token generate \
  -socketPath "$SPIRE_SERVER_SOCKET_PATH" \
  -spiffeID "$SPIRE_AGENT_HOST_SPIFFE_ID" \
  | awk '{print $2}' > "$SPIRE_AGENT_HOST_TOKEN"

if [ ! -s "$SPIRE_AGENT_HOST_TOKEN" ]; then
  echo "FAILED to generate SPIRE agent (host) join token" >&2
  exit 1
else
  echo "SPIRE agent (host) join token: $(cat $SPIRE_AGENT_HOST_TOKEN)"
fi
