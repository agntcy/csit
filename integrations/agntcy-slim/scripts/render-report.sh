#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Render Slim integration Ginkgo JSON reports to HTML.
#
# Usage: render-report.sh <reports-dir> <output-html> [title]

set -euo pipefail

REPORTS_DIR="${1:?reports directory required}"
OUTPUT="${2:?output HTML path required}"
TITLE="${3:-Slim integration}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INTEGRATIONS_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

resolve_path() {
  if [[ "$1" = /* ]]; then
    printf '%s\n' "$1"
    return
  fi

  local dir base
  dir="$(dirname "$1")"
  base="$(basename "$1")"
  if [[ "$dir" == "." ]]; then
    printf '%s\n' "$(pwd)/$base"
  else
    printf '%s\n' "$(cd "$dir" && pwd)/$base"
  fi
}

REPORTS_DIR="$(resolve_path "$REPORTS_DIR")"
OUTPUT="$(resolve_path "$OUTPUT")"

(
  cd "$INTEGRATIONS_ROOT"
  go run ./agntcy-slim/tools/report_dashboard.go \
    --reports-dir "$REPORTS_DIR" \
    --output "$OUTPUT" \
    --title "$TITLE"
)
