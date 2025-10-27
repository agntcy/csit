// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	testrunner "github.com/agntcy/csit/integrations/testutils/runner"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Agntcy directory networking test", func() {
	var (
		dockerImage      string
		mountDest        string
		mountString      string
		recordModelFile  string
		cid              string
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

		recordModelFile = filepath.Join(mountDest, "record_031.json")
	})

	ginkgo.Context("record push, publish and list from another peer", func() {
		ginkgo.It("should push an record", func() {
			dirAPIPort = peerApiHostPorts[0]

			dirctlArgs := []string{
				"push",
				recordModelFile,
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

			var found bool
			cid, found = strings.CutPrefix(cmdOutput, "Pushed record with CID: ")

			gomega.Expect(found).To(gomega.BeTrue(), "Could not find CID prefix in dirctl output")

			cid = strings.TrimSpace(cid)
		})

		ginkgo.It("should publish an record to network", func() {
			_, err := fmt.Fprintf(ginkgo.GinkgoWriter, "digest: %s\n", cid)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			dirAPIPort = peerApiHostPorts[0]

			dirctlArgs := []string{
				"routing",
				"publish",
				cid,
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

			time.Sleep(3 * time.Second) // NOTE: Wait for publication
		})

		ginkgo.It("should list an record from another peer", func() {
			_, err := fmt.Fprintf(ginkgo.GinkgoWriter, "CID: %s\n", cid)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			dirAPIPort = peerApiHostPorts[1]

			dirctlArgs := []string{
				"routing",
				"search",
				"--locator",
				"docker-image",
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

			gomega.Expect(cmdOutput).To(gomega.ContainSubstring(cid))
		})
	})
})
