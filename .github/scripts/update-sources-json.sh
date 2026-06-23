#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Update one workflow entry in sources.json.
#
# Always updates last_run_* from the triggering workflow run.
# Updates published_report_* only when a new report artifact was published.
#
# Usage: update-sources-json.sh <sources.json> <workflow_id> <run_id> <conclusion> <published_new_report>

set -euo pipefail

SOURCES_JSON="${1:?sources.json path required}"
WORKFLOW_ID="${2:?workflow id required}"
RUN_ID="${3:?run id required}"
CONCLUSION="${4:?conclusion required}"
PUBLISHED_NEW_REPORT="${5:-false}"
UPDATED_AT="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

"$(dirname "$0")/init-sources-json.sh" "$SOURCES_JSON"

# Migrate legacy single-field schema if present.
if jq -e '.workflows[] | select(has("run_id"))' "$SOURCES_JSON" >/dev/null 2>&1; then
  tmp_migrate="$(mktemp)"
  jq '
    .workflows = [
      .workflows[] |
      if has("last_run_id") then .
      else
        .last_run_id = (.run_id // "") |
        .last_run_conclusion = (.conclusion // "") |
        .last_run_updated_at = (.updated_at // "") |
        .published_report_run_id = (if (.has_report // false) then (.run_id // "") else "" end) |
        .published_report_updated_at = (if (.has_report // false) then (.updated_at // "") else "" end) |
        del(.run_id, .conclusion, .updated_at, .has_report)
      end
    ]
  ' "$SOURCES_JSON" > "$tmp_migrate"
  mv "$tmp_migrate" "$SOURCES_JSON"
fi

published_json="$([[ "$PUBLISHED_NEW_REPORT" == "true" ]] && echo true || echo false)"

tmp="$(mktemp)"
jq \
  --arg id "$WORKFLOW_ID" \
  --arg run_id "$RUN_ID" \
  --arg conclusion "$CONCLUSION" \
  --arg updated_at "$UPDATED_AT" \
  --argjson published_new_report "$published_json" \
  '
  .workflows = [
    .workflows[] |
    if .id == $id then
      .last_run_id = $run_id |
      .last_run_conclusion = $conclusion |
      .last_run_updated_at = $updated_at |
      if $published_new_report then
        .published_report_run_id = $run_id |
        .published_report_updated_at = $updated_at
      else .
      end
    else .
    end
  ]
  ' "$SOURCES_JSON" > "$tmp"
mv "$tmp" "$SOURCES_JSON"

echo "updated sources.json for ${WORKFLOW_ID} (published_new_report=${PUBLISHED_NEW_REPORT})"
