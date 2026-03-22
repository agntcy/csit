package tests

import (
	"fmt"
	"os"
	"time"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func appendSinkSummary(reportFile string, statsFile string, includeMetrics bool) {
	if !includeMetrics {
		file, err := os.OpenFile(reportFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		defer file.Close()
		_, writeErr := fmt.Fprintln(file, "\n## Responder Summary\n- **Responder Metrics:** not collected for write mode (blackhole peer only)")
		gomega.Expect(writeErr).NotTo(gomega.HaveOccurred())
		return
	}

	statsContent, err := os.ReadFile(statsFile)
	if err != nil {
		file, createErr := os.OpenFile(reportFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		gomega.Expect(createErr).NotTo(gomega.HaveOccurred())
		defer file.Close()
		_, writeErr := fmt.Fprintln(file, "\n## Sink Summary\n- **Sink Stats:** unavailable")
		gomega.Expect(writeErr).NotTo(gomega.HaveOccurred())
		return
	}

	stats := parseSinkStats(string(statsContent))
	file, openErr := os.OpenFile(reportFile, os.O_APPEND|os.O_WRONLY, 0644)
	gomega.Expect(openErr).NotTo(gomega.HaveOccurred())
	defer file.Close()

	_, writeErr := fmt.Fprintf(file, `

## Sink Summary
| Parameter | Value |
| :--- | :--- |
| **Mode** | %s |
| **Received Messages** | %d |
| **Received Bytes** | %d |
| **Reply Messages** | %d |
| **Errors** | %d |
| **Warmup Messages** | %d |
| **Warmup Replies** | %d |
| **Drain Messages** | %d |
| **Drain Replies** | %d |
| **Elapsed Seconds** | %s |
| **Active Receive Seconds** | %s |
| **Receive Throughput** | %s msg/sec |
| **Receive Bandwidth** | %s MB/sec |
| **Active Receive Throughput** | %s msg/sec |
| **Active Receive Bandwidth** | %s MB/sec |
`,
		defaultIfEmpty(stats.Mode, "unknown"),
		stats.ReceivedMessages,
		stats.ReceivedBytes,
		stats.ReplyMessages,
		stats.Errors,
		stats.WarmupMessages,
		stats.WarmupReplies,
		stats.DrainMessages,
		stats.DrainReplies,
		formatFloat(stats.ElapsedSeconds),
		formatFloat(stats.ActiveReceiveSeconds),
		formatFloat(stats.ReceiveMPS),
		formatFloat(stats.ReceiveMBPS),
		formatFloat(stats.ActiveReceiveMPS),
		formatFloat(stats.ActiveReceiveMBPS),
	)
	gomega.Expect(writeErr).NotTo(gomega.HaveOccurred())
}

func collectProcessCPUUsage(rateSession *gexec.Session, runElapsed time.Duration, slimCPUStart float64, echoCPUStart float64, responderEnabled bool) processCPUUsage {
	senderCPUSeconds := 0.0
	if rateSession != nil && rateSession.Command != nil && rateSession.Command.ProcessState != nil {
		senderCPUSeconds = rateSession.Command.ProcessState.UserTime().Seconds() + rateSession.Command.ProcessState.SystemTime().Seconds()
	}

	echoCPUEnd := 0.0
	if responderEnabled {
		var err error
		echoCPUEnd, err = readSessionCPUSeconds(echoSession)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}
	slimCPUEnd, err := readSessionCPUSeconds(slimSession)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	responderCPUSeconds := maxFloat(0, echoCPUEnd-echoCPUStart)
	nodeCPUSeconds := maxFloat(0, slimCPUEnd-slimCPUStart)
	elapsedSeconds := runElapsed.Seconds()
	if elapsedSeconds <= 0 {
		elapsedSeconds = 1e-9
	}

	totalCPUSeconds := senderCPUSeconds + responderCPUSeconds + nodeCPUSeconds
	return processCPUUsage{
		SenderCPUSeconds:    senderCPUSeconds,
		SenderCPUPercent:    100 * senderCPUSeconds / elapsedSeconds,
		ResponderCPUSeconds: responderCPUSeconds,
		ResponderCPUPercent: 100 * responderCPUSeconds / elapsedSeconds,
		NodeCPUSeconds:      nodeCPUSeconds,
		NodeCPUPercent:      100 * nodeCPUSeconds / elapsedSeconds,
		TotalCPUSeconds:     totalCPUSeconds,
		TotalCPUPercent:     100 * totalCPUSeconds / elapsedSeconds,
	}
}

func appendProcessCPUSummary(reportFile string, usage processCPUUsage) {
	file, err := os.OpenFile(reportFile, os.O_APPEND|os.O_WRONLY, 0644)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer file.Close()

	_, writeErr := fmt.Fprintf(file, `

## Process CPU Summary
| Process | CPU Time (sec) | Avg CPU %% |
| :--- | :--- | :--- |
| **Sender (tests/rate-client)** | %s | %s |
| **Responder (tests/echo-client)** | %s | %s |
| **Node (slimctl)** | %s | %s |
| **Total** | %s | %s |
`,
		formatFloat(usage.SenderCPUSeconds),
		formatFloat(usage.SenderCPUPercent),
		formatFloat(usage.ResponderCPUSeconds),
		formatFloat(usage.ResponderCPUPercent),
		formatFloat(usage.NodeCPUSeconds),
		formatFloat(usage.NodeCPUPercent),
		formatFloat(usage.TotalCPUSeconds),
		formatFloat(usage.TotalCPUPercent),
	)
	gomega.Expect(writeErr).NotTo(gomega.HaveOccurred())
}
