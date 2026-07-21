#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Builds docs/index.html for the CSIT GitHub Pages landing page.
# GitHub Pages serves from the gh-pages branch /docs folder; all paths below
# are relative to that docs root.
#
# Optional env:
#   OUTPUT   landing page path (default: site/docs/index.html)

set -euo pipefail

OUTPUT="${OUTPUT:-site/docs/index.html}"
DOCS_ROOT="$(dirname "$OUTPUT")"

mkdir -p "$DOCS_ROOT"
touch "$DOCS_ROOT/.nojekyll"

report_index_file() {
  local report_path="$1"
  echo "${DOCS_ROOT}/${report_path%/}/index.html"
}

report_index_exists() {
  local report_path="$1"
  [[ -n "$report_path" && -f "$(report_index_file "$report_path")" ]]
}

report_updated_at() {
  local index_file
  index_file="$(report_index_file "$1")"
  if [[ -f "$index_file" ]]; then
    date -u -r "$index_file" +"%Y-%m-%d"
  fi
}

render_entry() {
  local name="$1"
  local report_path="$2"
  local blurb="$3"

  if report_index_exists "$report_path"; then
    local updated
    updated="$(report_updated_at "$report_path")"
    cat <<HTML
                    <li class="report-entry">
                        <div class="report-entry-header">
                            <a class="title" href="./${report_path}">${name}</a>
                            <div class="meta">
                                <time datetime="${updated}">Updated ${updated}</time>
                                <span aria-hidden="true">·</span>
                                <a class="open" href="./${report_path}">Open</a>
                            </div>
                        </div>
                        <p class="report-blurb">${blurb}</p>
                    </li>
HTML
  else
    cat <<HTML
                    <li class="report-entry report-entry-disabled">
                        <div class="report-entry-header">
                            <span class="title-disabled">${name}</span>
                            <div class="meta">
                                <span>—</span>
                            </div>
                        </div>
                        <p class="report-blurb">${blurb}</p>
                    </li>
HTML
  fi
}

render_section() {
  local group="$1"
  local title="$2"
  local entries=""

  while IFS='|' read -r entry_group name report_path blurb; do
    [[ "$entry_group" == "$group" ]] || continue
    entries+="$(render_entry "$name" "$report_path" "$blurb")"
  done <<'CATALOG'
integrations|A2A interoperability|a2a/|Cross-SDK interoperability results with merged JSON, XML, and HTML dashboard output.
integrations|A2A SlimRPC interoperability|a2a-slimrpc/|Cross-language A2A-over-SlimRPC interoperability results with merged JSON, XML, and HTML dashboard output.
integrations|Agentic evidence dashboard|agentic-evidence/|Agentic systems taxonomy evidence dashboard (C1 live today) with assertion-based Ginkgo proof per use case.
integrations|Directory conformance|directory/|Client/server conformance results across supported Directory client and server versions.
integrations|Slim integration|slim-integration/|KinD multicluster Slim topology integration tests with bindings examples.
integrations|Slim MCP integration|slim-mcp/|MCP proxy and kubernetes-mcp-server over SLIM in KinD, with client test output.
integrations|Slim multicluster private|slim-multicluster-private/|Two-cluster SPIRE federation verification with private cluster B constraints.
benchmarks|Agent Consensus Convergence|benchmarks/slim-vs-a2a/|SLIM native group-session multicast vs P2P relay-hub streaming: consensus latency, propagation, and RPC counts.
benchmarks|Slim benchmarks|benchmarks/slim/|Throughput and latency benchmark dashboards across modes, payload sizes, and sender counts.
CATALOG

  if [[ -z "$entries" ]]; then
    return
  fi

  cat <<HTML
            <section class="report-section">
                <h2>${title}</h2>
                <ul class="report-list">
${entries}
                </ul>
            </section>

HTML
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
    .intro {
      margin: 0 0 16px;
      font-size: 1.05rem;
      line-height: 1.7;
      color: var(--muted);
    }
    .eyebrow {
      display: inline-block;
      font-size: 0.78rem;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: var(--accent);
      margin-bottom: 14px;
    }
    .report-sections { margin-top: 28px; }
    .report-section + .report-section {
      margin-top: 28px;
      padding-top: 28px;
      border-top: 1px solid var(--border);
    }
    .report-section h2 {
      margin: 0 0 14px;
      font-size: 0.82rem;
      font-weight: 600;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--muted);
    }
    .report-list { list-style: none; margin: 0; padding: 0; }
    .report-entry { padding: 14px 0; }
    .report-entry + .report-entry { border-top: 1px solid var(--border); }
    .report-entry-header {
      display: flex;
      flex-wrap: wrap;
      align-items: baseline;
      gap: 8px 16px;
    }
    .report-entry-header a.title {
      font-size: 1.08rem;
      font-weight: 600;
      color: var(--accent-strong);
      text-decoration: none;
    }
    .report-entry-header a.title:hover { text-decoration: underline; }
    .report-entry-header .title-disabled {
      font-size: 1.08rem;
      font-weight: 600;
      color: var(--text);
    }
    .report-entry-header .meta {
      display: flex;
      flex-wrap: wrap;
      align-items: baseline;
      gap: 8px;
      margin-left: auto;
      font-size: 0.88rem;
      color: var(--muted);
    }
    .report-entry-header .meta time { white-space: nowrap; }
    .report-entry-header .meta a.open {
      color: var(--accent);
      font-weight: 600;
      text-decoration: none;
    }
    .report-entry-header .meta a.open:hover { text-decoration: underline; }
    .report-blurb {
      margin: 6px 0 0;
      font-size: 0.95rem;
      line-height: 1.55;
      color: var(--muted);
      max-width: 72ch;
    }
    .report-entry-disabled { opacity: 0.62; }
    .report-entry-disabled .report-blurb::after {
      content: " · Not published yet";
      font-style: italic;
    }
    footer { margin-top: 32px; font-size: 0.95rem; color: var(--muted); }
    @media (max-width: 640px) {
      body { padding: 20px 14px; }
      main { padding: 28px 20px; border-radius: 22px; }
      .report-entry-header { flex-direction: column; align-items: flex-start; }
      .report-entry-header .meta { margin-left: 0; }
    }
  </style>
</head>
<body>
  <main>
    <div class="eyebrow">GitHub Pages</div>
    <h1>CSIT Test Reports</h1>
    <p class="intro">
      Static test reports from CI on <code>main</code>.
      Pass/fail details live inside each report — this page is just the index.
    </p>

    <div class="report-sections">
HTML

  render_section integrations Integrations
  render_section benchmarks Benchmarks

  cat <<'HTML'
    </div>

    <footer>
      Reports are published under <code>gh-pages/docs</code> after workflow runs on <code>main</code>.
    </footer>
  </main>
</body>
</html>
HTML
} > "$OUTPUT"

echo "wrote $OUTPUT"
