#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORTS_DIR="${REPORTS_DIR:-$BASE_DIR/reports}"
OUTPUT_DIR="${OUTPUT_DIR:-$BASE_DIR/published/evidence}"

# shellcheck source=evidence-lib.sh
source "$BASE_DIR/scripts/evidence-lib.sh"

json_file="$(evidence_json)"
md_file="${OUTPUT_DIR}/c1-evidence.md"
html_file="${OUTPUT_DIR}/c1-evidence.html"

if [[ ! -f "$json_file" ]]; then
  echo "evidence JSON not found: $json_file" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to render evidence reports" >&2
  exit 1
fi

mkdir -p "$OUTPUT_DIR"

schema_version="$(jq -r '.schema_version' "$json_file")"
generated_at="$(jq -r '.generated_at' "$json_file")"
source_path="$(jq -r '.source' "$json_file")"

render_markdown_case() {
  local case_json="$1"
  local row_id mode status use_case
  row_id="$(jq -r '.row_id' <<<"$case_json")"
  mode="$(jq -r '.mode' <<<"$case_json")"
  status="$(jq -r '.status' <<<"$case_json")"
  use_case="$(jq -r '.use_case' <<<"$case_json")"

  {
    echo "### ${row_id} — ${status}"
    echo
    echo "- **Mode:** \`${mode}\`"
    echo "- **Use case:** ${use_case}"
    jq -r '
      "- **Sender messages:** \(.sender_messages)",
      "- **Sender errors:** \(.sender_errors)",
      (if .sink_received != null then "- **Sink received:** \(.sink_received)" else empty end),
      (if .sink_replies != null then "- **Sink replies:** \(.sink_replies)" else empty end),
      (if .sink_errors != null then "- **Sink errors:** \(.sink_errors)" else empty end),
      (if .mean_latency_ms != null then "- **Mean latency (ms):** \(.mean_latency_ms)" else empty end)
    ' <<<"$case_json"
    echo
    echo "**Assertions:**"
    jq -r '.assertions[] | "- \(.)"' <<<"$case_json"
    echo
  }
}

render_html_case() {
  local case_json="$1"
  local row_id mode status use_case status_class status_label_text
  row_id="$(jq -r '.row_id' <<<"$case_json")"
  mode="$(jq -r '.mode' <<<"$case_json")"
  status="$(jq -r '.status' <<<"$case_json")"
  use_case="$(html_escape "$(jq -r '.use_case' <<<"$case_json")")"
  status_class="$(status_css_class "$status")"
  status_label_text="$(status_label "$status")"

  local metrics_html assertions_html
  metrics_html="$(jq -r '
    "<tr><th>Sender messages</th><td>\(.sender_messages)</td></tr>",
    "<tr><th>Sender errors</th><td>\(.sender_errors)</td></tr>",
    (if .sink_received != null then "<tr><th>Sink received</th><td>\(.sink_received)</td></tr>" else empty end),
    (if .sink_replies != null then "<tr><th>Sink replies</th><td>\(.sink_replies)</td></tr>" else empty end),
    (if .sink_errors != null then "<tr><th>Sink errors</th><td>\(.sink_errors)</td></tr>" else empty end),
    (if .mean_latency_ms != null then "<tr><th>Mean latency (ms)</th><td>\(.mean_latency_ms)</td></tr>" else empty end)
  ' <<<"$case_json")"
  assertions_html=""
  while IFS= read -r assertion; do
    [[ -z "$assertion" ]] && continue
    assertions_html+="<li>$(html_escape "$assertion")</li>"
  done < <(jq -r '.assertions[]' <<<"$case_json")

  cat <<HTML
    <article class="case-card" id="${row_id}">
      <div class="case-head">
        <h2>${row_id}</h2>
        <span class="status ${status_class}">${status_label_text}</span>
      </div>
      <p class="use-case">${use_case}</p>
      <p class="mode">Mode: <code>${mode}</code></p>
      <table class="metrics">
        <tbody>
          ${metrics_html}
        </tbody>
      </table>
      <h3>Assertions</h3>
      <ul class="assertions">
        ${assertions_html}
      </ul>
    </article>
HTML
}

{
  echo "# C1 Evidence Report"
  echo
  echo "| Field | Value |"
  echo "|-------|-------|"
  echo "| Schema version | \`${schema_version}\` |"
  echo "| Generated at | ${generated_at} |"
  echo "| Source | \`${source_path}\` |"
  echo "| Machine-readable | [c1-evidence.json](./c1-evidence.json) |"
  echo "| HTML | [c1-evidence.html](./c1-evidence.html) |"
  echo
  echo "## Use cases"
  echo
  while IFS= read -r case_json; do
    render_markdown_case "$case_json"
  done < <(jq -c '.cases[]' "$json_file")
} > "$md_file"

{
  cat <<'HTML_HEAD'
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>C1 Evidence Report</title>
  <style>
    :root {
      --bg: #f4f6f8; --panel: #fff; --text: #1a2332; --muted: #5a6b7d;
      --accent: #0d6e6e; --border: #dde3ea;
      --ok: #1d6b3a; --ok-bg: #e8f5ec; --fail: #9b2c2c; --fail-bg: #fdecec;
      --unknown: #5a6b7d; --unknown-bg: #eef1f5;
    }
    * { box-sizing: border-box; }
    body { margin: 0; font-family: system-ui, sans-serif; color: var(--text); background: var(--bg); padding: 2rem 1rem 3rem; }
    main { max-width: 820px; margin: 0 auto; }
    h1 { margin: 0 0 1rem; }
    .meta { background: var(--panel); border: 1px solid var(--border); border-radius: 10px; padding: 1rem 1.25rem; margin-bottom: 1.5rem; }
    .meta dl { margin: 0; display: grid; grid-template-columns: 10rem 1fr; gap: .35rem .75rem; font-size: .95rem; }
    .meta dt { color: var(--muted); font-weight: 600; }
    .meta a { color: var(--accent); }
    .case-card { background: var(--panel); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem 1.5rem; margin-bottom: 1rem; }
    .case-head { display: flex; flex-wrap: wrap; align-items: center; gap: .75rem; margin-bottom: .5rem; }
    .case-head h2 { margin: 0; font-size: 1.15rem; }
    .use-case { margin: 0 0 .5rem; color: var(--text); }
    .mode { margin: 0 0 1rem; color: var(--muted); font-size: .9rem; }
    .metrics { width: 100%; border-collapse: collapse; margin-bottom: 1rem; font-size: .9rem; }
    .metrics th, .metrics td { text-align: left; padding: .4rem .5rem; border-bottom: 1px solid var(--border); }
    .metrics th { width: 40%; color: var(--muted); font-weight: 600; }
    .assertions { margin: 0; padding-left: 1.2rem; color: var(--text); }
    .status { display: inline-block; font-size: .8125rem; font-weight: 600; padding: .2rem .55rem; border-radius: 6px; }
    .status-verified { background: var(--ok-bg); color: var(--ok); }
    .status-failed { background: var(--fail-bg); color: var(--fail); }
    .status-unknown { background: var(--unknown-bg); color: var(--unknown); }
  </style>
</head>
<body>
  <main>
    <h1>C1 Evidence Report</h1>
HTML_HEAD

  echo '    <section class="meta"><dl>'
  echo "      <dt>Schema version</dt><dd>${schema_version}</dd>"
  echo "      <dt>Generated at</dt><dd>${generated_at}</dd>"
  echo "      <dt>Source</dt><dd><code>$(html_escape "$source_path")</code></dd>"
  echo '      <dt>Formats</dt><dd><a href="./c1-evidence.json">JSON</a> · <a href="./c1-evidence.md">Markdown</a></dd>'
  echo '    </dl></section>'

  while IFS= read -r case_json; do
    render_html_case "$case_json"
  done < <(jq -c '.cases[]' "$json_file")

  cat <<'HTML_FOOT'
  </main>
</body>
</html>
HTML_FOOT
} > "$html_file"

echo "Rendered $md_file"
echo "Rendered $html_file"
