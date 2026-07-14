#!/usr/bin/env bash
# Shared helpers for evaluating C1 evidence from Ginkgo assertion reports.

evidence_json() {
  echo "${REPORTS_DIR}/c1-evidence.json"
}

mode_status() {
  local mode="$1"
  local json_file
  json_file="$(evidence_json)"

  if [[ ! -f "$json_file" ]]; then
    echo "unknown"
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo "unknown"
    return 0
  fi

  local status
  status="$(jq -r --arg mode "$mode" '.cases[] | select(.mode == $mode) | .status' "$json_file" 2>/dev/null | head -1)"
  if [[ -n "$status" && "$status" != "null" ]]; then
    echo "$status"
    return 0
  fi
  echo "unknown"
}

mode_rows() {
  local mode="$1"
  local json_file
  json_file="$(evidence_json)"

  if [[ ! -f "$json_file" ]]; then
    echo "0"
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo "0"
    return 0
  fi

  jq -r --arg mode "$mode" '[.cases[] | select(.mode == $mode)] | length' "$json_file" 2>/dev/null || echo "0"
}

status_label() {
  case "$1" in
    verified) echo "Verified" ;;
    failed) echo "Failed" ;;
    partial) echo "Partial" ;;
    planned) echo "Planned" ;;
    *) echo "Unknown" ;;
  esac
}

status_css_class() {
  case "$1" in
    verified) echo "status-verified" ;;
    failed) echo "status-failed" ;;
    partial) echo "status-partial" ;;
    planned) echo "status-planned" ;;
    *) echo "status-unknown" ;;
  esac
}

html_escape() {
  local value="$1"
  value="${value//&/&amp;}"
  value="${value//</&lt;}"
  value="${value//>/&gt;}"
  value="${value//\"/&quot;}"
  printf '%s' "$value"
}
