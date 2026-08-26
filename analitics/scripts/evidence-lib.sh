#!/usr/bin/env bash
# Shared helpers for evaluating C1 evidence from Ginkgo assertion reports.

evidence_json() {
  echo "${REPORTS_DIR}/c1-evidence.json"
}

c2_evidence_json() {
  if [[ -n "${C2_EVIDENCE_JSON:-}" ]]; then
    echo "$C2_EVIDENCE_JSON"
    return 0
  fi
  if [[ -f "${REPORTS_DIR}/c2-evidence.json" ]]; then
    echo "${REPORTS_DIR}/c2-evidence.json"
    return 0
  fi
  echo "${REPORTS_DIR}/c2-evidence.json"
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

row_status() {
  local row_id="$1"
  local expected_scenarios="${2:-0}"
  local json_file
  json_file="$(c2_evidence_json)"

  if [[ ! -f "$json_file" ]]; then
    echo "unknown"
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo "unknown"
    return 0
  fi

  local statuses
  statuses="$(jq -r --arg row_id "$row_id" '[.cases[] | select(.row_id == $row_id) | .status] | .[]' "$json_file" 2>/dev/null)"
  if [[ -z "$statuses" ]]; then
    echo "unknown"
    return 0
  fi

  local status verified_count=0 failed_count=0 total_count=0
  while IFS= read -r status; do
    [[ -z "$status" || "$status" == "null" ]] && continue
    total_count=$((total_count + 1))
    case "$status" in
      failed) failed_count=$((failed_count + 1)) ;;
      verified) verified_count=$((verified_count + 1)) ;;
    esac
  done <<< "$statuses"

  if [[ "$failed_count" -gt 0 ]]; then
    echo "failed"
    return 0
  fi
  if [[ "$verified_count" -eq 0 ]]; then
    echo "unknown"
    return 0
  fi
  if [[ "$expected_scenarios" -gt 0 ]]; then
    if [[ "$verified_count" -eq "$expected_scenarios" ]]; then
      echo "verified"
      return 0
    fi
    if [[ "$verified_count" -lt "$expected_scenarios" ]]; then
      echo "partial"
      return 0
    fi
  fi
  if [[ "$verified_count" -eq "$total_count" ]]; then
    echo "verified"
    return 0
  fi
  echo "unknown"
}

row_scenario_count() {
  local row_id="$1"
  local json_file
  json_file="$(c2_evidence_json)"

  if [[ ! -f "$json_file" ]]; then
    echo "0"
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo "0"
    return 0
  fi

  jq -r --arg row_id "$row_id" '[.cases[] | select(.row_id == $row_id)] | length' "$json_file" 2>/dev/null || echo "0"
}

status_label() {
  case "$1" in
    verified) echo "Verified" ;;
    failed) echo "Failed" ;;
    partial) echo "Partial" ;;
    planned) echo "Evidence pending" ;;
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
