// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/agntcy/csit/integrations/testutils"
)

var _ = ginkgo.Describe("Agntcy gateway tests", func() {
	var (
		dockerImage            string
		azure_openapi_api_key  string
		azure_openapi_endpoint string
		runner                 testutils.Runner
	)

	ginkgo.BeforeEach(func() {
		dockerImage = fmt.Sprintf("%s/csit/test-langchain-agent:%s", os.Getenv("IMAGE_REPO"), os.Getenv("TEST_APP_TAG"))
		azure_openapi_api_key = os.Getenv("AZURE_OPENAI_API_KEY")
		azure_openapi_endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	})

	ginkgo.Context("agent gateway", func() {
		ginkgo.It("simple agent gateway test", func() {
			langchainAgentArgs := []string{
				"run",
				"python",
				"langchain_agent.py",
				"-m",
				"Budapest",
				"-g",
			}

			gwHost := "http://127.0.0.1:46357"
			if runtime.GOOS != "linux" {
				gwHost = "http://host.docker.internal:46357"
			}

			langchainAgentArgs = append(langchainAgentArgs, gwHost)

			envVars := map[string]string{
				"AZURE_OPENAI_API_KEY":  azure_openapi_api_key,
				"AZURE_OPENAI_ENDPOINT": azure_openapi_endpoint,
			}

			var err error

			switch os.Getenv("RUNNER_TYPE") {
			// NOTE: No binary release for agp yet
			// case "local":
			// 	runner, err = testutils.NewRunner(testutils.RunnerTypeLocal, testutils.WithEnvVars(envVars))
			default:
				runner, err = testutils.NewRunner(testutils.RunnerTypeDocker,
					testutils.WithDockerCmd("docker"),
					testutils.WithDockerImage(dockerImage),
					testutils.WithDockerArgs([]string{"run"}),
					testutils.WithEnvVars(envVars),
				)
			}

			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			_, err = runner.Run("poetry", langchainAgentArgs...)
			if err != nil {
				exitErr, ok := err.(*exec.ExitError)
				if ok {
					err = fmt.Errorf("%s, stderr:%s", exitErr.String(), string(exitErr.Stderr))
				}
			}

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})
