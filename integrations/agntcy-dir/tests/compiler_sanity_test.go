// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/agntcy/csit/integrations/testutils"
)

var _ = ginkgo.Describe("Agntcy compiler sanity tests", func() {
	var (
		tempAgentPath          string
		dockerImage            string
		mountDest              string
		mountString            string
		modelConfigFilePath    string
		expectedAgentModelFile string
	)

	ginkgo.BeforeEach(func() {
		examplesDir := "../examples/"
		testDataPath, err := filepath.Abs(filepath.Join(examplesDir, "dir/e2e/testdata"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		tempAgentPath = filepath.Join(os.TempDir(), "agent.json")
		dockerImage = fmt.Sprintf("%s/dir-ctl:%s", os.Getenv("IMAGE_REPO"), os.Getenv("DIRECTORY_IMAGE_TAG"))
		mountDest = "/testdata"
		mountString = fmt.Sprintf("%s:%s", testDataPath, mountDest)

		modelConfigFilePath = filepath.Join(mountDest, "build.config.yaml")
		expectedAgentModelFile = filepath.Join(testDataPath, "agent.json")
	})

	ginkgo.Context("agent compilation", func() {
		ginkgo.It("should compile an agent", func() {
			dirctlArgs := []string{
				"build",
				"--config",
				modelConfigFilePath,
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

			// Filter "created_at" field
			filter := cmp.FilterPath(func(p cmp.Path) bool {
				// Ensure the path is deep enough
				if len(p) >= 3 {
					if mapStep, ok := p[len(p)-3].(cmp.MapIndex); ok {
						if key, ok := mapStep.Key().Interface().(string); ok && key == "created_at" || key == "extensions" {
							return true // Ignore these paths
						}
					}
				}
				return false // Include all other paths
			}, cmp.Ignore())

			gomega.Expect(expected).Should(gomega.BeComparableTo(compiled, filter))
			gomega.Expect(expected["extensions"]).Should(gomega.ConsistOf(compiled["extensions"]))
		})
	})
})
