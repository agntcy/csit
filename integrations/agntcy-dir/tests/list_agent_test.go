// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/agntcy/csit/integrations/testutils"
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
	)

	ginkgo.Context("agents push for listing", func() {
		ginkgo.It("should push and publish agents", func() {
			examplesDir := "../examples/"
			testDataPath, err := filepath.Abs(filepath.Join(examplesDir, "dir/e2e/testdata/examples/"))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			dockerImage = fmt.Sprintf("%s/dir-ctl:%s", os.Getenv("IMAGE_REPO"), os.Getenv("DIRECTORY_IMAGE_TAG"))
			mountDest = "/testdata"
			mountString = fmt.Sprintf("%s:%s", testDataPath, mountDest)

			agents = append(agents, &agent{modelFile: filepath.Join(mountDest, "crewai.agent.json")})
			agents = append(agents, &agent{modelFile: filepath.Join(mountDest, "langgraph.agent.json")})
			agents = append(agents, &agent{modelFile: filepath.Join(mountDest, "llama-index.agent.json")})

			for _, agent := range agents {
				dirctlArgs := []string{
					"push",
					agent.modelFile,
				}

				if runtime.GOOS != "linux" {
					dirctlArgs = append(dirctlArgs,
						"--server-addr",
						"host.docker.internal:8888",
					)
				}

				runner := testutils.NewDockerRunner(dockerImage, mountString, nil)
				outputBuffer, err := runner.Run(dirctlArgs...)
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), outputBuffer.String())

				agent.digest = strings.Trim(outputBuffer.String(), "\n")
				_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "DIGEST: %v\n", agent.digest)

				dirctlArgs = []string{
					"publish",
					agent.digest,
				}

				if runtime.GOOS != "linux" {
					dirctlArgs = append(dirctlArgs,
						"--server-addr",
						"host.docker.internal:8888",
					)
				}

				runner = testutils.NewDockerRunner(dockerImage, mountString, nil)
				outputBuffer, err = runner.Run(dirctlArgs...)
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), outputBuffer.String())
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
				}

				dirctlArgs = append(dirctlArgs, labels...)

				if runtime.GOOS != "linux" {
					dirctlArgs = append(dirctlArgs,
						"--server-addr",
						"host.docker.internal:8888",
					)
				}

				_, err := fmt.Fprintf(ginkgo.GinkgoWriter, "dirctl args: %v\n", dirctlArgs)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				runner := testutils.NewDockerRunner(dockerImage, mountString, nil)
				outputBuffer, err := runner.Run(dirctlArgs...)
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), outputBuffer.String())

				if expectFound {
					for _, agent := range agents {
						gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring(agent.digest))
					}
				} else {
					gomega.Expect(outputBuffer.String()).To(gomega.BeEmpty())
				}

			},
			ginkgo.Entry("list with one label", []string{"Natural Language Understanding"}, true),
			ginkgo.Entry("list with two labes", []string{"Natural Language Understanding", "Fact Extraction"}, true),
			ginkgo.Entry("list with non-existing label", []string{"Lorem ipsum dolor sit amet"}, false),
		)
	})
})
