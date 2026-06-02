#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS_DIR="${REPORTS_DIR:-$BASE_DIR/../benchmarks/agntcy-slim/reports}"
TEMPLATE_FILE="${TEMPLATE_FILE:-$BASE_DIR/templates/dashboard.html.tmpl}"
OUTPUT_FILE="${OUTPUT_FILE:-$BASE_DIR/published/index.html}"

# shellcheck source=evidence-lib.sh
source "$BASE_DIR/scripts/evidence-lib.sh"

mkdir -p "$(dirname "$OUTPUT_FILE")"

generated_at="$(date -u +"%Y-%m-%d %H:%M UTC")"

c1_rr_status="$(mode_status "request-reply")"
c1_ff_status="$(mode_status "fire-and-forget")"
c1_w_status="$(mode_status "write")"
c1_rr_rows="$(mode_rows "request-reply")"
c1_ff_rows="$(mode_rows "fire-and-forget")"
c1_w_rows="$(mode_rows "write")"

render_status_span() {
  local status="$1"
  local label
  local css
  label="$(status_label "$status")"
  css="$(status_css_class "$status")"
  printf '<span class="status %s">%s</span>' "$css" "$label"
}

C1_RR_STATUS_SPAN="$(render_status_span "$c1_rr_status")"
C1_FF_STATUS_SPAN="$(render_status_span "$c1_ff_status")"
C1_W_STATUS_SPAN="$(render_status_span "$c1_w_status")"

# Overall C1 class status: failed if any failed, unknown if any unknown, else verified
c1_class_status="verified"
for s in "$c1_rr_status" "$c1_ff_status" "$c1_w_status"; do
  if [[ "$s" == "failed" ]]; then
    c1_class_status="failed"
    break
  fi
  if [[ "$s" == "unknown" ]]; then
    c1_class_status="unknown"
  fi
done
C1_CLASS_STATUS_SPAN="$(render_status_span "$c1_class_status")"

sed \
  -e "s|{{GENERATED_AT}}|$generated_at|g" \
  -e "s|{{C1_CLASS_STATUS_SPAN}}|$C1_CLASS_STATUS_SPAN|g" \
  -e "s|{{C1_RR_STATUS_SPAN}}|$C1_RR_STATUS_SPAN|g" \
  -e "s|{{C1_FF_STATUS_SPAN}}|$C1_FF_STATUS_SPAN|g" \
  -e "s|{{C1_W_STATUS_SPAN}}|$C1_W_STATUS_SPAN|g" \
  -e "s|{{C1_RR_ROWS}}|$c1_rr_rows|g" \
  -e "s|{{C1_FF_ROWS}}|$c1_ff_rows|g" \
  -e "s|{{C1_W_ROWS}}|$c1_w_rows|g" \
  "$TEMPLATE_FILE" > "$OUTPUT_FILE"

echo "Rendered $OUTPUT_FILE"
