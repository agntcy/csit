// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/metrics"
	"github.com/agntcy/csit/benchmarks/slim-vs-a2a/internal/plan"
)

var (
	binDir         string
	planDir        string
	reportsDir     string
	resultsTSV        string
	slimctlPath       string
	slimConfigPath    string
	slimSession       *gexec.Session
)

var _ = ginkgo.BeforeSuite(func() {
	if os.Getenv("RUN_SLIM_VS_A2A") != "1" {
		return
	}

	var err error
	binDir, err = os.MkdirTemp("", "slim-vs-a2a-bin-*")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	build := exec.Command("go", "build", "-o", filepath.Join(binDir, "a2a-agent"), "./a2a/cmd/agent")
	build.Dir = repoRoot()
	gomega.Expect(build.Run()).To(gomega.Succeed())

	build = exec.Command("go", "build", "-o", filepath.Join(binDir, "a2a-runner"), "./a2a/cmd/runner")
	build.Dir = repoRoot()
	gomega.Expect(build.Run()).To(gomega.Succeed())

	setup := exec.Command("go", "run", "github.com/agntcy/slim-bindings-go/cmd/slim-bindings-setup@v1.4.0")
	setup.Dir = repoRoot()
	gomega.Expect(setup.Run()).To(gomega.Succeed())

	build = exec.Command("go", "build", "-o", filepath.Join(binDir, "slim-agent"), "./slim/cmd/agent")
	build.Dir = repoRoot()
	build.Env = append(os.Environ(), slimCgoLDFlags()...)
	gomega.Expect(build.Run()).To(gomega.Succeed())

	build = exec.Command("go", "build", "-o", filepath.Join(binDir, "slim-runner"), "./slim/cmd/runner")
	build.Dir = repoRoot()
	build.Env = append(os.Environ(), slimCgoLDFlags()...)
	gomega.Expect(build.Run()).To(gomega.Succeed())

	planDir = filepath.Join(repoRoot(), "plans", "domains")
	reportsDir = filepath.Join(repoRoot(), "reports")
	resultsTSV = filepath.Join(reportsDir, "results.tsv")
	gomega.Expect(os.MkdirAll(reportsDir, 0o755)).To(gomega.Succeed())
	slimctlPath = filepath.Join(repoRoot(), "..", "agntcy-slim", "bin", "slimctl")
	if _, err := os.Stat(slimctlPath); err != nil {
		download := exec.Command("task", "deps:slimctl-download")
		download.Dir = repoRoot()
		gomega.Expect(download.Run()).To(gomega.Succeed())
	}
})

var _ = ginkgo.AfterSuite(func() {
	stopSlimStack()
	if binDir != "" {
		_ = os.RemoveAll(binDir)
	}
	gexec.CleanupBuildArtifacts()
})

var _ = ginkgo.Describe("SLIM vs A2A comparison", ginkgo.Label("slim-vs-a2a"), func() {
	ginkgo.BeforeEach(func() {
		if os.Getenv("RUN_SLIM_VS_A2A") != "1" {
			ginkgo.Skip("set RUN_SLIM_VS_A2A=1 to run comparison suite")
		}
	})

	ginkgo.It("runs all domain plans on A2A and SLIM", func() {
		plans, err := plan.LoadDir(planDir)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(plans).NotTo(gomega.BeEmpty())

		_ = os.Remove(resultsTSV)

		for _, p := range plans {
			planPath := filepath.Join(planDir, p.Metadata.Name+".yaml")
			a2aResult := runA2A(planPath, p.Metadata.Name, 0)
			gomega.Expect(a2aResult.PlanName).To(gomega.Equal(p.Metadata.Name))
			gomega.Expect(a2aResult.Implementation).To(gomega.Equal("a2a-grpc"))
			gomega.Expect(a2aResult.TotalWallClockMS).To(gomega.BeNumerically(">", 0))

			startSlimStack()
			slimResult := runSLIM(planPath, p.Metadata.Name, "multicast", 0)
			stopSlimStack()

			gomega.Expect(slimResult.PlanName).To(gomega.Equal(p.Metadata.Name))
			gomega.Expect(slimResult.Implementation).To(gomega.Equal("slim-multicast"))
			gomega.Expect(slimResult.TotalWallClockMS).To(gomega.BeNumerically(">", 0))

			logComparison(p.Metadata.Name, a2aResult, slimResult)
		}
	})

	ginkgo.It("logs sweep crossover hints at 10 agents / 20ms budget", func() {
		sweepDir := filepath.Join(repoRoot(), "plans", "sweeps")
		gomega.Expect(os.MkdirAll(sweepDir, 0o755)).To(gomega.Succeed())
		planPath := filepath.Join(sweepDir, "ginkgo-sweep-sustainable-10ag-20ms.yaml")

		gen := exec.Command("go", "run", "./tools/gen_plan",
			"-family", "sustainable-resource",
			"-agents", "10",
			"-round-budget-ms", "20",
			"-output", planPath,
		)
		gen.Dir = repoRoot()
		gomega.Expect(gen.Run()).To(gomega.Succeed())

		a2aResult := runA2A(planPath, "sweep-a2a", 20)
		startSlimStack()
		slimResult := runSLIM(planPath, "sweep-slim", "multicast", 20)
		stopSlimStack()

		ginkgo.GinkgoWriter.Printf(
			"sweep agents=10 budget=20ms a2a_p95=%d slim_p95=%d a2a_missing=%d slim_missing=%d\n",
			a2aResult.ContextPushP95MS,
			slimResult.ContextPushP95MS,
			a2aResult.CoordMissingResponses,
			slimResult.CoordMissingResponses,
		)

		if slimResult.ContextPushP95MS > a2aResult.ContextPushP95MS {
			ginkgo.GinkgoWriter.Printf("note: SLIM p95 higher than A2A at this scale (expected below crossover)\n")
		}
	})
})

func logComparison(name string, a2aResult, slimResult metrics.RunResult) {
	ginkgo.GinkgoWriter.Printf(
		"plan=%s a2a_wall=%dms slim_wall=%dms a2a_ctx=%dms slim_ctx=%dms a2a_p95=%d slim_p95=%d a2a_missing=%d slim_missing=%d\n",
		name,
		a2aResult.TotalWallClockMS,
		slimResult.TotalWallClockMS,
		a2aResult.ContextPushMS,
		slimResult.ContextPushMS,
		a2aResult.ContextPushP95MS,
		slimResult.ContextPushP95MS,
		a2aResult.CoordMissingResponses,
		slimResult.CoordMissingResponses,
	)
}

func runA2A(planPath, jsonName string, roundBudgetMS int64) metrics.RunResult {
	jsonPath := filepath.Join(reportsDir, jsonName+".json")
	args := []string{
		"--plan", planPath,
		"--agent-bin", filepath.Join(binDir, "a2a-agent"),
		"--output-json", jsonPath,
		"--wait-ready", "4s",
		"--quiet",
	}
	if roundBudgetMS > 0 {
		args = append(args, "--round-budget-ms", fmt.Sprintf("%d", roundBudgetMS))
	}
	cmd := exec.Command(filepath.Join(binDir, "a2a-runner"), args...)
	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(session, 3*time.Minute).Should(gexec.Exit(0))
	return readResult(jsonPath)
}

func runSLIM(planPath, jsonName, coordMode string, roundBudgetMS int64) metrics.RunResult {
	jsonPath := filepath.Join(reportsDir, jsonName+".json")
	args := []string{
		"--plan", planPath,
		"--endpoint", "http://127.0.0.1:46357",
		"--agent-bin", filepath.Join(binDir, "slim-agent"),
		"--coord-mode", coordMode,
		"--output-json", jsonPath,
		"--wait-ready", "4s",
		"--quiet",
	}
	if roundBudgetMS > 0 {
		args = append(args, "--round-budget-ms", fmt.Sprintf("%d", roundBudgetMS))
	}
	cmd := exec.Command(filepath.Join(binDir, "slim-runner"), args...)
	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(session, 3*time.Minute).Should(gexec.Exit(0))
	return readResult(jsonPath)
}

func readResult(path string) metrics.RunResult {
	data, err := os.ReadFile(path)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	var result metrics.RunResult
	gomega.Expect(json.Unmarshal(data, &result)).To(gomega.Succeed())
	return result
}

func repoRoot() string {
	wd, err := os.Getwd()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(wd, "a2a")); err == nil {
				return wd
			}
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			ginkgo.Fail("could not locate slim-vs-a2a module root")
		}
		wd = parent
	}
}

func slimCgoLDFlags() []string {
	out, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return nil
	}
	cacheDir := filepath.Join(strings.TrimSpace(string(out)), ".cgo-cache", "slim-bindings", "v1.4.0")
	return []string{"CGO_LDFLAGS=-L" + cacheDir}
}

func startSlimStack() {
	stopSlimStack()
	dataplanePort := 46357
	controllerPort := 46358
	configFile, err := os.CreateTemp("", "slim-vs-a2a-*.yaml")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	slimConfigPath = configFile.Name()
	config := fmt.Sprintf(`services:
  slim/0:
    node_id: "slim-vs-a2a"
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
	_, err = configFile.WriteString(config)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(configFile.Close()).To(gomega.Succeed())

	cmd := exec.Command(slimctlPath, "slim", "start", "-c", slimConfigPath)
	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	slimSession = session
	gomega.Expect(waitForPort(fmt.Sprintf("127.0.0.1:%d", dataplanePort), 20*time.Second)).To(gomega.Succeed())
}

func stopSlimStack() {
	if slimSession != nil {
		slimSession.Terminate().Wait(5 * time.Second)
		slimSession = nil
	}
	if slimConfigPath != "" {
		_ = os.Remove(slimConfigPath)
		slimConfigPath = ""
	}
}

func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", addr)
}
