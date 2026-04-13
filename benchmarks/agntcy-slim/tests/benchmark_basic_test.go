// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

// Basic throughput benchmark reproducing the upstream agntcy/slim stress_publish baseline.
//
// Topology: N senders (rate-client in fire-and-forget mode, -rate 0 = unlimited) publishing to
// a single echo-client sink, routed through a local slimctl SLIM node.
//
// CSV output schema (matches upstream agntcy/slim benchmark-results.csv):
//   senders, messages_per_sender, payload_bytes, total_messages,
//   run, send_elapsed_s, total_elapsed_s, total_received, send_mps, recv_mps
//
// Activate with: SLIM_RUN_BASIC_BENCHMARK=1

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type basicBenchConfig struct {
	Senders       []int
	PayloadSizes  []int
	MsgsPerSender int
	Runs          int
	ReportDir     string
	ResultsCSV    string
}

type basicBenchRow struct {
	Senders       int
	MsgsPerSender int
	PayloadBytes  int
	TotalMessages int64
	Run           int
	SendElapsedS  float64
	TotalElapsedS float64
	TotalReceived int64
	SendMPS       float64
	RecvMPS       float64
}

var _ = ginkgo.Describe("SLIM Basic Throughput Benchmark", ginkgo.Label("basic-benchmark"), func() {
	ginkgo.It("runs the basic throughput benchmark and generates a CSV report", func() {
		if envString("SLIM_RUN_BASIC_BENCHMARK", "") == "" {
			ginkgo.Skip("set SLIM_RUN_BASIC_BENCHMARK=1 to run the basic throughput benchmark")
		}

		cfg := loadBasicBenchConfig()

		ginkgo.By("resetting basic benchmark report artifacts")
		gomega.Expect(os.MkdirAll(cfg.ReportDir, 0o755)).To(gomega.Succeed())
		_ = os.Remove(cfg.ResultsCSV)

		ginkgo.By("starting the local SLIM stack for the basic benchmark")
		startLocalSlimStack()
		defer stopLocalSlimStack()
		stopEchoResponder()

		ginkgo.By("running the basic benchmark matrix")
		rows := runBasicBenchMatrix(cfg)

		ginkgo.By("writing results CSV")
		writeBasicBenchCSV(cfg.ResultsCSV, rows)

		gomega.Expect(cfg.ResultsCSV).To(gomega.BeAnExistingFile())
		ginkgo.AddReportEntry("Basic Benchmark Results CSV", cfg.ResultsCSV)
	})
})

func loadBasicBenchConfig() basicBenchConfig {
	reportDir := envString("BASIC_BENCH_REPORT_DIR", "./reports")
	return basicBenchConfig{
		Senders:       envIntList("BASIC_BENCH_SENDERS", []int{1, 2, 4, 8}),
		PayloadSizes:  envIntList("BASIC_BENCH_PAYLOAD_SIZES", []int{8, 16, 64, 256, 1024, 4096}),
		MsgsPerSender: envInt("BASIC_BENCH_MSGS_PER_SENDER", 100000),
		Runs:          envInt("BASIC_BENCH_RUNS", 3),
		ReportDir:     reportDir,
		ResultsCSV:    filepath.Join(reportDir, "basic-benchmark-results.csv"),
	}
}

func runBasicBenchMatrix(cfg basicBenchConfig) []basicBenchRow {
	rows := make([]basicBenchRow, 0, len(cfg.Senders)*len(cfg.PayloadSizes)*cfg.Runs)
	for _, senders := range cfg.Senders {
		for _, payload := range cfg.PayloadSizes {
			for run := 1; run <= cfg.Runs; run++ {
				ginkgo.By(fmt.Sprintf("basic benchmark: senders=%d payload=%d bytes run=%d/%d",
					senders, payload, run, cfg.Runs))
				row := runBasicBenchCase(senders, payload, run, cfg.MsgsPerSender)
				logBasicBenchRow(row)
				rows = append(rows, row)
			}
		}
	}
	return rows
}

func runBasicBenchCase(senders, payloadBytes, run, msgsPerSender int) basicBenchRow {
	totalMsgs := senders * msgsPerSender

	statsFile, err := os.CreateTemp(buildDir, "basic-bench-sink-*.txt")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	statsPath := statsFile.Name()
	gomega.Expect(statsFile.Close()).To(gomega.Succeed())

	reportFile, err := os.CreateTemp(buildDir, "basic-bench-sender-*.md")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	reportPath := reportFile.Name()
	gomega.Expect(reportFile.Close()).To(gomega.Succeed())

	// Restart sink fresh for each case so elapsed_seconds is scoped to this run.
	stopEchoResponder()
	startEchoResponder("sink", 1, statsPath)

	runStart := time.Now()

	benchCmd := exec.Command(
		rateClientPath,
		"-mode", "fire-and-forget",
		"-clients", strconv.Itoa(senders),
		"-msgs", strconv.Itoa(totalMsgs),
		"-rate", "0",
		"-size", strconv.Itoa(payloadBytes),
		"-local", "agntcy/demo/client",
		"-server", serverEndpoint,
		"-dest", "agntcy/demo/echo",
		"-output", reportPath,
	)
	session, startErr := gexec.Start(benchCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(startErr).NotTo(gomega.HaveOccurred())

	// Generous timeout scaled to message count: at least 60s + 1s per 10K messages.
	timeout := 60*time.Second + time.Duration(totalMsgs/10000)*time.Second
	gomega.Eventually(session, timeout).Should(gexec.Exit(0),
		"rate-client failed for senders=%d payload=%d run=%d", senders, payloadBytes, run)

	// Brief drain window to let the sink receive any in-flight messages.
	time.Sleep(500 * time.Millisecond)
	stopEchoResponder()
	totalElapsedS := time.Since(runStart).Seconds()

	reportContent, readErr := os.ReadFile(reportPath)
	gomega.Expect(readErr).NotTo(gomega.HaveOccurred())
	sender := parseSenderReport(string(reportContent))

	sinkContent, sinkReadErr := os.ReadFile(statsPath)
	gomega.Expect(sinkReadErr).NotTo(gomega.HaveOccurred())
	sink := parseSinkStats(string(sinkContent))

	// Prefer the duration reported by rate-client; fall back to wall time.
	sendElapsedS := time.Since(runStart).Seconds()
	if d, parseErr := time.ParseDuration(sender.ActualDuration); parseErr == nil && d > 0 {
		sendElapsedS = d.Seconds()
	}

	sendMPS := 0.0
	if sendElapsedS > 0 && sender.TotalMessages > 0 {
		sendMPS = float64(sender.TotalMessages) / sendElapsedS
	}

	recvMPS := 0.0
	if totalElapsedS > 0 && sink.ReceivedMessages > 0 {
		recvMPS = float64(sink.ReceivedMessages) / totalElapsedS
	}

	return basicBenchRow{
		Senders:       senders,
		MsgsPerSender: msgsPerSender,
		PayloadBytes:  payloadBytes,
		TotalMessages: sender.TotalMessages,
		Run:           run,
		SendElapsedS:  sendElapsedS,
		TotalElapsedS: totalElapsedS,
		TotalReceived: sink.ReceivedMessages,
		SendMPS:       sendMPS,
		RecvMPS:       recvMPS,
	}
}

func logBasicBenchRow(row basicBenchRow) {
	_ = writeProgressLine(
		"BASIC_BENCH_RESULT senders=%d msgs_per_sender=%d payload=%d run=%d "+
			"total_messages=%d send_elapsed_s=%.3f total_elapsed_s=%.3f "+
			"total_received=%d send_mps=%.1f recv_mps=%.1f",
		row.Senders, row.MsgsPerSender, row.PayloadBytes, row.Run,
		row.TotalMessages, row.SendElapsedS, row.TotalElapsedS,
		row.TotalReceived, row.SendMPS, row.RecvMPS,
	)
}

func writeBasicBenchCSV(path string, rows []basicBenchRow) {
	f, err := os.Create(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer f.Close()

	w := csv.NewWriter(f)
	gomega.Expect(w.Write([]string{
		"senders", "messages_per_sender", "payload_bytes", "total_messages",
		"run", "send_elapsed_s", "total_elapsed_s", "total_received", "send_mps", "recv_mps",
	})).To(gomega.Succeed())

	for _, row := range rows {
		gomega.Expect(w.Write([]string{
			strconv.Itoa(row.Senders),
			strconv.Itoa(row.MsgsPerSender),
			strconv.Itoa(row.PayloadBytes),
			strconv.FormatInt(row.TotalMessages, 10),
			strconv.Itoa(row.Run),
			strconv.FormatFloat(row.SendElapsedS, 'f', 6, 64),
			strconv.FormatFloat(row.TotalElapsedS, 'f', 6, 64),
			strconv.FormatInt(row.TotalReceived, 10),
			strconv.FormatFloat(row.SendMPS, 'f', 2, 64),
			strconv.FormatFloat(row.RecvMPS, 'f', 2, 64),
		})).To(gomega.Succeed())
	}

	w.Flush()
	gomega.Expect(w.Error()).NotTo(gomega.HaveOccurred())
}
