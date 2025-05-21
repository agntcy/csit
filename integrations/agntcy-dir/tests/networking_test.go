// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	testrunner "github.com/agntcy/csit/integrations/testutils/runner"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Agntcy directory networking test", func() {
	var (
		dockerImage      string
		mountDest        string
		mountString      string
		agentModelFile   string
		digest           string
		runner           testrunner.Runner
		peerApiHostPorts = []int{8890, 8891, 8892}
		dirAPIPort       = dirAPIPort // NOTE: Shadow the suite variable
	)

	ginkgo.BeforeEach(func() {
		examplesDir := "../examples/"
		testDataPath, err := filepath.Abs(filepath.Join(examplesDir, "dir/e2e/testdata"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dockerImage = fmt.Sprintf("%s/dir-ctl:%s", os.Getenv("IMAGE_REPO"), os.Getenv("DIRECTORY_IMAGE_TAG"))

		if os.Getenv("RUNNER_TYPE") == "local" {
			mountDest = testDataPath
		} else {
			mountDest = "/testdata"
			mountString = fmt.Sprintf("%s:%s", testDataPath, mountDest)
		}

		agentModelFile = filepath.Join(mountDest, "agent.json")
	})

	ginkgo.Context("agent push, publish and list from another peer", func() {
		ginkgo.It("should push an agent", func() {
			dirAPIPort = peerApiHostPorts[0]

			dirctlArgs := []string{
				"push",
				agentModelFile,
				"--server-addr",
				fmt.Sprintf("%s:%d", dirAPIHost, dirAPIPort),
			}

			var err error

			switch os.Getenv("RUNNER_TYPE") {
			case "local":
				runner, err = testrunner.NewRunner(testrunner.RunnerTypeLocal, nil)
			default:
				runner, err = testrunner.NewRunner(testrunner.RunnerTypeDocker,
					testrunner.WithDockerCmd("docker"),
					testrunner.WithDockerArgs([]string{"run", "-v", mountString}),
					testrunner.WithDockerImage(dockerImage),
				)
			}

			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cmdOutput, err := runner.Run("dirctl", dirctlArgs...)

			if err != nil {
				exitErr, ok := err.(*exec.ExitError)
				if ok {
					err = fmt.Errorf("%s, stderr:%s", exitErr.String(), string(exitErr.Stderr))
				}
			}

			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			digest = strings.Trim(cmdOutput, "\n")
		})

		ginkgo.It("should publish an agent to network", func() {
			_, err := fmt.Fprintf(ginkgo.GinkgoWriter, "digest: %s\n", digest)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			dirAPIPort = peerApiHostPorts[0]

			dirctlArgs := []string{
				"publish",
				digest,
				"--network",
				"--server-addr",
				fmt.Sprintf("%s:%d", dirAPIHost, dirAPIPort),
			}

			_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "dirctl args: %v\n", dirctlArgs)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			_, err = runner.Run("dirctl", dirctlArgs...)

			if err != nil {
				exitErr, ok := err.(*exec.ExitError)
				if ok {
					err = fmt.Errorf("%s, stderr:%s", exitErr.String(), string(exitErr.Stderr))
				}
			}

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should list an agent from another peer", func() {
			_, err := fmt.Fprintf(ginkgo.GinkgoWriter, "digest: %s\n", digest)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			dirAPIPort = peerApiHostPorts[1]

			dirctlArgs := []string{
				"list",
				"--digest",
				digest,
				"--server-addr",
				fmt.Sprintf("%s:%d", dirAPIHost, dirAPIPort),
			}

			_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "dirctl args: %v\n", dirctlArgs)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cmdOutput, err := runner.Run("dirctl", dirctlArgs...)

			if err != nil {
				exitErr, ok := err.(*exec.ExitError)
				if ok {
					err = fmt.Errorf("%s, stderr:%s", exitErr.String(), string(exitErr.Stderr))
				}
			}

			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(cmdOutput).To(gomega.ContainSubstring(digest))
		})
	})
})
