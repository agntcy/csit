// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"os"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"

	"github.com/agntcy/csit/integrations/testutils/k8shelper"
)

var _ = ginkgo.Describe("Agntcy gateway sanity test", func() {
	var (
		langchainImage         string
		autogenImage           string
		azure_openapi_api_key  string
		azure_openapi_endpoint string
		namespace              string
		clientset              kubernetes.Interface
	)

	ginkgo.BeforeEach(func() {
		// Setup test images
		langchainImage = fmt.Sprintf("%s/csit/test-langchain-agent:%s", os.Getenv("IMAGE_REPO"), os.Getenv("LANGCHAIN_APP_TAG"))
		autogenImage = fmt.Sprintf("%s/csit/test-autogen-agent:%s", os.Getenv("IMAGE_REPO"), os.Getenv("AUTOGEN_APP_TAG"))

		// Setup LLM credentials
		azure_openapi_api_key = os.Getenv("AZURE_OPENAI_API_KEY")
		azure_openapi_endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")

		// Create Kubernetes client
		var err error
		clientset, err = k8shelper.CreateK8sClientSet()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a client")

		namespace = os.Getenv("NAMESPACE")
	})

	ginkgo.Context("AGP sanity test", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			podName := "autogen-agent"
			k8sHelper := k8shelper.NewK8sHelper(podName, namespace, autogenImage, clientset)

			createdPod, err := k8sHelper.WithEnvVars(map[string]string{
				"AZURE_OPENAI_ENDPOINT": azure_openapi_endpoint,
				"AZURE_OPENAI_API_KEY":  azure_openapi_api_key,
			}).WithCommand([]string{"python"}).WithArgs([]string{
				"autogen_agent.py",
				"-g",
				"http://agntcy-agp:46357",
			}).CreatePod()

			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create MCP time server pod")

			// Register cleanup to run after all the spec is done
			ginkgo.DeferCleanup(func(ctx context.Context) {
				err := k8sHelper.CleanupPod(ctx)
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete pod")
			})

			// Wait for pod to be running
			err = k8sHelper.WaitForPodRunning(k8sTimeOutSeconds * time.Second)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdPod)
		})

		ginkgo.It("Create langchain agent Job", func() {
			jobName := "langchain-agent"
			k8sHelper := k8shelper.NewK8sHelper(jobName, namespace, langchainImage, clientset)

			createdJob, err := k8sHelper.WithEnvVars(map[string]string{
				"AZURE_OPENAI_ENDPOINT": azure_openapi_endpoint,
				"AZURE_OPENAI_API_KEY":  azure_openapi_api_key,
			}).WithCommand([]string{"python"}).WithArgs([]string{
				"langchain_agent.py",
				"-m",
				"Budapest",
				"-g",
				"http://agntcy-agp:46357",
			}).CreateJob()

			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create Llamaindext time agent job")

			// Register cleanup to run after this spec completes
			ginkgo.DeferCleanup(func(ctx context.Context) {
				err := k8sHelper.CleanupJob(ctx)
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete job")
			})

			// Wait for job to be succeded
			err = k8sHelper.WaitForJobCompletion(k8sTimeOutSeconds * time.Second)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdJob)
		})
	})
})
