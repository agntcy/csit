// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	testrunner "github.com/agntcy/csit/integrations/testutils/runner"
)

var _ = ginkgo.Describe("Agntcy agent list tests", func() {
	type agent struct {
		modelFile string
		digest    string
	}

	var (
		dockerImage string
		mountDest   string
		mountString string
		agents      []*agent
		runner      testrunner.Runner
	)

	ginkgo.Context("agents push for listing", func() {
		ginkgo.It("should push and publish agents", func() {
			examplesDir := "../examples/"
			testDataPath, err := filepath.Abs(filepath.Join(examplesDir, "dir/e2e/testdata/examples/"))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			dockerImage = fmt.Sprintf("%s/dir-ctl:%s", os.Getenv("IMAGE_REPO"), os.Getenv("DIRECTORY_IMAGE_TAG"))

			if os.Getenv("RUNNER_TYPE") == "local" {
				mountDest = testDataPath
			} else {
				mountDest = "/testdata"
				mountString = fmt.Sprintf("%s:%s", testDataPath, mountDest)
			}

			agents = append(agents, &agent{modelFile: filepath.Join(mountDest, "crewai.agent.json")})
			agents = append(agents, &agent{modelFile: filepath.Join(mountDest, "langgraph.agent.json")})
			agents = append(agents, &agent{modelFile: filepath.Join(mountDest, "llama-index.agent.json")})

			for _, agent := range agents {
				dirctlArgs := []string{
					"push",
					agent.modelFile,
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
						testrunner.WithDockerImage(dockerImage),
						testrunner.WithDockerArgs([]string{"run", "-v" + mountString}),
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

				agent.digest = strings.Trim(cmdOutput, "\n")
				_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "DIGEST: %v\n", agent.digest)

				gomega.Expect(err).NotTo(gomega.HaveOccurred(), cmdOutput)

				dirctlArgs = []string{
					"publish",
					agent.digest,
					"--server-addr",
					fmt.Sprintf("%s:%d", dirAPIHost, dirAPIPort),
				}

				_, err = runner.Run("dirctl", dirctlArgs...)
				if err != nil {
					exitErr, ok := err.(*exec.ExitError)
					if ok {
						err = fmt.Errorf("%s, stderr:%s", exitErr.String(), string(exitErr.Stderr))
					}
				}

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		})

		ginkgo.DescribeTable("list agents using categories",
			func(categories []string, expectFound bool) {

				labels := []string{}
				for _, category := range categories {
					labels = append(labels, "/skills/"+category)
				}

				dirctlArgs := []string{
					"list",
					"--server-addr",
					fmt.Sprintf("%s:%d", dirAPIHost, dirAPIPort),
				}

				dirctlArgs = append(dirctlArgs, labels...)

				_, err := fmt.Fprintf(ginkgo.GinkgoWriter, "dirctl args: %v\n", dirctlArgs)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				cmdOutput, err := runner.Run("dirctl", dirctlArgs...)

				if err != nil {
					exitErr, ok := err.(*exec.ExitError)
					if ok {
						err = fmt.Errorf("%s, stderr:%s", exitErr.String(), string(exitErr.Stderr))
					}
				}

				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				if expectFound {
					for _, agent := range agents {
						gomega.Expect(cmdOutput).To(gomega.ContainSubstring(agent.digest))
					}
				} else {
					gomega.Expect(cmdOutput).To(gomega.BeEmpty())
				}

			},
			ginkgo.Entry("list with one label", []string{"Natural Language Understanding"}, true),
			ginkgo.Entry("list with two labes", []string{"Natural Language Understanding", "Fact Extraction"}, true),
			ginkgo.Entry("list with non-existing label", []string{"Lorem ipsum dolor sit amet"}, false),
		)
	})
})
