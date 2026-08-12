// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func cleanupProcesses() {
	patterns := []string{
		"/p2p-agent",
		"/slim-agent",
		"slimctl slim start -c",
	}
	for _, pattern := range patterns {
		cmd := exec.Command("pkill", "-f", pattern)
		_ = cmd.Run()
	}
	time.Sleep(200 * time.Millisecond)
}

func ensureScenario(cfg matrixConfig) {
	if err := os.MkdirAll(cfg.MatrixDir, 0o755); err != nil {
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}
	if _, err := os.Stat(cfg.ScenarioPath); err == nil {
		return
	}
	cmd := exec.Command(
		"go", "run", "./tools/gen_scenario",
		"-matrix-naming",
		"-family", "matrix",
		"-agents", fmt.Sprintf("%d", cfg.Agents),
		"-think-ms", fmt.Sprintf("%d", cfg.ThinkMs),
		"-payload-bytes", fmt.Sprintf("%d", cfg.PayloadBytes),
		"-latency-ms", fmt.Sprintf("%d", cfg.LatencyMs),
		"-epochs", fmt.Sprintf("%d", cfg.Epochs),
		"-max-epoch-ms", fmt.Sprintf("%d", cfg.MaxEpochMs),
		"-output", cfg.ScenarioPath,
	)
	cmd.Dir = cfg.RootDir
	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(session, 2*time.Minute).Should(gexec.Exit(0))
}

func resetCellTSV(cfg matrixConfig) {
	if err := os.MkdirAll(cfg.RawDir, 0o755); err != nil {
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}
	_ = os.Remove(cfg.CellTSV)
}

type slimStack struct {
	configPath string
	slimctl    string
	cmd        *exec.Cmd
}

func startSlimStack(cfg matrixConfig) *slimStack {
	dataplanePort := 46357
	controllerPort := 46358
	configPath, err := os.CreateTemp("", "consensus-slim-*.yaml")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	config := fmt.Sprintf(`services:
  slim/0:
    node_id: "slim-bench-v2"
    group_name: "bench-v2"
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
	_, err = configPath.WriteString(config)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(configPath.Close()).To(gomega.Succeed())

	logPath := filepath.Join(cfg.ReportsDir, "slimctl.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	cmd := exec.Command(cfg.SlimctlPath, "slim", "start", "-c", configPath.Name())
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	gomega.Expect(cmd.Start()).To(gomega.Succeed())

	deadline := time.Now().Add(15 * time.Second)
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(dataplanePort))
	for time.Now().Before(deadline) {
		conn, dialErr := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	return &slimStack{configPath: configPath.Name(), slimctl: cfg.SlimctlPath, cmd: cmd}
}

func (s *slimStack) stop() {
	if s == nil {
		return
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Signal(syscall.SIGTERM)
		_, _ = s.cmd.Process.Wait()
	}
	stop := exec.Command(s.slimctl, "slim", "stop")
	_ = stop.Run()
	if s.configPath != "" {
		_ = os.Remove(s.configPath)
	}
}

func runP2P(cfg matrixConfig, repeat int) {
	cleanupProcesses()
	jsonOut := filepath.Join(cfg.ReportsDir, fmt.Sprintf("p2p-%s-rep%02d.json", cfg.ScenarioBase, repeat))
	cmd := exec.Command(
		filepath.Join(cfg.BinDir, "p2p-runner"),
		"--scenario", cfg.ScenarioPath,
		"--agent-bin", filepath.Join(cfg.BinDir, "p2p-agent"),
		"--quiet",
		"--run-id", fmt.Sprintf("%d", repeat),
		"--relay-grpc-port", fmt.Sprintf("%d", cfg.P2PRelayGRPC),
		"--relay-card-port", fmt.Sprintf("%d", cfg.P2PRelayCard),
		"--output-json", jsonOut,
		"--output-tsv", cfg.CellTSV,
	)
	cmd.Dir = cfg.RootDir
	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(session, cfg.epochBudget()).Should(gexec.Exit())
}

func runSLIM(cfg matrixConfig, repeat int) {
	cleanupProcesses()
	if _, err := os.Stat(cfg.SlimctlPath); err != nil {
		download := exec.Command("task", "deps:slimctl-download", fmt.Sprintf("COMPARE_SLIMCTL=%s", cfg.SlimctlPath))
		download.Dir = cfg.RootDir
		session, err := gexec.Start(download, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Eventually(session, 5*time.Minute).Should(gexec.Exit(0))
	}

	stack := startSlimStack(cfg)
	defer stack.stop()

	jsonOut := filepath.Join(cfg.ReportsDir, fmt.Sprintf("slim-%s-rep%02d.json", cfg.ScenarioBase, repeat))
	cmd := exec.Command(
		filepath.Join(cfg.BinDir, "slim-runner"),
		"--scenario", cfg.ScenarioPath,
		"--endpoint", cfg.SlimEndpoint,
		"--agent-bin", filepath.Join(cfg.BinDir, "slim-agent"),
		"--quiet",
		"--run-id", fmt.Sprintf("%d", repeat),
		"--output-json", jsonOut,
		"--output-tsv", cfg.CellTSV,
	)
	cmd.Dir = cfg.RootDir
	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(session, cfg.epochBudget()).Should(gexec.Exit())
}

func (cfg matrixConfig) epochBudget() time.Duration {
	perEpoch := time.Duration(cfg.MaxEpochMs)*time.Millisecond + 60*time.Second
	return time.Duration(cfg.Epochs)*perEpoch + 10*time.Minute
}

func writeProgressLine(format string, args ...any) error {
	_, err := fmt.Fprintf(ginkgo.GinkgoWriter, format+"\n", args...)
	return err
}
