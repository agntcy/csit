#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS_DIR="${REPORTS_DIR:-$BASE_DIR/../benchmarks/agntcy-slim/reports}"
TEMPLATE_FILE="${TEMPLATE_FILE:-$BASE_DIR/templates/c1-summary.md.tmpl}"
OUTPUT_FILE="${OUTPUT_FILE:-$BASE_DIR/published/c1-evidence-summary.md}"

# shellcheck source=evidence-lib.sh
source "$BASE_DIR/scripts/evidence-lib.sh"

mkdir -p "$(dirname "$OUTPUT_FILE")"

generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
rr_status="$(mode_status "request-reply")"
ff_status="$(mode_status "fire-and-forget")"
w_status="$(mode_status "write")"
rr_rows="$(mode_rows "request-reply")"
ff_rows="$(mode_rows "fire-and-forget")"
w_rows="$(mode_rows "write")"

sed \
  -e "s|{{GENERATED_AT_UTC}}|$generated_at|g" \
  -e "s|{{STATUS_REQUEST_REPLY}}|$rr_status|g" \
  -e "s|{{STATUS_FIRE_AND_FORGET}}|$ff_status|g" \
  -e "s|{{STATUS_WRITE}}|$w_status|g" \
  -e "s|{{ROWS_REQUEST_REPLY}}|$rr_rows|g" \
  -e "s|{{ROWS_FIRE_AND_FORGET}}|$ff_rows|g" \
  -e "s|{{ROWS_WRITE}}|$w_rows|g" \
  "$TEMPLATE_FILE" > "$OUTPUT_FILE"

echo "Rendered $OUTPUT_FILE"
