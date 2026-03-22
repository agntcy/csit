// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/gmeasure"
)

var (
	buildDir       string
	rateClientPath string
	echoClientPath string
	slimSession    *gexec.Session
	echoSession    *gexec.Session
	serverEndpoint string
	configPath     string
)

var _ = ginkgo.BeforeSuite(func() {
	var err error
	buildDir, err = os.MkdirTemp("", "slim-bench-tests-*")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	absEchoDir, err := filepath.Abs("./echo-client")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	echoClientPath = filepath.Join(buildDir, "echo-client")
	echoCmd := exec.Command("go", "build", "-o", echoClientPath, ".")
	echoCmd.Dir = absEchoDir
	output, err := echoCmd.CombinedOutput()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to build echo-client: %s", string(output))

	absRateClientDir, err := filepath.Abs("./rate-client")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	rateClientPath = filepath.Join(buildDir, "rate-client")
	rateCmd := exec.Command("go", "build", "-o", rateClientPath, ".")
	rateCmd.Dir = absRateClientDir
	output, err = rateCmd.CombinedOutput()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to build rate-client: %s", string(output))
})

var _ = ginkgo.AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	if buildDir != "" {
		_ = os.RemoveAll(buildDir)
	}
})

var _ = ginkgo.Describe("Benchmarking slim local", func() {
	ginkgo.BeforeEach(func() {
		startLocalSlimStack()
	})

	ginkgo.AfterEach(func() {
		stopLocalSlimStack()
	})

	ginkgo.It("Measures throughput locally", func() {
		experiment := gmeasure.NewExperiment("Slim Benchmark Local")
		ginkgo.AddReportEntry(experiment.Name, experiment)

		experiment.SampleDuration("slim test 1000 messages", func(_ int) {
			stopEchoResponder()
			startEchoResponder("sink", 1, "")
			benchCmd := exec.Command(rateClientPath,
				"-mode", "fire-and-forget",
				"-clients", "1",
				"-msgs", "1000",
				"-local", "agntcy/demo/client",
				"-server", serverEndpoint,
				"-dest", "agntcy/demo/echo",
			)
			benchSession, err := gexec.Start(benchCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Eventually(benchSession, 20*time.Second).Should(gexec.Exit(0))
			stopEchoResponder()
			startEchoResponder("echo", 1, "")
		}, gmeasure.SamplingConfig{N: 1})
	})

	ginkgo.It("measures node peak capacity in msg/sec", func() {
		clientCount := envInt("SLIM_PEAK_CLIENTS", 10)
		payloadSize := envInt("SLIM_PEAK_PAYLOAD_SIZE", 128)
		duration := envDuration("SLIM_PEAK_DURATION", 3*time.Second)
		peakMode := envString("SLIM_PEAK_MODE", "fire-and-forget")
		gomega.Expect(peakMode).To(gomega.Or(gomega.Equal("fire-and-forget"), gomega.Equal("request-reply"), gomega.Equal("write")))

		stopEchoResponder()

		statsFile, err := os.CreateTemp(buildDir, "peak-capacity-*.stats")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		statsPath := statsFile.Name()
		gomega.Expect(statsFile.Close()).To(gomega.Succeed())
		defer os.Remove(statsPath)

		responderMode := ""
		if peakMode == "fire-and-forget" {
			responderMode = "sink"
		}
		if peakMode == "request-reply" {
			responderMode = "echo"
		}
		if peakMode == "write" {
			responderMode = "blackhole"
		}
		if responderMode != "" {
			startEchoResponder(responderMode, clientCount, statsPath)
		}

		reportPath := filepath.Join(buildDir, "peak-capacity-report.md")
		benchCmd := exec.Command(
			rateClientPath,
			"-mode", peakMode,
			"-clients", strconv.Itoa(clientCount),
			"-dest-sharded",
			"-size", strconv.Itoa(payloadSize),
			"-rate", "0",
			"-duration", duration.String(),
			"-local", "agntcy/demo/client",
			"-server", serverEndpoint,
			"-dest", "agntcy/demo/echo",
			"-output", reportPath,
		)
		benchSession, err := gexec.Start(benchCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Eventually(benchSession, 30*time.Second).Should(gexec.Exit(0))

		reportContent, err := os.ReadFile(reportPath)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		peakMPS := extractThroughputMPS(string(reportContent))
		gomega.Expect(peakMPS).To(gomega.BeNumerically(">", 0))

		time.Sleep(500 * time.Millisecond)
		ginkgo.AddReportEntry("Node Peak Capacity Config", fmt.Sprintf("mode=%s clients=%d payload=%d duration=%s", peakMode, clientCount, payloadSize, duration))
		ginkgo.AddReportEntry("Node Peak Capacity (msg/sec)", fmt.Sprintf("%.2f", peakMPS))
		if responderMode != "" {
			statsContent, err := os.ReadFile(statsPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			receivedMessages := extractStatsInt(string(statsContent), "received_messages")
			sinkMPS := extractStatsFloat(string(statsContent), "active_receive_mps")
			gomega.Expect(receivedMessages).To(gomega.BeNumerically(">", 0))
			gomega.Expect(sinkMPS).To(gomega.BeNumerically(">", 0))
			ginkgo.AddReportEntry("Node Peak Capacity Sink (msg/sec)", fmt.Sprintf("%.2f", sinkMPS))
			ginkgo.AddReportEntry("Node Peak Capacity Sink Messages", receivedMessages)
		}
	})
})

func waitForPort(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for %s", address)
}

func startLocalSlimStack() {
	var err error
	dataplanePort := allocatePort()
	controllerPort := allocatePort()
	serverEndpoint = fmt.Sprintf("http://127.0.0.1:%d", dataplanePort)

	configFile, err := os.CreateTemp(buildDir, "local-slim-*.yaml")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	configPath = configFile.Name()

	_, err = fmt.Fprintf(configFile, `tracing:
  log_level: info
  display_thread_names: true
  display_thread_ids: true

runtime:
  n_cores: 0
  thread_name: "slim-local"
  drain_timeout: 5s

services:
  slim/0:
    node_id: "slim-local"
    group_name: "bench"
    dataplane:
      servers:
        - endpoint: "127.0.0.1:%d"
          metadata:
            local_endpoint: "127.0.0.1"
            external_endpoint: "127.0.0.1:%d"
            trust_domain: "example.org"
          tls:
            insecure: true
    controller:
      servers:
        - endpoint: "127.0.0.1:%d"
          tls:
            insecure: true
`, dataplanePort, dataplanePort, controllerPort)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(configFile.Close()).To(gomega.Succeed())

	slimctlCmd := exec.Command("slimctl", "slim", "start", "-c", configPath)
	slimSession, err = gexec.Start(slimctlCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(waitForPort(fmt.Sprintf("127.0.0.1:%d", dataplanePort), 10*time.Second)).To(gomega.Succeed())

	echoCmd := exec.Command(
		echoClientPath,
		"-local", "agntcy/demo/echo",
		"-server", serverEndpoint,
	)
	echoSession, err = gexec.Start(echoCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(echoSession.Out, 10*time.Second).Should(gbytes.Say("ready"))
}

func startEchoResponder(mode string, clients int, statsFile string) {
	args := []string{
		echoClientPath,
		"-local", "agntcy/demo/echo",
		"-clients", strconv.Itoa(clients),
		"-mode", mode,
		"-server", serverEndpoint,
	}
	if statsFile != "" {
		args = append(args, "-stats-file", statsFile)
	}

	echoCmd := exec.Command(args[0], args[1:]...)
	var err error
	echoSession, err = gexec.Start(echoCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(echoSession.Out, 10*time.Second).Should(gbytes.Say("ready"))
}

func stopEchoResponder() {
	if echoSession != nil {
		echoSession.Terminate().Wait(5 * time.Second)
		echoSession = nil
	}
}

func stopLocalSlimStack() {
	stopEchoResponder()
	if slimSession != nil {
		slimSession.Terminate().Wait(5 * time.Second)
		slimSession = nil
	}
	if configPath != "" {
		_ = os.Remove(configPath)
		configPath = ""
	}
}

func allocatePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func extractThroughputMPS(report string) float64 {
	re := regexp.MustCompile(`Throughput:\*\*\s+([0-9]+\.[0-9]+) msg/sec`)
	matches := re.FindStringSubmatch(report)
	gomega.Expect(matches).To(gomega.HaveLen(2), "expected throughput line in report")
	value, err := strconv.ParseFloat(matches[1], 64)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return value
}

func extractStatsInt(stats string, key string) int64 {
	prefix := key + "="
	for _, line := range strings.Split(stats, "\n") {
		if strings.HasPrefix(line, prefix) {
			value, err := strconv.ParseInt(strings.TrimPrefix(line, prefix), 10, 64)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			return value
		}
	}
	ginkgo.Fail(fmt.Sprintf("missing stats key %s", key))
	return 0
}

func extractStatsFloat(stats string, key string) float64 {
	prefix := key + "="
	for _, line := range strings.Split(stats, "\n") {
		if strings.HasPrefix(line, prefix) {
			value, err := strconv.ParseFloat(strings.TrimPrefix(line, prefix), 64)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			return value
		}
	}
	ginkgo.Fail(fmt.Sprintf("missing stats key %s", key))
	return 0
}

func envString(key string, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

func envInt(key string, defaultValue int) int {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		parsed, err := strconv.Atoi(value)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid %s value %q", key, value)
		return parsed
	}
	return defaultValue
}

func envDuration(key string, defaultValue time.Duration) time.Duration {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		parsed, err := time.ParseDuration(value)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid %s value %q", key, value)
		return parsed
	}
	return defaultValue
}

func sessionPID(session *gexec.Session) int {
	if session == nil || session.Command == nil || session.Command.Process == nil {
		return 0
	}
	return session.Command.Process.Pid
}

func readSessionCPUSeconds(session *gexec.Session) (float64, error) {
	pid := sessionPID(session)
	if pid == 0 {
		return 0, fmt.Errorf("session pid is unavailable")
	}
	return readProcessCPUSeconds(pid)
}

func readProcessCPUSeconds(pid int) (float64, error) {
	if runtime.GOOS == "linux" {
		seconds, err := readLinuxProcessCPUSeconds(pid)
		if err == nil {
			return seconds, nil
		}
	}

	output, err := exec.Command("ps", "-o", "time=", "-p", strconv.Itoa(pid)).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("read cpu time for pid %d: %w (%s)", pid, err, strings.TrimSpace(string(output)))
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return 0, fmt.Errorf("empty cpu time for pid %d", pid)
	}
	lines := strings.Split(text, "\n")
	return parsePSCPUTime(lines[len(lines)-1])
}

func readLinuxProcessCPUSeconds(pid int) (float64, error) {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	content, err := os.ReadFile(statPath)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", statPath, err)
	}

	text := strings.TrimSpace(string(content))
	commandEnd := strings.LastIndex(text, ")")
	if commandEnd < 0 || commandEnd+2 >= len(text) {
		return 0, fmt.Errorf("unexpected proc stat format for pid %d", pid)
	}

	fields := strings.Fields(text[commandEnd+2:])
	if len(fields) < 13 {
		return 0, fmt.Errorf("insufficient proc stat fields for pid %d", pid)
	}

	userTicks, err := strconv.ParseFloat(fields[11], 64)
	if err != nil {
		return 0, fmt.Errorf("parse utime for pid %d: %w", pid, err)
	}
	systemTicks, err := strconv.ParseFloat(fields[12], 64)
	if err != nil {
		return 0, fmt.Errorf("parse stime for pid %d: %w", pid, err)
	}

	clockTicksPerSecond, err := linuxClockTicksPerSecond()
	if err != nil {
		return 0, err
	}
	return (userTicks + systemTicks) / clockTicksPerSecond, nil
}

func linuxClockTicksPerSecond() (float64, error) {
	output, err := exec.Command("getconf", "CLK_TCK").CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("read CLK_TCK: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	value := strings.TrimSpace(string(output))
	ticks, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse CLK_TCK %q: %w", value, err)
	}
	if ticks <= 0 {
		return 0, fmt.Errorf("invalid CLK_TCK %q", value)
	}
	return ticks, nil
}

func parsePSCPUTime(value string) (float64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("empty ps cpu time")
	}

	dayParts := strings.SplitN(trimmed, "-", 2)
	days := 0.0
	timePart := trimmed
	if len(dayParts) == 2 {
		parsedDays, err := strconv.Atoi(strings.TrimSpace(dayParts[0]))
		if err != nil {
			return 0, fmt.Errorf("parse ps cpu days %q: %w", dayParts[0], err)
		}
		days = float64(parsedDays)
		timePart = dayParts[1]
	}

	components := strings.Split(strings.TrimSpace(timePart), ":")
	if len(components) < 2 || len(components) > 3 {
		return 0, fmt.Errorf("unsupported ps cpu time %q", value)
	}

	secondsValue, err := strconv.ParseFloat(components[len(components)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("parse ps cpu seconds %q: %w", components[len(components)-1], err)
	}

	minutesValue, err := strconv.Atoi(components[len(components)-2])
	if err != nil {
		return 0, fmt.Errorf("parse ps cpu minutes %q: %w", components[len(components)-2], err)
	}

	hoursValue := 0
	if len(components) == 3 {
		hoursValue, err = strconv.Atoi(components[0])
		if err != nil {
			return 0, fmt.Errorf("parse ps cpu hours %q: %w", components[0], err)
		}
	}

	return days*24*3600 + float64(hoursValue*3600) + float64(minutesValue*60) + secondsValue, nil
}
