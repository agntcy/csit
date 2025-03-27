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

var _ = ginkgo.Describe("Agntcy agent push tests", func() {
	var (
		dockerImage    string
		mountDest      string
		mountString    string
		agentModelFile string
		digest         string
	)

	ginkgo.BeforeEach(func() {
		examplesDir := "../examples/"
		testDataPath, err := filepath.Abs(filepath.Join(examplesDir, "dir/e2e/testdata"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dockerImage = fmt.Sprintf("%s/dir-ctl:%s", os.Getenv("IMAGE_REPO"), os.Getenv("DIRECTORY_IMAGE_TAG"))
		mountDest = "/testdata"
		mountString = fmt.Sprintf("%s:%s", testDataPath, mountDest)

		agentModelFile = filepath.Join(mountDest, "agent.json")
	})

	ginkgo.Context("agent push and pull", func() {
		ginkgo.It("should push an agent", func() {

			dirctlArgs := []string{
				"push",
				agentModelFile,
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

			digest = strings.Trim(outputBuffer.String(), "\n")
		})

		ginkgo.It("should pull an agent", func() {

			_, err := fmt.Fprintf(ginkgo.GinkgoWriter, "digest: %s\n", digest)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			dirctlArgs := []string{
				"pull",
				digest,
			}

			if runtime.GOOS != "linux" {
				dirctlArgs = append(dirctlArgs,
					"--server-addr",
					"host.docker.internal:8888",
				)
			}

			_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "dirctl args: %v\n", dirctlArgs)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			runner := testutils.NewDockerRunner(dockerImage, mountString, nil)
			outputBuffer, err := runner.Run(dirctlArgs...)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), outputBuffer.String())
		})
	})
})
