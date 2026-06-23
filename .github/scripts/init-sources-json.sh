#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Writes a default sources.json when missing under docs/.

set -euo pipefail

OUTPUT="${1:?sources.json path required}"

if [[ -f "$OUTPUT" ]]; then
  exit 0
fi

mkdir -p "$(dirname "$OUTPUT")"
cat > "$OUTPUT" <<'JSON'
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
    }
  ]
}
JSON

echo "initialized $OUTPUT"
