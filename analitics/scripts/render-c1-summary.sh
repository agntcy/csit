#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS_DIR="${REPORTS_DIR:-$BASE_DIR/../benchmarks/agntcy-slim/reports}"
TEMPLATE_FILE="${TEMPLATE_FILE:-$BASE_DIR/templates/c1-summary.md.tmpl}"
OUTPUT_FILE="${OUTPUT_FILE:-$BASE_DIR/published/c1-evidence-summary.md}"

mkdir -p "$(dirname "$OUTPUT_FILE")"

mode_rows() {
  local mode="$1"
  local results_file="$REPORTS_DIR/results.tsv"
  if [[ ! -f "$results_file" ]]; then
    echo "0"
    return 0
  fi

  awk -F'\t' -v wanted_mode="$mode" '
    NR == 1 {
      for (i = 1; i <= NF; i++) {
        if ($i == "mode") mode_col = i
      }
      next
    }
    $mode_col == wanted_mode { count++ }
    END { print count + 0 }
  ' "$results_file"
}

mode_status() {
  local mode="$1"
  local results_file="$REPORTS_DIR/results.tsv"
  if [[ ! -f "$results_file" ]]; then
    echo "unknown"
    return 0
  fi

  awk -F'\t' -v wanted_mode="$mode" '
    NR == 1 {
      for (i = 1; i <= NF; i++) {
        if ($i == "mode") mode_col = i
        if ($i == "sender_runtime_errors") sender_err_col = i
        if ($i == "sink_errors") sink_err_col = i
      }
      next
    }
    $mode_col == wanted_mode {
      rows++
      if ($sender_err_col != 0 || $sink_err_col != 0) {
        bad = 1
      }
    }
    END {
      if (rows == 0) {
        print "unknown"
      } else if (bad == 1) {
        print "failed"
      } else {
        print "verified"
      }
    }
  ' "$results_file"
}

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
