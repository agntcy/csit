#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Canonical sources.json for gh-pages docs/.
#
# - Creates the file when missing.
# - Merges any new workflow entries into an existing file (by id), without
#   overwriting last_run_* / published_report_* on entries that already exist.
#
# Add new publishable workflows here only.

set -euo pipefail

OUTPUT="${1:?sources.json path required}"

defaults="$(mktemp)"
trap 'rm -f "$defaults"' EXIT

cat > "$defaults" <<'JSON'
{
  "workflows": [
    {
      "id": "test-a2a",
      "name": "A2A interoperability",
      "workflow_file": "test-a2a.yaml",
      "report_path": "a2a/",
      "last_run_id": "",
      "last_run_conclusion": "",
      "last_run_updated_at": "",
      "published_report_run_id": "",
      "published_report_updated_at": ""
    },
    {
      "id": "test-a2a-slimrpc",
      "name": "A2A SlimRPC interoperability",
      "workflow_file": "test-a2a-slimrpc.yaml",
      "report_path": "a2a-slimrpc/",
      "last_run_id": "",
      "last_run_conclusion": "",
      "last_run_updated_at": "",
      "published_report_run_id": "",
      "published_report_updated_at": ""
    },
    {
      "id": "test-slim-integration",
      "name": "Slim integration",
      "workflow_file": "test-slim-integration.yaml",
      "report_path": "slim-integration/",
      "last_run_id": "",
      "last_run_conclusion": "",
      "last_run_updated_at": "",
      "published_report_run_id": "",
      "published_report_updated_at": ""
    },
    {
      "id": "test-slim-mcp",
      "name": "Slim MCP integration",
      "workflow_file": "test-slim-mcp.yaml",
      "report_path": "slim-mcp/",
      "last_run_id": "",
      "last_run_conclusion": "",
      "last_run_updated_at": "",
      "published_report_run_id": "",
      "published_report_updated_at": ""
    },
    {
      "id": "test-slim-benchmarks",
      "name": "Slim benchmarks",
      "workflow_file": "test-slim-benchmarks.yaml",
      "report_path": "benchmarks/slim/",
      "last_run_id": "",
      "last_run_conclusion": "",
      "last_run_updated_at": "",
      "published_report_run_id": "",
      "published_report_updated_at": ""
    },
    {
      "id": "test-slim-agent-consensus",
      "name": "Agent Consensus Convergence",
      "workflow_file": "test-slim-agent-consensus.yaml",
      "report_path": "slim-agent-consensus/",
      "last_run_id": "",
      "last_run_conclusion": "",
      "last_run_updated_at": "",
      "published_report_run_id": "",
      "published_report_updated_at": ""
    },
    {
      "id": "test-slim-multicluster-private",
      "name": "Slim multicluster private",
      "workflow_file": "test-slim-multicluster-private.yaml",
      "report_path": "slim-multicluster-private/",
      "last_run_id": "",
      "last_run_conclusion": "",
      "last_run_updated_at": "",
      "published_report_run_id": "",
      "published_report_updated_at": ""
    },
    {
      "id": "test-directory-conformance",
      "name": "Directory conformance",
      "workflow_file": "test-directory-conformance.yaml",
      "report_path": "directory/",
      "last_run_id": "",
      "last_run_conclusion": "",
      "last_run_updated_at": "",
      "published_report_run_id": "",
      "published_report_updated_at": ""
    },
    {
      "id": "test-agentic-evidence",
      "name": "Agentic evidence dashboard",
      "workflow_file": "test-agentic-evidence.yaml",
      "report_path": "agentic-evidence/",
      "last_run_id": "",
      "last_run_conclusion": "",
      "last_run_updated_at": "",
      "published_report_run_id": "",
      "published_report_updated_at": ""
    }
  ]
}
JSON

mkdir -p "$(dirname "$OUTPUT")"

if [[ ! -f "$OUTPUT" ]]; then
  cp "$defaults" "$OUTPUT"
  echo "initialized $OUTPUT"
  exit 0
fi

tmp="$(mktemp)"
jq --slurpfile defaults "$defaults" '
  ([.workflows[].id]) as $existing_ids |
  .workflows += [
    $defaults[0].workflows[] |
    select(.id as $id | ($existing_ids | index($id) | not))
  ] |
  ($defaults[0].workflows | map(.id)) as $order |
  (.workflows | map({(.id): .}) | add) as $by_id |
  .workflows = [$order[] | $by_id[.]]
' "$OUTPUT" > "$tmp"
mv "$tmp" "$OUTPUT"

echo "synced workflow catalog into $OUTPUT"
