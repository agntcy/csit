// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/agntcy/csit/integrations/testutils"
)

var _ = ginkgo.Describe("Agntcy compiler tests", func() {
	var (
		tempAgentPath          string
		dockerImage            string
		mountDest              string
		mountString            string
		expectedAgentModelFile string
	)

	ginkgo.BeforeEach(func() {
		examplesDir := "../examples/"
		marketingStrategyPath, err := filepath.Abs(filepath.Join(examplesDir, "dir/e2e/testdata/marketing-strategy"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		tempAgentPath = filepath.Join(os.TempDir(), "agent.json")
		dockerImage = fmt.Sprintf("%s/dir-ctl:%s", os.Getenv("IMAGE_REPO"), os.Getenv("DIRECTORY_IMAGE_TAG"))
		mountDest = "/opt/marketing-strategy"
		mountString = fmt.Sprintf("%s:%s", marketingStrategyPath, mountDest)

		testdataDir, err := filepath.Abs(filepath.Join(examplesDir, "testdata"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		expectedAgentModelFile = filepath.Join(testdataDir, "expected_agent.json")
	})

	ginkgo.Context("agent compilation", func() {
		ginkgo.It("should compile an agent", func() {

			dirctlArgs := []string{
				"build",
				"--name=marketing-strategy",
				"--version=v1.0.0",
				"--created-at=2025-01-01T00:00:00Z",
				"--artifact-type=python-package",
				"--artifact-url=http://ghcr.io/agntcy/marketing-strategy",
				"--author=author1",
				"--author=author2",
				mountDest,
			}

			runner := testutils.NewDockerRunner(dockerImage, mountString, nil)
			outputBuffer, err := runner.Run(dirctlArgs...)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), outputBuffer.String())

			err = os.WriteFile(tempAgentPath, outputBuffer.Bytes(), 0644)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("agent model should be the expected", func() {
			var expected, compiled map[string]any

			expactedModelJSON, err := os.ReadFile(expectedAgentModelFile)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Unmarshal or Decode the JSON to the interface.
			err = json.Unmarshal([]byte(expactedModelJSON), &expected)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			compiledModelJSON, err := os.ReadFile(tempAgentPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Unmarshal or Decode the JSON to the interface.
			err = json.Unmarshal([]byte(compiledModelJSON), &compiled)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Check the compiled agent model without extensions field
			gomega.Expect(expected).To(gomega.BeComparableTo(compiled))
		})
	})
})
