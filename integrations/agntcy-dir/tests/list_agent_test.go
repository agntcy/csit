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

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	testrunner "github.com/agntcy/csit/integrations/testutils/runner"
)

var _ = ginkgo.Describe("Agntcy record list tests", func() {
	type record struct {
		modelFile string
		cid       string
	}

	var (
		dockerImage string
		mountDest   string
		mountString string
		records     []*record
		runner      testrunner.Runner
	)

	ginkgo.Context("record push for listing", func() {
		ginkgo.It("should push and publish record", func() {
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

			records = append(records, &record{modelFile: filepath.Join(mountDest, "crewai.agent.json")})
			records = append(records, &record{modelFile: filepath.Join(mountDest, "langgraph.agent.json")})
			records = append(records, &record{modelFile: filepath.Join(mountDest, "llama-index.agent.json")})

			for _, record := range records {
				dirctlArgs := []string{
					"push",
					record.modelFile,
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
				var found bool

				record.cid, found = strings.CutPrefix(cmdOutput, "Pushed record with CID: ")
				gomega.Expect(found).To(gomega.BeTrue(), "Could not find CID prefix in dirctl output")

				record.cid = strings.TrimSpace(record.cid)
				_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "CID: %v\n", record.cid)

				gomega.Expect(err).NotTo(gomega.HaveOccurred(), cmdOutput)

				dirctlArgs = []string{
					"routing",
					"publish",
					record.cid,
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

			time.Sleep(3 * time.Second) // NOTE: Wait for publication
		})

		ginkgo.DescribeTable("list records using categories",
			func(categories []string, expectFound bool) {

				labels := []string{}
				for _, category := range categories {
					labels = append(labels, "/skills/"+category)
				}

				dirctlArgs := []string{
					"routing",
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
					for _, record := range records {
						gomega.Expect(cmdOutput).To(gomega.ContainSubstring(record.cid))
					}
				} else {
					gomega.Expect(cmdOutput).ToNot(gomega.BeEmpty())
				}

			},
			ginkgo.Entry("list with one label", []string{"Natural Language Processing"}, true),
			ginkgo.Entry("list with two labes", []string{"Natural Language Processing", "Natural Language Processing"}, true), // NOTE: The samples jsons only contains one label
			ginkgo.Entry("list with non-existing label", []string{"Lorem ipsum dolor sit amet"}, false),
		)
	})
})
