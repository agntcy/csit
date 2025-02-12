// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"os"
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
	)

	ginkgo.BeforeEach(func() {
		dockerImage = fmt.Sprintf("%s/csit/test-langchain-agent:%s", os.Getenv("IMAGE_REPO"), os.Getenv("TEST_APP_TAG"))
		azure_openapi_api_key = os.Getenv("AZURE_OPENAI_API_KEY")
		azure_openapi_endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	})

	ginkgo.Context("agent gateway", func() {
		ginkgo.It("simple agent gateway test", func() {
			langchainAgentArgs := []string{
				"poetry",
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
			runner := testutils.NewDockerRunner(dockerImage, "", envVars)
			outputBuffer, err := runner.Run(langchainAgentArgs...)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), outputBuffer.String())
		})
	})
})
