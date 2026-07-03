// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/agntcy/csit/analitics/harness"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	c1EvidenceMessageCount = 20
	c1EvidencePayloadSize  = 16
	c1EvidenceClientRate   = 10
)

type c1EvidenceScenario struct {
	RowID       string
	Mode        string
	UseCase     string
	Responder   string
	CollectSink bool
	assert      func(sender harness.SenderReport, sink harness.SinkStats)
}

var slimRuntime *harness.Runtime

var _ = ginkgo.BeforeSuite(func() {
	slimRuntime = harness.New("")
	slimRuntime.InitBuildArtifacts()
})

var _ = ginkgo.AfterSuite(func() {
	if slimRuntime != nil {
		slimRuntime.Cleanup()
	}
})

var _ = ginkgo.Describe("C1 centralized use-case evidence", ginkgo.Label("c1-evidence"), func() {
	ginkgo.BeforeEach(func() {
		slimRuntime.StartStack()
		slimRuntime.StopResponder()
	})

	ginkgo.AfterEach(func() {
		slimRuntime.StopStack()
	})

	ginkgo.DescribeTable("proves C1 messaging use cases with behavioral assertions",
		func(scenario c1EvidenceScenario) {
			reportPath := filepath.Join(slimRuntime.BuildDir, fmt.Sprintf("c1-%s-report.md", scenario.Mode))
			statsPath := filepath.Join(slimRuntime.BuildDir, fmt.Sprintf("c1-%s-sink.stats", scenario.Mode))

			if scenario.Responder != "" {
				slimRuntime.StartResponder(scenario.Responder, 1, statsPath)
			}

			ginkgo.By(fmt.Sprintf("exercising %s (%s)", scenario.UseCase, scenario.Mode))
			session := runC1RateClient(scenario.Mode, reportPath)
			gomega.Expect(session.ExitCode()).To(gomega.Equal(0), "rate-client failed for mode %s", scenario.Mode)

			time.Sleep(300 * time.Millisecond)

			reportContent, err := os.ReadFile(reportPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			sender := harness.ParseSenderReport(string(reportContent))

			sink := harness.SinkStats{}
			if scenario.CollectSink {
				statsContent, readErr := os.ReadFile(statsPath)
				gomega.Expect(readErr).NotTo(gomega.HaveOccurred())
				sink = harness.ParseSinkStats(string(statsContent))
			}

			ginkgo.By("asserting sender and sink behavior")
			scenario.assert(sender, sink)

			row := c1EvidenceCase{
				RowID:          scenario.RowID,
				Mode:           scenario.Mode,
				UseCase:        scenario.UseCase,
				Status:         "verified",
				SenderMessages: sender.TotalMessages,
				SenderErrors:   sender.RuntimeErrors,
				SinkReceived:   sink.ReceivedMessages,
				SinkReplies:    sink.ReplyMessages,
				SinkErrors:     sink.Errors,
				MeanLatencyMS:  sender.MeanLatencyMS,
				Assertions:     c1EvidenceAssertions(scenario, sender, sink),
			}

			gomega.Expect(upsertC1EvidenceCase(c1EvidenceReportPath(), row)).To(gomega.Succeed())
			logC1EvidenceSummary(row)
			ginkgo.AddReportEntry("C1 Evidence Row", row.RowID+"="+row.Status)
		},
		ginkgo.Entry("c1-request-reply", c1EvidenceScenario{
			RowID:       "c1-request-reply",
			Mode:        "request-reply",
			UseCase:     "Agent A calls B and waits for a reply",
			Responder:   "echo",
			CollectSink: true,
			assert: func(sender harness.SenderReport, sink harness.SinkStats) {
				gomega.Expect(sender.TotalMessages).To(gomega.Equal(int64(c1EvidenceMessageCount)))
				gomega.Expect(sender.RuntimeErrors).To(gomega.BeZero())
				gomega.Expect(sender.MeanLatencyMS).To(gomega.BeNumerically(">", 0))
				gomega.Expect(sink.ReceivedMessages).To(gomega.Equal(int64(c1EvidenceMessageCount)))
				gomega.Expect(sink.ReplyMessages).To(gomega.Equal(int64(c1EvidenceMessageCount)))
				gomega.Expect(sink.Errors).To(gomega.BeZero())
			},
		}),
		ginkgo.Entry("c1-fire-and-forget", c1EvidenceScenario{
			RowID:       "c1-fire-and-forget",
			Mode:        "fire-and-forget",
			UseCase:     "Agent fires an event; consumer handles async",
			Responder:   "sink",
			CollectSink: true,
			assert: func(sender harness.SenderReport, sink harness.SinkStats) {
				gomega.Expect(sender.TotalMessages).To(gomega.Equal(int64(c1EvidenceMessageCount)))
				gomega.Expect(sender.RuntimeErrors).To(gomega.BeZero())
				gomega.Expect(sink.ReceivedMessages).To(gomega.Equal(int64(c1EvidenceMessageCount)))
				gomega.Expect(sink.Errors).To(gomega.BeZero())
			},
		}),
		ginkgo.Entry("c1-write", c1EvidenceScenario{
			RowID:       "c1-write",
			Mode:        "write",
			UseCase:     "Publish into the mesh without a paired responder",
			Responder:   "",
			CollectSink: false,
			assert: func(sender harness.SenderReport, sink harness.SinkStats) {
				gomega.Expect(sender.TotalMessages).To(gomega.Equal(int64(c1EvidenceMessageCount)))
				gomega.Expect(sender.RuntimeErrors).To(gomega.BeZero())
				gomega.Expect(sender.ThroughputMPS).To(gomega.BeNumerically(">", 0))
				_ = sink
			},
		}),
	)
})

func runC1RateClient(mode string, reportPath string) *gexec.Session {
	return slimRuntime.RunRateClient([]string{
		"-mode", mode,
		"-clients", "1",
		"-msgs", strconv.Itoa(c1EvidenceMessageCount),
		"-rate", strconv.Itoa(c1EvidenceClientRate),
		"-size", strconv.Itoa(c1EvidencePayloadSize),
		"-local", "agntcy/demo/client",
		"-server", slimRuntime.ServerEndpoint,
		"-dest", "agntcy/demo/echo",
		"-output", reportPath,
	}, 45*time.Second)
}

func c1EvidenceAssertions(scenario c1EvidenceScenario, sender harness.SenderReport, sink harness.SinkStats) []string {
	assertions := []string{
		fmt.Sprintf("sender completed %d messages with %d runtime errors", sender.TotalMessages, sender.RuntimeErrors),
	}
	switch scenario.Mode {
	case "request-reply":
		assertions = append(assertions,
			fmt.Sprintf("round-trip mean latency %.3f ms", sender.MeanLatencyMS),
			fmt.Sprintf("sink received %d messages and replied %d times with %d errors", sink.ReceivedMessages, sink.ReplyMessages, sink.Errors),
		)
	case "fire-and-forget":
		assertions = append(assertions,
			fmt.Sprintf("sink received %d async messages with %d errors", sink.ReceivedMessages, sink.Errors),
		)
	case "write":
		assertions = append(assertions,
			fmt.Sprintf("sender wrote at %.2f msg/sec without a bound responder", sender.ThroughputMPS),
		)
	}
	return assertions
}
