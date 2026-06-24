#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Build one gh-pages docs section from a staging directory populated by download-artifact.
#
# Required env:
#   SUITE         a2a | slim-integration | slim-benchmarks | slim-multicluster-private | directory-conformance
#   STAGING_DIR   directory with downloaded artifact contents
#   SITE_DIR      gh-pages docs root (e.g. site/docs; suite dirs a2a/, directory/, …)
#   REPO_ROOT     csit checkout (for go run)

set -euo pipefail

SUITE="${SUITE:?SUITE required}"
STAGING_DIR="${STAGING_DIR:?STAGING_DIR required}"
SITE_DIR="${SITE_DIR:?SITE_DIR required}"
REPO_ROOT="${REPO_ROOT:-$PWD}"

has_report=false

case "$SUITE" in
  a2a)
    mkdir -p "$SITE_DIR/a2a"
    find "$STAGING_DIR" -type f \( -name '*.json' -o -name '*.xml' \) -exec cp {} "$SITE_DIR/a2a/" \;
    if compgen -G "$SITE_DIR/a2a/*.json" > /dev/null; then
      (
        cd "$REPO_ROOT/integrations"
        go run ./agntcy-a2a/tools/report_dashboard.go \
          --reports-dir "$SITE_DIR/a2a" \
          --output "$SITE_DIR/a2a/index.html"
      )
      has_report=true
    fi
    ;;

  slim-integration)
    mkdir -p "$SITE_DIR/slim-integration"
    find "$STAGING_DIR" -type f \( -name '*.json' -o -name '*.xml' \) -exec cp {} "$SITE_DIR/slim-integration/" \;
    if compgen -G "$SITE_DIR/slim-integration/*.json" > /dev/null; then
      "$(dirname "$0")/render-slim-integration-report.sh" \
        "$SITE_DIR/slim-integration" \
        "$SITE_DIR/slim-integration/index.html" \
        "Slim integration"
      has_report=true
    fi
    ;;

  slim-benchmarks)
    if [[ -d "$STAGING_DIR/smoke" ]]; then
      mkdir -p "$SITE_DIR/benchmarks/slim/smoke"
      cp -a "$STAGING_DIR/smoke/." "$SITE_DIR/benchmarks/slim/smoke/"
    fi
    capacity_dir=""
    if [[ -d "$STAGING_DIR/capacity" ]] && [[ -n "$(ls -A "$STAGING_DIR/capacity" 2>/dev/null)" ]]; then
      mkdir -p "$SITE_DIR/benchmarks/slim/capacity"
      cp -a "$STAGING_DIR/capacity/." "$SITE_DIR/benchmarks/slim/capacity/"
      capacity_dir="$SITE_DIR/benchmarks/slim/capacity"
    elif [[ -d "$SITE_DIR/benchmarks/slim/capacity" ]] && compgen -G "$SITE_DIR/benchmarks/slim/capacity/results"*.tsv > /dev/null; then
      capacity_dir="$SITE_DIR/benchmarks/slim/capacity"
    fi
    if [[ -d "$STAGING_DIR/basic" ]]; then
      mkdir -p "$SITE_DIR/benchmarks/slim/basic"
      cp -a "$STAGING_DIR/basic/." "$SITE_DIR/benchmarks/slim/basic/"
    fi
    if [[ -d "$SITE_DIR/benchmarks/slim/smoke" || -n "$capacity_dir" || -d "$SITE_DIR/benchmarks/slim/basic" ]]; then
      args=(go run ./agntcy-slim/tools/report_dashboard.go \
        --output "$SITE_DIR/benchmarks/slim/index.html")
      if [[ -d "$SITE_DIR/benchmarks/slim/smoke" ]]; then
        args+=(--smoke-dir "$SITE_DIR/benchmarks/slim/smoke")
      fi
      if [[ -n "$capacity_dir" ]]; then
        args+=(--capacity-dir "$capacity_dir")
      fi
      if [[ -f "$SITE_DIR/benchmarks/slim/basic/basic-benchmark-results.csv" ]]; then
        args+=(--basic-csv "$SITE_DIR/benchmarks/slim/basic/basic-benchmark-results.csv")
      fi
      (cd "$REPO_ROOT/benchmarks" && "${args[@]}")
      has_report=true
    fi
    ;;

  slim-multicluster-private)
    summary=""
    if [[ -f "$STAGING_DIR/summary.html" ]]; then
      summary="$STAGING_DIR/summary.html"
    else
      summary="$(find "$STAGING_DIR" -name 'summary.html' -print -quit || true)"
    fi
    if [[ -n "$summary" && -f "$summary" ]]; then
      mkdir -p "$SITE_DIR/slim-multicluster-private"
      cp "$summary" "$SITE_DIR/slim-multicluster-private/index.html"
      has_report=true
    fi
    ;;

  directory-conformance)
    summary=""
    if [[ -f "$STAGING_DIR/summary.html" ]]; then
      summary="$STAGING_DIR/summary.html"
    else
      summary="$(find "$STAGING_DIR" -name 'summary.html' -print -quit || true)"
    fi
    if [[ -n "$summary" && -f "$summary" ]]; then
      mkdir -p "$SITE_DIR/directory"
      cp "$summary" "$SITE_DIR/directory/index.html"
      has_report=true
    fi
    ;;

  *)
    echo "unknown suite: $SUITE" >&2
    exit 1
    ;;
esac

echo "has_report=${has_report}"
