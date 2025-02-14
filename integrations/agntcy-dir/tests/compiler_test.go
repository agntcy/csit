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

		testdataDir, err := filepath.Abs(filepath.Join(examplesDir, "destdata"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		expectedAgentModelFile = filepath.Join(testdataDir, "expected_agent.json")
	})

	ginkgo.Context("agent compilation", func() {
		ginkgo.It("should compile an agent", func() {

			dirctlArgs := []string{
				"build",
				"--name=marketing-strategy",
				"--version=v1.0.0",
				"--artifact-type=docker-image",
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

			// Filter "created_at" and "security.signature" fields
			filter := cmp.FilterPath(func(p cmp.Path) bool {
				// Ensure the path is deep enough
				if len(p) >= 7 {
					parentStep := p[len(p)-3]
					currentStep := p[len(p)-1]
					if mapStep, ok := p[len(p)-7].(cmp.MapIndex); ok {
						if key, ok := mapStep.Key().Interface().(string); ok && key == "extensions" {
							// Check if the parentStep is a map lookup with key "specs"
							if parentMapIndex, ok := parentStep.(cmp.MapIndex); ok {
								if parentKey, ok := parentMapIndex.Key().Interface().(string); ok && parentKey == "specs" {
									// Check if the currentStep is a map lookup with key "created_at" and "signature"
									if currentMapIndex, ok := currentStep.(cmp.MapIndex); ok {
										if currentKey, ok := currentMapIndex.Key().Interface().(string); ok && (currentKey == "created_at" || currentKey == "signature") {
											return true // Ignore these paths
										}
									}
								}
							}
						}
					}
				}
				return false // Include all other paths
			}, cmp.Ignore())

			// Check the compiled agent model without extensions field
			gomega.Expect(expected).To(gomega.BeComparableTo(compiled, filter))
		})
	})
})
