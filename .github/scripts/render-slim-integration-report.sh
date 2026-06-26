#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Render Slim integration Ginkgo JSON reports to HTML.
#
# Usage: render-slim-integration-report.sh <reports-dir> <output-html> [title]

set -euo pipefail

REPORTS_DIR="${1:?reports directory required}"
OUTPUT="${2:?output HTML path required}"
TITLE="${3:-Slim integration}"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

(
  cd "$REPO_ROOT/integrations"
  go run ./agntcy-slim/tools/report_dashboard.go \
    --reports-dir "$REPORTS_DIR" \
    --output "$OUTPUT" \
    --title "$TITLE"
)
