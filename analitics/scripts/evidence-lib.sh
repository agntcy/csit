#!/usr/bin/env bash
# Shared helpers for evaluating C1 smoke benchmark evidence from results.tsv.

mode_rows() {
  local mode="$1"
  local results_file="${REPORTS_DIR}/results.tsv"
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
  local results_file="${REPORTS_DIR}/results.tsv"
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
