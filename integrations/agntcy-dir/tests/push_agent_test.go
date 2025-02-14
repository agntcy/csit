// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
		agentID        string
	)

	ginkgo.BeforeEach(func() {
		examplesDir := "../examples/"
		testDataDir, err := filepath.Abs(filepath.Join(examplesDir, "testdata"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		dockerImage = fmt.Sprintf("%s/dir-ctl:%s", os.Getenv("IMAGE_REPO"), os.Getenv("DIRECTORY_IMAGE_TAG"))
		mountDest = "/opt/testdata"
		mountString = fmt.Sprintf("%s:%s", testDataDir, mountDest)

		agentModelFile = filepath.Join(mountDest, "expected_agent.json")
	})

	ginkgo.Context("agent push and pull", func() {
		ginkgo.It("should push an agent", func() {

			dirctlArgs := []string{
				"push",
				"--from-file",
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

			var response map[string]any

			// Unmarshal or Decode the JSON to the interface.
			err = json.Unmarshal(outputBuffer.Bytes(), &response)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), outputBuffer.String())

			_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "agentID: %v\n", response["id"])
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			agentID = fmt.Sprintf("%v", response["id"])
		})

		ginkgo.It("should pull an agent", func() {

			dirctlArgs := []string{
				"pull",
				"--id",
				agentID,
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
		})
	})
})
