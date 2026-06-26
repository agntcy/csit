#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Builds docs/index.html from docs/sources.json.
# GitHub Pages serves from the gh-pages branch /docs folder; all paths below
# are relative to that docs root.
#
# Required env:
#   GITHUB_REPOSITORY   e.g. agntcy/csit
#
# Optional env:
#   SOURCES_JSON        path to sources.json (default: site/docs/sources.json)
#   OUTPUT              landing page path (default: site/docs/index.html)
#
# New report paths (under docs/, from sources.json):
#   a2a/, slim-integration/, benchmarks/slim/, slim-multicluster-private/, directory/

set -euo pipefail

SOURCES_JSON="${SOURCES_JSON:-site/docs/sources.json}"
OUTPUT="${OUTPUT:-site/docs/index.html}"
GITHUB_REPOSITORY="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY required}"
DOCS_ROOT="$(dirname "$OUTPUT")"

if [[ ! -f "$SOURCES_JSON" ]]; then
  "$(dirname "$0")/init-sources-json.sh" "$SOURCES_JSON"
fi

mkdir -p "$DOCS_ROOT"
touch "$DOCS_ROOT/.nojekyll"

status_class() {
  case "$1" in
    success) echo "status-success" ;;
    failure) echo "status-failure" ;;
    cancelled) echo "status-cancelled" ;;
    skipped) echo "status-skipped" ;;
    *) echo "status-unknown" ;;
  esac
}

status_label() {
  case "$1" in
    success) echo "Success" ;;
    failure) echo "Failure" ;;
    cancelled) echo "Cancelled" ;;
    skipped) echo "Skipped" ;;
    "") echo "Unknown" ;;
    *) echo "$1" ;;
  esac
}

run_link() {
  local run_id="$1"
  if [[ -n "$run_id" ]]; then
    printf '<a href="https://github.com/%s/actions/runs/%s">#%s</a>' "$GITHUB_REPOSITORY" "$run_id" "$run_id"
  else
    printf '—'
  fi
}

report_index_exists() {
  local report_path="$1"
  [[ -n "$report_path" && -f "${DOCS_ROOT}/${report_path%/}/index.html" ]]
}

{
  cat <<'HTML'
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CSIT Test Reports</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f5f1e8;
      --panel: rgba(255, 252, 245, 0.92);
      --text: #1f2933;
      --muted: #52606d;
      --accent: #0f766e;
      --accent-strong: #134e4a;
      --border: rgba(15, 118, 110, 0.18);
      --shadow: 0 24px 60px rgba(31, 41, 51, 0.12);
      --success: #166534;
      --failure: #b91c1c;
      --cancelled: #92400e;
      --skipped: #64748b;
      --unknown: #475569;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", Georgia, serif;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(15, 118, 110, 0.16), transparent 30%),
        radial-gradient(circle at bottom right, rgba(180, 83, 9, 0.12), transparent 28%),
        linear-gradient(180deg, #fbf8f3 0%, var(--bg) 100%);
      padding: 48px 20px;
    }
    main {
      max-width: 1080px;
      margin: 0 auto;
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 28px;
      box-shadow: var(--shadow);
      padding: 40px;
    }
    h1 {
      margin: 0 0 12px;
      font-size: clamp(2.5rem, 4vw, 4rem);
      line-height: 0.95;
      letter-spacing: -0.04em;
    }
    p { margin: 0 0 16px; font-size: 1.05rem; line-height: 1.7; color: var(--muted); }
    .eyebrow {
      display: inline-block;
      font-size: 0.78rem;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: var(--accent);
      margin-bottom: 14px;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      margin: 24px 0 32px;
      font-size: 0.98rem;
    }
    th, td {
      text-align: left;
      padding: 12px 14px;
      border-bottom: 1px solid var(--border);
      vertical-align: top;
    }
    th { color: var(--muted); font-weight: 600; font-size: 0.82rem; text-transform: uppercase; letter-spacing: 0.08em; }
    .status { font-weight: 600; }
    .status-success { color: var(--success); }
    .status-failure { color: var(--failure); }
    .status-cancelled { color: var(--cancelled); }
    .status-skipped { color: var(--skipped); }
    .status-unknown { color: var(--unknown); }
    .note { display: block; margin-top: 4px; font-size: 0.88rem; color: var(--muted); }
    .card-grid { display: grid; gap: 18px; margin-top: 12px; }
    .report-card, .report-card-disabled {
      display: block;
      text-decoration: none;
      color: inherit;
      background: rgba(255, 255, 255, 0.8);
      border: 1px solid var(--border);
      border-radius: 20px;
      padding: 24px;
    }
    .report-card {
      transition: transform 160ms ease, box-shadow 160ms ease, border-color 160ms ease;
    }
    .report-card:hover {
      transform: translateY(-2px);
      border-color: rgba(15, 118, 110, 0.34);
      box-shadow: 0 20px 40px rgba(15, 118, 110, 0.12);
    }
    .report-card-disabled { opacity: 0.62; }
    .report-card h2, .report-card-disabled h2 { margin: 0 0 10px; font-size: 1.55rem; }
    .report-card span { color: var(--accent-strong); font-weight: 600; }
    .report-card-disabled span { color: var(--muted); font-weight: 600; }
    footer { margin-top: 32px; font-size: 0.95rem; color: var(--muted); }
    @media (max-width: 640px) {
      body { padding: 20px 14px; }
      main { padding: 28px 20px; border-radius: 22px; }
      table { font-size: 0.9rem; }
    }
  </style>
</head>
<body>
  <main>
    <div class="eyebrow">GitHub Pages</div>
    <h1>CSIT Test Reports</h1>
    <p>Static report outputs from CI. The dashboard lists the latest workflow run separately from the report currently published on this site.</p>
    <table>
      <thead>
        <tr>
          <th>Workflow</th>
          <th>Last run</th>
          <th>Last run status</th>
          <th>Published report</th>
          <th>Report from run</th>
        </tr>
      </thead>
      <tbody>
HTML

  jq -r '.workflows[] | [
    .name,
    (.last_run_id // ""),
    (.last_run_conclusion // ""),
    (.last_run_updated_at // ""),
    (.published_report_run_id // ""),
    (.published_report_updated_at // ""),
    (.report_path // ""),
    (.id // "")
  ] | @tsv' "$SOURCES_JSON" | while IFS=$'\t' read -r name last_run_id last_run_conclusion last_run_updated_at published_report_run_id published_report_updated_at report_path workflow_id; do
    class="$(status_class "$last_run_conclusion")"
    label="$(status_label "$last_run_conclusion")"
    last_run_cell="$(run_link "$last_run_id")"
    if [[ -n "$last_run_updated_at" ]]; then
      last_run_cell+=$'\n'"<span class=\"note\">${last_run_updated_at}</span>"
    fi

    if report_index_exists "$report_path"; then
      published_cell="<a href=\"./${report_path}\">Open</a>"
      if [[ -n "$published_report_updated_at" ]]; then
        published_cell+=$'\n'"<span class=\"note\">Updated ${published_report_updated_at}</span>"
      fi
    else
      published_cell="Not published"
    fi

    report_from_run_cell="$(run_link "$published_report_run_id")"
    if [[ -n "$published_report_run_id" && -n "$last_run_id" && "$published_report_run_id" != "$last_run_id" ]]; then
      report_from_run_cell+=$'\n'"<span class=\"note\">Older than last run</span>"
    fi

    printf '        <tr>\n'
    printf '          <td>%s</td>\n' "$name"
    printf '          <td>%s</td>\n' "$last_run_cell"
    printf '          <td class="status %s">%s</td>\n' "$class" "$label"
    printf '          <td>%s</td>\n' "$published_cell"
    printf '          <td>%s</td>\n' "$report_from_run_cell"
    printf '        </tr>\n'
  done

  cat <<'HTML'
      </tbody>
    </table>
    <div class="card-grid">
HTML

  jq -r '.workflows[] | [
    .name,
    (.report_path // ""),
    (.published_report_run_id // ""),
    (.last_run_id // ""),
    (.published_report_updated_at // ""),
    (.id // "")
  ] | @tsv' "$SOURCES_JSON" | while IFS=$'\t' read -r name report_path published_report_run_id last_run_id published_report_updated_at workflow_id; do
    case "$workflow_id" in
      test-a2a)
        blurb="Cross-SDK interoperability results with merged JSON, XML, and HTML dashboard output."
        ;;
      test-a2a-slimrpc)
        blurb="Cross-language A2A-over-SlimRPC interoperability results with merged JSON, XML, and HTML dashboard output."
        ;;
      test-slim-integration)
        blurb="KinD multicluster Slim topology integration tests with bindings examples."
        ;;
      test-slim-benchmarks)
        blurb="Throughput and latency benchmark dashboards across modes, payload sizes, and sender counts."
        ;;
      test-slim-multicluster-private)
        blurb="Two-cluster SPIRE federation verification with private cluster B constraints."
        ;;
      test-directory-conformance)
        blurb="Client/server conformance results across supported Directory client and server versions."
        ;;
      *)
        blurb="Published CI report output."
        ;;
    esac

    if report_index_exists "$report_path"; then
      printf '      <a class="report-card" href="./%s">\n' "$report_path"
      printf '        <h2>%s</h2>\n' "$name"
      printf '        <p>%s</p>\n' "$blurb"
      if [[ -n "$published_report_run_id" && -n "$last_run_id" && "$published_report_run_id" != "$last_run_id" ]]; then
        printf '        <span>Open report (from run #%s)</span>\n' "$published_report_run_id"
      else
        printf '        <span>Open report</span>\n'
      fi
      printf '      </a>\n'
    else
      printf '      <div class="report-card-disabled">\n'
      printf '        <h2>%s</h2>\n' "$name"
      printf '        <p>%s</p>\n' "$blurb"
      printf '        <span>No report published yet</span>\n'
      printf '      </div>\n'
    fi
  done

  cat <<'HTML'
    </div>
    <footer>Reports are published under <code>gh-pages/docs</code> after workflow runs on <code>main</code>. When a run produces no artifact, the previous published report remains available.</footer>
  </main>
</body>
</html>
HTML
} > "$OUTPUT"

echo "wrote $OUTPUT"
