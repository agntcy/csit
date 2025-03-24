// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agntcy/csit/integrations/testutils"
	"github.com/google/go-cmp/cmp"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	buildConfigName   = "build.config.yml"
	expectedModelName = "model.json"
	samplesPath       = "../../../samples"
)

var _ = ginkgo.Describe("Samples build test", func() {
	var (
		dockerImage string
		samples     []string
	)

	samplesDir, err := filepath.Abs(samplesPath)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	samples, err = FindFilePairs(samplesDir, buildConfigName, expectedModelName)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	dockerImage = fmt.Sprintf("%s/dir-ctl:%s", os.Getenv("IMAGE_REPO"), os.Getenv("DIRECTORY_IMAGE_TAG"))

	_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "samples: %v\n", samples)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	for _, entry := range samples {
		entry := entry
		ginkgo.Context(entry, func() {
			var (
				tempAgentPath          string
				mountDest              string
				mountString            string
				modelConfigFilePath    string
				expectedAgentModelFile string
			)

			ginkgo.BeforeEach(func() {
				mountDest = fmt.Sprintf("/%s", filepath.Base(entry))
				mountString = fmt.Sprintf("%s:%s", entry, mountDest)
				modelConfigFilePath = filepath.Join(mountDest, buildConfigName)
				expectedAgentModelFile = filepath.Join(entry, expectedModelName)
				tempFileName := fmt.Sprintf("%s.json", strings.ReplaceAll(entry, "/", "-"))
				tempAgentPath = filepath.Join(os.TempDir(), tempFileName)
			})

			ginkgo.It("Should compile", func() {
				_, err := fmt.Fprintf(ginkgo.GinkgoWriter, "Compiling agent model: %v\n", entry)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "tempagent path: %v\n", tempAgentPath)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				dirctlArgs := []string{
					"build",
					"--config",
					modelConfigFilePath,
					mountDest,
				}
				runner := testutils.NewDockerRunner(dockerImage, mountString, nil)

				_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "dirctl command: %v %s\n", runner.GetCommandArgs(), strings.Join(dirctlArgs, " "))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				outputBuffer, err := runner.Run(dirctlArgs...)
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), outputBuffer.String())

				err = os.WriteFile(tempAgentPath, outputBuffer.Bytes(), 0644)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})

			ginkgo.It("Agent model should be the expected", func() {
				var expected, compiled map[string]any

				_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "tempagent path: %v\n", tempAgentPath)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

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

				// Filter "created_at" field and extensions
				filter := cmp.FilterPath(func(p cmp.Path) bool {
					// Ensure the path is deep enough
					if len(p) >= 2 {
						if mapStep, ok := p[len(p)-2].(cmp.MapIndex); ok {
							if key, ok := mapStep.Key().Interface().(string); ok && key == "created_at" || key == "extensions" {
								return true // Ignore these paths
							}
						}
					}
					return false // Include all other paths
				}, cmp.Ignore())

				gomega.Expect(expected).To(gomega.BeComparableTo(compiled, filter))
				gomega.Expect(expected["extensions"]).Should(gomega.ConsistOf(compiled["extensions"]))
			})
		})
	}
})
