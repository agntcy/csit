#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Download workflow run artifacts when present; exit 0 when missing.
#
# Usage:
#   download-workflow-artifact.sh <run_id> <dest_dir> --name <artifact_name>
#   download-workflow-artifact.sh <run_id> <dest_dir> --pattern <glob>
#
# Requires: GH_TOKEN or GITHUB_TOKEN, GITHUB_REPOSITORY

set -euo pipefail

RUN_ID="${1:?run id required}"
DEST="${2:?destination directory required}"
shift 2

REPO="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY required}"
TOKEN="${GH_TOKEN:-${GITHUB_TOKEN:-}}"
MODE=""
TARGET=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name)
      MODE="name"
      TARGET="${2:?artifact name required after --name}"
      shift 2
      ;;
    --pattern)
      MODE="pattern"
      TARGET="${2:?artifact pattern required after --pattern}"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "$MODE" ]]; then
  echo "specify --name or --pattern" >&2
  exit 1
fi

mkdir -p "$DEST"

api() {
  if [[ -n "$TOKEN" ]]; then
    gh api "$@" --header "Authorization: Bearer ${TOKEN}"
  else
    gh api "$@"
  fi
}

artifact_names="$(api "repos/${REPO}/actions/runs/${RUN_ID}/artifacts" --jq '.artifacts[].name' || true)"
if [[ -z "$artifact_names" ]]; then
  echo "No artifacts found for run ${RUN_ID}; skipping download"
  exit 0
fi

artifact_present=false
case "$MODE" in
  name)
    if grep -Fxq "$TARGET" <<< "$artifact_names"; then
      artifact_present=true
    fi
    ;;
  pattern)
    while IFS= read -r name; do
      [[ -z "$name" ]] && continue
      case "$name" in
        $TARGET)
          artifact_present=true
          break
          ;;
      esac
    done <<< "$artifact_names"
    ;;
esac

if [[ "$artifact_present" != "true" ]]; then
  echo "Artifact (${MODE}=${TARGET}) not found in run ${RUN_ID}; skipping download"
  exit 0
fi

download_args=(run download "$RUN_ID" --repo "$REPO" --dir "$DEST")
case "$MODE" in
  name) download_args+=(--name "$TARGET") ;;
  pattern) download_args+=(--pattern "$TARGET") ;;
esac

gh "${download_args[@]}"
echo "Downloaded artifacts (${MODE}=${TARGET}) to ${DEST}"
