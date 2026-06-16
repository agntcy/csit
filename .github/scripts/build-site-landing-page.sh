#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Builds site/index.html from environment variables. Keeps all HTML out of
# workflow YAML files.
#
# Required env:
#   GITHUB_REPOSITORY   e.g. agntcy/csit
#   GITHUB_RUN_ID       ID of the workflow run that is deploying the page
#
# Optional env:
#   HAS_A2A             "true" to include the A2A interoperability card
#   HAS_A2A_SLIMRPC     "true" to include the A2A over SLIMRPC interoperability card
#   HAS_BENCHMARKS      "true" to include the SLIM benchmarks card
#   HAS_DIRECTORY       "true" to include the Directory conformance card
#   OUTPUT              destination path  (default: site/index.html)
#   INTEGRATIONS_RUN_ID run ID of the test-integrations workflow (footer link)
#   BENCHMARK_RUN_ID    run ID of the test-benchmarks-slim workflow (footer link)

set -euo pipefail

HAS_A2A="${HAS_A2A:-false}"
HAS_A2A_SLIMRPC="${HAS_A2A_SLIMRPC:-false}"
HAS_BENCHMARKS="${HAS_BENCHMARKS:-false}"
HAS_DIRECTORY="${HAS_DIRECTORY:-false}"
OUTPUT="${OUTPUT:-site/index.html}"
INTEGRATIONS_RUN_ID="${INTEGRATIONS_RUN_ID:-}"
BENCHMARK_RUN_ID="${BENCHMARK_RUN_ID:-}"

mkdir -p "$(dirname "$OUTPUT")"

cat > "$OUTPUT" <<'HTML'
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
      max-width: 900px;
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
    .card-grid { display: grid; gap: 18px; margin-top: 32px; }
    .report-card {
      display: block;
      text-decoration: none;
      color: inherit;
      background: rgba(255, 255, 255, 0.8);
      border: 1px solid var(--border);
      border-radius: 20px;
      padding: 24px;
      transition: transform 160ms ease, box-shadow 160ms ease, border-color 160ms ease;
    }
    .report-card:hover {
      transform: translateY(-2px);
      border-color: rgba(15, 118, 110, 0.34);
      box-shadow: 0 20px 40px rgba(15, 118, 110, 0.12);
    }
    .eyebrow {
      display: inline-block;
      font-size: 0.78rem;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: var(--accent);
      margin-bottom: 14px;
    }
    .report-card h2 { margin: 0 0 10px; font-size: 1.55rem; }
    .report-card span { color: var(--accent-strong); font-weight: 600; }
    footer { margin-top: 32px; font-size: 0.95rem; color: var(--muted); }
    @media (max-width: 640px) {
      body { padding: 20px 14px; }
      main { padding: 28px 20px; border-radius: 22px; }
    }
  </style>
</head>
<body>
  <main>
    <div class="eyebrow">GitHub Pages</div>
    <h1>CSIT Test Reports</h1>
    <p>Static report outputs from CI — A2A interoperability results and SLIM data-plane benchmark dashboards.</p>
    <div class="card-grid">
HTML

if [[ "$HAS_A2A" == "true" ]]; then
  cat >> "$OUTPUT" <<'HTML'
      <a class="report-card" href="./a2a/">
        <h2>A2A interoperability</h2>
        <p>Cross-SDK interoperability results with merged JSON, XML, and HTML dashboard output.</p>
        <span>Open report</span>
      </a>
HTML
fi

if [[ "$HAS_A2A_SLIMRPC" == "true" ]]; then
  cat >> "$OUTPUT" <<'HTML'
      <a class="report-card" href="./a2a-slimrpc/">
        <h2>A2A &mdash; SlimRPC interoperability</h2>
        <p>Cross-language A2A-over-SLIMRPC interoperability results with merged JSON, XML, and HTML dashboard output.</p>
        <span>Open report</span>
      </a>
HTML
fi

if [[ "$HAS_BENCHMARKS" == "true" ]]; then
  cat >> "$OUTPUT" <<'HTML'
      <a class="report-card" href="./benchmarks/slim/">
        <h2>SLIM benchmarks</h2>
        <p>Throughput and latency benchmark dashboards across modes, payload sizes, and sender counts.</p>
        <span>Open report</span>
      </a>
HTML
fi

if [[ "$HAS_DIRECTORY" == "true" ]]; then
  cat >> "$OUTPUT" <<'HTML'
      <a class="report-card" href="./directory/">
        <h2>Directory conformance</h2>
        <p>Client/server conformance results across supported Directory client and server versions.</p>
        <span>Open report</span>
      </a>
HTML
fi

# Build footer with links to source workflow runs.
footer_links=""
if [[ -n "$INTEGRATIONS_RUN_ID" ]]; then
  footer_links+="<a href=\"https://github.com/${GITHUB_REPOSITORY}/actions/runs/${INTEGRATIONS_RUN_ID}\">test-integrations #${INTEGRATIONS_RUN_ID}</a>"
fi
if [[ -n "$BENCHMARK_RUN_ID" ]]; then
  [[ -n "$footer_links" ]] && footer_links+=" &mdash; "
  footer_links+="<a href=\"https://github.com/${GITHUB_REPOSITORY}/actions/runs/${BENCHMARK_RUN_ID}\">test-benchmarks-slim #${BENCHMARK_RUN_ID}</a>"
fi
if [[ -z "$footer_links" && -n "${GITHUB_RUN_ID:-}" ]]; then
  footer_links="<a href=\"https://github.com/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}\">workflow run #${GITHUB_RUN_ID}</a>"
fi

cat >> "$OUTPUT" <<HTML
    </div>
    <footer>Sources: ${footer_links}</footer>
  </main>
</body>
</html>
HTML

touch "$(dirname "$OUTPUT")/.nojekyll"
echo "wrote $OUTPUT"
