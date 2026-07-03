// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package harness

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

// Runtime manages a local SLIM node and benchmark client binaries for integration tests.
type Runtime struct {
	ClientsDir     string
	RateClientPath string
	EchoClientPath string
	BuildDir       string
	ServerEndpoint string

	slimSession *gexec.Session
	echoSession *gexec.Session
	configPath  string
}

// DefaultClientsDir resolves ../clients relative to this harness package.
func DefaultClientsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve harness package path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "clients"))
}

// New creates a runtime. clientsDir should contain echo-client/ and rate-client/.
func New(clientsDir string) *Runtime {
	if clientsDir == "" {
		clientsDir = DefaultClientsDir()
	}
	return &Runtime{ClientsDir: clientsDir}
}

// InitBuildArtifacts compiles echo-client and rate-client into a temp directory.
func (r *Runtime) InitBuildArtifacts() {
	var err error
	r.BuildDir, err = os.MkdirTemp("", "slim-evidence-tests-*")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	echoDir := filepath.Join(r.ClientsDir, "echo-client")
	r.EchoClientPath = filepath.Join(r.BuildDir, "echo-client")
	echoCmd := exec.Command("go", "build", "-o", r.EchoClientPath, ".")
	echoCmd.Dir = echoDir
	output, err := echoCmd.CombinedOutput()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to build echo-client: %s", string(output))

	rateDir := filepath.Join(r.ClientsDir, "rate-client")
	r.RateClientPath = filepath.Join(r.BuildDir, "rate-client")
	rateCmd := exec.Command("go", "build", "-o", r.RateClientPath, ".")
	rateCmd.Dir = rateDir
	output, err = rateCmd.CombinedOutput()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to build rate-client: %s", string(output))
}

// Cleanup removes build artifacts.
func (r *Runtime) Cleanup() {
	gexec.CleanupBuildArtifacts()
	if r.BuildDir != "" {
		_ = os.RemoveAll(r.BuildDir)
		r.BuildDir = ""
	}
}

// StartStack starts slimctl and a default echo peer.
func (r *Runtime) StartStack() {
	r.startLocalSlimStack()
}

// StopStack tears down slimctl and responders.
func (r *Runtime) StopStack() {
	r.stopLocalSlimStack()
}

// StopResponder stops the active echo/sink responder process.
func (r *Runtime) StopResponder() {
	r.stopEchoResponder()
}

// StartResponder starts echo-client in the given mode (echo, sink, blackhole).
func (r *Runtime) StartResponder(mode string, clients int, statsFile string) {
	r.startEchoResponder(mode, clients, statsFile)
}

func (r *Runtime) startLocalSlimStack() {
	dataplanePort := allocatePort()
	controllerPort := allocatePort()
	r.ServerEndpoint = fmt.Sprintf("http://127.0.0.1:%d", dataplanePort)

	configFile, err := os.CreateTemp(r.BuildDir, "local-slim-*.yaml")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	r.configPath = configFile.Name()

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

	slimctlCmd := exec.Command("slimctl", "slim", "start", "-c", r.configPath)
	r.slimSession, err = gexec.Start(slimctlCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(waitForPort(fmt.Sprintf("127.0.0.1:%d", dataplanePort), 10*time.Second)).To(gomega.Succeed())

	echoCmd := exec.Command(
		r.EchoClientPath,
		"-local", "agntcy/demo/echo",
		"-server", r.ServerEndpoint,
	)
	r.echoSession, err = gexec.Start(echoCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(r.echoSession.Out, 10*time.Second).Should(gbytes.Say("ready"))
}

func (r *Runtime) startEchoResponder(mode string, clients int, statsFile string) {
	args := []string{
		r.EchoClientPath,
		"-local", "agntcy/demo/echo",
		"-clients", strconv.Itoa(clients),
		"-mode", mode,
		"-server", r.ServerEndpoint,
	}
	if statsFile != "" {
		args = append(args, "-stats-file", statsFile)
	}

	echoCmd := exec.Command(args[0], args[1:]...)
	var err error
	r.echoSession, err = gexec.Start(echoCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(r.echoSession.Out, 10*time.Second).Should(gbytes.Say("ready"))
}

func (r *Runtime) stopEchoResponder() {
	if r.echoSession != nil {
		r.echoSession.Terminate().Wait(5 * time.Second)
		r.echoSession = nil
	}
}

func (r *Runtime) stopLocalSlimStack() {
	r.stopEchoResponder()
	if r.slimSession != nil {
		r.slimSession.Terminate().Wait(5 * time.Second)
		r.slimSession = nil
	}
	if r.configPath != "" {
		_ = os.Remove(r.configPath)
		r.configPath = ""
	}
}

// RunRateClient starts rate-client with the given arguments and waits for completion.
func (r *Runtime) RunRateClient(args []string, timeout time.Duration) *gexec.Session {
	cmd := exec.Command(r.RateClientPath, args...)
	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(session, timeout).Should(gexec.Exit())
	return session
}

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

func allocatePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// EnvString returns the environment variable value or default.
func EnvString(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// LogProgress writes a structured progress line to stderr (for CI log scraping).
func LogProgress(format string, args ...any) error {
	line := fmt.Sprintf(format, args...)
	_, err := fmt.Fprintf(os.Stderr, "\n%s\n", line)
	return err
}
