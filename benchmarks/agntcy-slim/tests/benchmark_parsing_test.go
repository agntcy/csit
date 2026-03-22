package tests

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func parseSenderReport(report string) senderReport {
	lines := strings.Split(report, "\n")
	return senderReport{
		TotalMessages:  mustParseInt(extractMarkdownValue(lines, "- **Total Messages:** ")),
		ThroughputMPS:  mustParseReportFloat(extractThroughputValue(lines)),
		MeanLatencyMS:  mustParseDurationMS(extractLatencyTableValue(lines, "**Mean**")),
		P50LatencyMS:   mustParseDurationMS(extractLatencyTableValue(lines, "**P50 (Median)**")),
		P90LatencyMS:   mustParseDurationMS(extractLatencyTableValue(lines, "**P90**")),
		P99LatencyMS:   mustParseDurationMS(extractLatencyTableValue(lines, "**P99**")),
		MaxLatencyMS:   mustParseDurationMS(extractLatencyTableValue(lines, "**Max**")),
		RuntimeErrors:  mustParseInt(extractMarkdownValue(lines, "- **Runtime Errors:** ")),
		ActualDuration: extractMarkdownValue(lines, "- **Actual Duration:** "),
	}
}

func parseSinkStats(content string) sinkStats {
	values := parseKeyValueLines(content)
	return sinkStats{
		Mode:                 values["mode"],
		ReceivedMessages:     mustParseIntWithDefault(values["received_messages"], 0),
		ReceivedBytes:        mustParseIntWithDefault(values["received_bytes"], 0),
		ReplyMessages:        mustParseIntWithDefault(values["reply_messages"], 0),
		Errors:               mustParseIntWithDefault(values["errors"], 0),
		WarmupMessages:       mustParseIntWithDefault(values["warmup_messages"], 0),
		WarmupReplies:        mustParseIntWithDefault(values["warmup_replies"], 0),
		DrainMessages:        mustParseIntWithDefault(values["drain_messages"], 0),
		DrainReplies:         mustParseIntWithDefault(values["drain_replies"], 0),
		ElapsedSeconds:       mustParseFloatWithDefault(values["elapsed_seconds"], 0),
		ActiveReceiveSeconds: mustParseFloatWithDefault(values["active_receive_seconds"], 0),
		ReceiveMPS:           mustParseFloatWithDefault(values["receive_mps"], 0),
		ReceiveMBPS:          mustParseFloatWithDefault(values["receive_mbps"], 0),
		ActiveReceiveMPS:     mustParseFloatWithDefault(values["active_receive_mps"], 0),
		ActiveReceiveMBPS:    mustParseFloatWithDefault(values["active_receive_mbps"], 0),
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
	ginkgo.Fail(fmt.Sprintf("missing report field %q", prefix))
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
	ginkgo.Fail(fmt.Sprintf("missing latency metric %q", metric))
	return ""
}

func envStringList(key string, defaults []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]string(nil), defaults...)
	}
	parts := strings.Fields(value)
	gomega.Expect(parts).NotTo(gomega.BeEmpty(), "invalid %s value", key)
	return parts
}

func envIntList(key string, defaults []int) []int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]int(nil), defaults...)
	}
	return mustParseIntList(value)
}

func envBool(key string, defaultValue bool) bool {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		default:
			ginkgo.Fail(fmt.Sprintf("invalid %s value %q", key, value))
		}
	}
	return defaultValue
}

func envFloat(key string, defaultValue float64) float64 {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid %s value %q", key, value)
		return parsed
	}
	return defaultValue
}

func mustParseIntList(value string) []int {
	fields := strings.Fields(value)
	gomega.Expect(fields).NotTo(gomega.BeEmpty())
	parsed := make([]int, 0, len(fields))
	for _, field := range fields {
		current, err := strconv.Atoi(field)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid integer list value %q", field)
		parsed = append(parsed, current)
	}
	return parsed
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

func mustParseFloatWithDefault(value string, defaultValue float64) float64 {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return mustParseReportFloat(value)
}

func joinInts(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, " ")
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func defaultIfEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func maxFloat(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
