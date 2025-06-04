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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/agntcy/csit/integrations/testutils/k8shelper"
)

var _ = ginkgo.Describe("MCP over Slim test", func() {
	var (
		llamaindexTimeAgentImage string
		mcpServerTimeImage       string
		azure_openapi_api_key    string
		azure_openapi_endpoint   string
		clientset                kubernetes.Interface
		namespace                string
	)

	ginkgo.BeforeEach(func() {
		// Setup MCP server test images
		llamaindexTimeAgentImage = fmt.Sprintf("%s/slim/llamaindex-time-agent:%s", os.Getenv("IMAGE_REPO"), os.Getenv("LLAMAINDEX_TIME_AGENT_TAG"))
		mcpServerTimeImage = fmt.Sprintf("%s/slim/mcp-server-time:%s", os.Getenv("IMAGE_REPO"), os.Getenv("MCP_SERVER_TIME_TAG"))

		// Setup LLM credentials
		azure_openapi_api_key = os.Getenv("AZURE_OPENAI_API_KEY")
		azure_openapi_endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")

		// Create Kubernetes client
		var err error
		clientset, err = k8shelper.CreateK8sClientSet()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a client")

		namespace = os.Getenv("NAMESPACE")
	})

	ginkgo.Context("Slim native MCP server", ginkgo.Ordered, func() {
		// The MCP server is Slim-native and works on top of Slim using it as transport.
		// The client can address the MCP server as if it was a normal agent.
		ginkgo.BeforeAll(func() {
			podName := "mcp-server"
			k8sHelper := k8shelper.NewK8sHelper(podName, namespace, mcpServerTimeImage, clientset)

			createdPod, err := k8sHelper.WithArgs([]string{
				"--local-timezone",
				"America/New_York",
				"--config",
				`{"endpoint":"http://agntcy-slim:46357","tls":{"insecure":true}}`,
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

		ginkgo.It("Create Llamaindex time agent Job", func() {
			jobName := "llamaindex-time-agent"
			k8sHelper := k8shelper.NewK8sHelper(jobName, namespace, llamaindexTimeAgentImage, clientset)

			createdJob, err := k8sHelper.WithEnvVars(map[string]string{
				"AZURE_OPENAI_ENDPOINT": azure_openapi_endpoint,
				"AZURE_OPENAI_API_KEY":  azure_openapi_api_key,
			}).WithArgs([]string{
				"--city",
				"New York",
				"--config",
				`{"endpoint":"http://agntcy-slim:46357","tls":{"insecure":true}}`,
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

	ginkgo.Context("MCP server via MCP proxy", ginkgo.Ordered, func() {
		// The MCP server works on top of SSE and we can access it using the MCP proxy
		ginkgo.BeforeAll(func() {
			podName := "mcp-server-proxy"
			k8sHelper := k8shelper.NewK8sHelper(podName, namespace, mcpServerTimeImage, clientset)

			createdPod, err := k8sHelper.WithArgs([]string{
				"--local-timezone",
				"America/New_York",
				"--config",
				`{"endpoint":"http://agntcy-slim:46357","tls":{"insecure":true}}`,
				"--transport",
				"sse",
			}).WithContainerPorts([]int32{
				8000,
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

			createdService, err := k8sHelper.CreateService("mcp-server")
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create MCP time server pod")

			// Register cleanup to run after all the spec is done
			ginkgo.DeferCleanup(func(ctx context.Context) {
				err := clientset.CoreV1().Services(namespace).Delete(ctx, createdService.Name, metav1.DeleteOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete service")
			})
		})

		ginkgo.It("Create Llamaindex time agent Job", func() {
			jobName := "llamaindex-time-agent"
			k8sHelper := k8shelper.NewK8sHelper(jobName, namespace, llamaindexTimeAgentImage, clientset)

			createdJob, err := k8sHelper.WithEnvVars(map[string]string{
				"AZURE_OPENAI_ENDPOINT": azure_openapi_endpoint,
				"AZURE_OPENAI_API_KEY":  azure_openapi_api_key,
			}).WithArgs([]string{
				"--city",
				"New York",
				"--config",
				`{"endpoint":"http://agntcy-slim:46357","tls":{"insecure":true}}`,
				"--mcp-server-organization",
				"org",
				"--mcp-server-namespace",
				"mcp",
				"--mcp-server-name",
				"proxy",
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
