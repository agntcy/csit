// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"fmt"
	"strings"
)

func ConvertTestReportToHTML(report map[string]string, runID string) string {
	html := "<h1>Test Report</h1>"

	html += "<h2>Link</h2>"
	link := fmt.Sprintf("<a href=\"%s\" target=\"_blank\">%s</a>", "https://github.com/cisco-eti/phoenix-csit/actions/runs/"+runID, "GitHub Actions Run")
	html += link

	// Add metadata to the report
	html += "<h2>Metadata</h2>"
	table := fmt.Sprintf("<table><tr>  <th>Test Property</th>  <th>Value</th></tr>")
	for key, value := range report {
		// If key starts with "test", add to table
		if strings.HasPrefix(key, "test") {
			table += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", key, value)
		}
	}
	table += "</table>"
	html += table

	html += "<h2>Inputs</h2>"
	table = fmt.Sprintf("<table><tr>  <th>Input</th>  <th>Value</th></tr>")
	for key, value := range report {
		// If key starts with "input", add to table
		if strings.HasPrefix(key, "input") {
			table += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", key, value)
		}
	}
	table += "</table>"
	html += table

	html += "<h2>Outputs</h2>"
	table = fmt.Sprintf("<table><tr>  <th>Output</th>  <th>Value</th></tr>")
	for key, value := range report {
		// If key starts with "output", add to table
		if strings.HasPrefix(key, "output") {
			table += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", key, value)
		}
	}
	table += "</table>"
	html += table

	html += "<h2>Misceallaneous</h2>"
	table = fmt.Sprintf("<table><tr>  <th>Property</th>  <th>Value</th></tr>")
	for key, value := range report {
		// If key does not start with "test", "input", or "output", add to table
		if !strings.HasPrefix(key, "test") && !strings.HasPrefix(key, "input") && !strings.HasPrefix(key, "output") {
			table += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", key, value)
		}
	}
	table += "</table>"
	html += table

	return html
}
