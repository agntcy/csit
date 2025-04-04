// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		runner                 testutils.Runner
	)

	ginkgo.BeforeEach(func() {
		examplesDir := "../examples/"
		testDataPath, err := filepath.Abs(filepath.Join(examplesDir, "dir/e2e/testdata"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		tempAgentPath = filepath.Join(os.TempDir(), "agent.json")
		dockerImage = fmt.Sprintf("%s/dir-ctl:%s", os.Getenv("IMAGE_REPO"), os.Getenv("DIRECTORY_IMAGE_TAG"))

		if os.Getenv("RUNNER_TYPE") == "local" {
			mountDest = testDataPath
		} else {
			mountDest = "/testdata"
			mountString = fmt.Sprintf("%s:%s", testDataPath, mountDest)
		}

		modelConfigFilePath = filepath.Join(mountDest, "build.config.yml")
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

			var err error

			switch os.Getenv("RUNNER_TYPE") {
			case "local":
				runner, err = testutils.NewRunner(testutils.RunnerTypeLocal, nil)
			default:
				runner, err = testutils.NewRunner(testutils.RunnerTypeDocker,
					testutils.WithDockerCmd("docker"),
					testutils.WithDockerArgs([]string{"run", "-v", mountString}),
					testutils.WithDockerImage(dockerImage),
				)
			}

			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "dirctl command: %v %s\n", runner.GetDockerCommandAndArgs(), strings.Join(dirctlArgs, " "))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cmdOutput, err := runner.Run("dirctl", dirctlArgs...)
			if err != nil {
				exitErr, ok := err.(*exec.ExitError)
				if ok {
					err = fmt.Errorf("%s, stderr:%s", exitErr.String(), string(exitErr.Stderr))
				}
			}

			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = os.WriteFile(tempAgentPath, []byte(cmdOutput), 0644)
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
})
