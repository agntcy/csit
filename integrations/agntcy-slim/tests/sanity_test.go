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
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/agntcy/csit/integrations/testutils/k8shelper"
)

var _ = ginkgo.Describe("Agntcy slim sanity test", func() {
	var (
		langchainImage         string
		autogenImage           string
		azure_openapi_api_key  string
		azure_openapi_endpoint string
		namespace              string
		slimConfig             string
		clientset              kubernetes.Interface
		dynamicClient          dynamic.Interface
	)

	ginkgo.BeforeEach(func() {
		// Setup test images
		langchainImage = fmt.Sprintf("%s/csit/test-langchain-agent:%s", os.Getenv("IMAGE_REPO"), os.Getenv("LANGCHAIN_APP_TAG"))
		autogenImage = fmt.Sprintf("%s/csit/test-autogen-agent:%s", os.Getenv("IMAGE_REPO"), os.Getenv("AUTOGEN_APP_TAG"))

		slimConfig = os.Getenv("SLIM_CONFIG")

		// Setup LLM credentials
		azure_openapi_api_key = os.Getenv("AZURE_OPENAI_API_KEY")
		azure_openapi_endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")

		// Create Kubernetes client
		var err error
		clientset, err = k8shelper.CreateK8sClientSet()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a client")

		dynamicClient, err = k8shelper.CreateDynamicK8sClient()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a dynamic client")

		namespace = os.Getenv("NAMESPACE")
	})

	ginkgo.Context("Slim sanity test", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			podName := "autogen-agent"
			k8sHelper := k8shelper.NewK8sHelper(podName, namespace, autogenImage, clientset, dynamicClient).WithEnvVars(
				map[string]string{
					"AZURE_OPENAI_ENDPOINT": azure_openapi_endpoint,
					"AZURE_OPENAI_API_KEY":  azure_openapi_api_key,
				})

			switch slimConfig {
			case "base":
				k8sHelper = k8sHelper.WithCommand([]string{"python"}).WithArgs([]string{
					"-u",
					"autogen_agent.py",
					"--config",
					`{"endpoint": "http://agntcy-slim:46357",
					"tls": {
						"insecure": true			    	
					}}`,
				})

			case "mtls":
				// Create a pod with the autogen agent with MTLS cert from secret
				k8sHelper = k8sHelper.WithCommand([]string{"python"}).WithArgs([]string{
					"-u",
					"autogen_agent.py",
					"--config",
					`{"endpoint": "https://agntcy-slim:46357",
				"tls": {
			    	"insecure_skip_verify": false,
			    	"cert_file": "/etc/certs/tls.crt",
			    	"key_file": "/etc/certs/tls.key",
			    	"ca_file": "/etc/certs/ca.crt"
				}}`,
				}).WithCertSecret()

			case "spire":

				createdConfigMap, err := k8sHelper.CreateConfigMapFromFile("helper.conf", "../components/config/spire/helper.conf")
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdConfigMap)

				// Register cleanup to run after all the spec is done
				ginkgo.DeferCleanup(func(ctx context.Context) {
					err := k8sHelper.CleanupConfigMap(ctx)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete config map")
				})

				// Create a pod with the autogen agent with MTLS from SPIRE
				k8sHelper = k8sHelper.WithCommand([]string{"python"}).WithArgs([]string{
					"-u",
					"autogen_agent.py",
					"--config",
					`{"endpoint": "https://agntcy-slim:46357",
				"tls": {
			    	"insecure_skip_verify": false,
			    	"cert_file": "/svids/tls.crt",
			    	"key_file": "/svids/tls.key",
			    	"ca_file": "/svids/svid_bundle.pem"
				}}`,
				}).WithSpire()
			default:
				ginkgo.Fail(fmt.Sprintf("Unknown SLIM_CONFIG value: %s", slimConfig))
			}

			// Create a pod with the autogen agent with MTLS cert from secret
			createdPod, err := k8sHelper.CreatePod()

			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create MCP time server pod")

			// Register cleanup to run after all the spec is done
			ginkgo.DeferCleanup(func(ctx context.Context) {
				err := k8sHelper.CleanupPod(ctx)
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete pod")
			})

			err = k8sHelper.WaitForPodRunning(k8sTimeOutSeconds * time.Second)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdPod)
		})

		ginkgo.It("Create langchain agent Job", func() {
			jobName := "langchain-agent"
			k8sHelper := k8shelper.NewK8sHelper(jobName, namespace, langchainImage, clientset, dynamicClient).WithEnvVars(map[string]string{
				"AZURE_OPENAI_ENDPOINT": azure_openapi_endpoint,
				"AZURE_OPENAI_API_KEY":  azure_openapi_api_key,
			})

			switch slimConfig {
			case "base":
				k8sHelper = k8sHelper.WithCommand([]string{"python"}).WithArgs([]string{
					"-u",
					"langchain_agent.py",
					"-m",
					"Budapest",
					"--config",
					`{"endpoint": "http://agntcy-slim:46357", 
					"tls": {
						"insecure": true
					}}`,
				})

			case "mtls":
				// Create a pod with the autogen agent with MTLS cert from secret
				k8sHelper = k8sHelper.WithCommand([]string{"python"}).WithArgs([]string{
					"-u",
					"langchain_agent.py",
					"-m",
					"Budapest",
					"--config",
					`{"endpoint": "https://agntcy-slim:46357", 
					"tls": {
						"insecure_skip_verify": false,
						"cert_file": "/etc/certs/tls.crt",
						"key_file": "/etc/certs/tls.key",
						"ca_file": "/etc/certs/ca.crt"            
					}}`,
				}).WithCertSecret()

			case "spire":

				createdConfigMap, err := k8sHelper.CreateConfigMapFromFile("helper.conf", "../components/config/spire/helper.conf")
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdConfigMap)

				// Register cleanup to run after all the spec is done
				ginkgo.DeferCleanup(func(ctx context.Context) {
					err := k8sHelper.CleanupConfigMap(ctx)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete config map")
				})

				// Create a pod with the autogen agent with MTLS from SPIRE
				k8sHelper = k8sHelper.WithCommand([]string{"python"}).WithArgs([]string{
					"-u",
					"langchain_agent.py",
					"-m",
					"Budapest",
					"--config",
					`{"endpoint": "https://agntcy-slim:46357", 
					"tls": {
						"insecure_skip_verify": false,
						"cert_file": "/svids/tls.crt",
						"key_file": "/svids/tls.key",
						"ca_file": "/svids/svid_bundle.pem"            
					}}`,
				}).WithSpire()

			default:
				ginkgo.Fail(fmt.Sprintf("Unknown SLIM_CONFIG value: %s", slimConfig))
			}

			createdJob, err := k8sHelper.CreateJob()

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
