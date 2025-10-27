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

var _ = ginkgo.Describe("Agntcy record push tests", func() {
	var (
		dockerImage     string
		mountDest       string
		mountString     string
		recordModelFile string
		cid             string
		runner          testrunner.Runner
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

	ginkgo.Context("record push and pull", func() {
		ginkgo.It("should push an record", func() {

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

		ginkgo.It("should pull an record", func() {

			_, err := fmt.Fprintf(ginkgo.GinkgoWriter, "cid: %s\n", cid)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			dirctlArgs := []string{
				"pull",
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
		})
	})
})
