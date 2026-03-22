package tests

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega"
)

type benchmarkModeSpec struct {
	Name                    string
	Title                   string
	ResponderKind           string
	ObservedThroughputLabel string
	UsesSinkMetrics         bool
}

var benchmarkModeOrder = []string{"request-reply", "fire-and-forget", "write"}

var benchmarkModeSpecs = map[string]benchmarkModeSpec{
	"request-reply": {
		Name:                    "request-reply",
		Title:                   "Request-Reply",
		ResponderKind:           "echo",
		ObservedThroughputLabel: "Observed Node Throughput",
		UsesSinkMetrics:         true,
	},
	"fire-and-forget": {
		Name:                    "fire-and-forget",
		Title:                   "Fire-And-Forget",
		ResponderKind:           "sink",
		ObservedThroughputLabel: "Observed Node Throughput",
		UsesSinkMetrics:         true,
	},
	"write": {
		Name:                    "write",
		Title:                   "Write",
		ResponderKind:           "blackhole",
		ObservedThroughputLabel: "Sender Write Throughput",
		UsesSinkMetrics:         false,
	},
}

func canonicalBenchmarkMode(mode string) (string, error) {
	canonical := strings.TrimSpace(strings.ToLower(mode))
	if _, ok := benchmarkModeSpecs[canonical]; !ok {
		return "", fmt.Errorf("unsupported mode %q", mode)
	}
	return canonical, nil
}

func normalizeBenchmarkModes(modes []string) []string {
	normalized := make([]string, 0, len(modes))
	for _, mode := range modes {
		canonical, err := canonicalBenchmarkMode(mode)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		normalized = append(normalized, canonical)
	}
	return normalized
}

func benchmarkMode(mode string) benchmarkModeSpec {
	canonical, err := canonicalBenchmarkMode(mode)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return benchmarkModeSpecs[canonical]
}

func modeUsesResponder(mode string) bool {
	return benchmarkMode(mode).ResponderKind != ""
}

func modeUsesSinkMetrics(mode string) bool {
	return benchmarkMode(mode).UsesSinkMetrics
}

func modeResponderKind(mode string) string {
	return benchmarkMode(mode).ResponderKind
}

func modeRateValues(cfg suiteConfig, mode string, clients int, size int) []int {
	spec := benchmarkMode(mode)
	switch spec.Name {
	case "request-reply":
		return cfg.RequestRates
	case "write":
		if len(cfg.WriteRates) > 0 {
			return cfg.WriteRates
		}
	}
	return pubRatesForCase(cfg, clients, size)
}

func pubRatesForCase(cfg suiteConfig, clients int, size int) []int {
	if !cfg.PubRateAutoProfile {
		return cfg.PubRates
	}
	return []int{pubRateForCase(clients, size)}
}

func pubRateForCase(clients int, size int) int {
	if size >= 10240 {
		switch {
		case clients >= 50:
			return 100
		case clients >= 10:
			return 200
		default:
			return 500
		}
	}
	if size >= 1024 {
		switch {
		case clients >= 50:
			return 250
		case clients >= 10:
			return 500
		default:
			return 1000
		}
	}
	if clients >= 50 {
		return 500
	}
	return 1000
}

func modeObservedThroughputLabel(mode string) string {
	return benchmarkMode(mode).ObservedThroughputLabel
}

func modeDisplayTitle(mode string) string {
	return benchmarkMode(mode).Title
}

func benchmarkObservedMPS(row benchmarkRunResult) float64 {
	if row.Mode == "write" {
		return row.SenderMPS
	}
	return row.SinkActiveReceiveMPS
}

func benchmarkObservedMessages(row benchmarkRunResult) float64 {
	if row.Mode == "write" {
		return float64(row.SenderTotalMessages)
	}
	return float64(row.SinkReceivedMessages)
}
