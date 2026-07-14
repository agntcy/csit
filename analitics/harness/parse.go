// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package harness

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/gomega"
)

// ParseSenderReport extracts C1 evidence fields from a rate-client markdown report.
func ParseSenderReport(report string) SenderReport {
	lines := strings.Split(report, "\n")
	return SenderReport{
		TotalMessages: mustParseInt(extractMarkdownValue(lines, "- **Total Messages:** ")),
		ThroughputMPS: mustParseReportFloat(extractThroughputValue(lines)),
		MeanLatencyMS: mustParseDurationMS(extractLatencyTableValue(lines, "**Mean**")),
		RuntimeErrors: mustParseInt(extractMarkdownValue(lines, "- **Runtime Errors:** ")),
	}
}

// ParseSinkStats parses echo-client stats used by C1 evidence assertions.
func ParseSinkStats(content string) SinkStats {
	values := parseKeyValueLines(content)
	return SinkStats{
		ReceivedMessages: mustParseIntWithDefault(values["received_messages"], 0),
		ReplyMessages:    mustParseIntWithDefault(values["reply_messages"], 0),
		Errors:           mustParseIntWithDefault(values["errors"], 0),
	}
}

func parseKeyValueLines(content string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		values[parts[0]] = parts[1]
	}
	return values
}

func extractMarkdownValue(lines []string, prefix string) string {
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	gomega.Expect(fmt.Errorf("missing report field %q", prefix)).NotTo(gomega.HaveOccurred())
	return ""
}

func extractThroughputValue(lines []string) string {
	value := extractMarkdownValue(lines, "- **Throughput:** ")
	re := regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?) msg/sec`)
	matches := re.FindStringSubmatch(value)
	gomega.Expect(matches).To(gomega.HaveLen(2), "expected throughput line to contain msg/sec value")
	return matches[1]
}

func extractLatencyTableValue(lines []string, metric string) string {
	prefix := fmt.Sprintf("| %s |", metric)
	for _, line := range lines {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		parts := strings.Split(line, "|")
		gomega.Expect(parts).To(gomega.HaveLen(4), "expected latency table row for %s", metric)
		return strings.TrimSpace(parts[2])
	}
	gomega.Expect(fmt.Errorf("missing latency metric %q", metric)).NotTo(gomega.HaveOccurred())
	return ""
}

func mustParseInt(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid integer value %q", value)
	return parsed
}

func mustParseIntWithDefault(value string, defaultValue int64) int64 {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return mustParseInt(value)
}

func mustParseReportFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid float value %q", value)
	return parsed
}

func mustParseDurationMS(value string) float64 {
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid duration value %q", value)
	return parsed.Seconds() * 1000
}
