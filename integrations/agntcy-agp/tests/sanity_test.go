// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
		autogenImage = fmt.Sprintf("%s/csit/test-langchain-agent:%s", os.Getenv("IMAGE_REPO"), os.Getenv("LANGCHAIN_APP_TAG"))

		// Setup LLM credentials
		azure_openapi_api_key = os.Getenv("AZURE_OPENAI_API_KEY")
		azure_openapi_endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")

		// Create Kubernetes client
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to load kubeconfig")
		clientset, err = kubernetes.NewForConfig(config)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a client")

		namespace = os.Getenv("NAMESPACE")
	})

	ginkgo.Context("AGP sanity test", ginkgo.Ordered, func() {
		// The MCP server is AGP-native and works on top of AGP using it as transport.
		// The client can address the MCP server as if it was a normal agent.
		ginkgo.BeforeAll(func() {
			podName := "autogen-agent"
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  podName,
							Image: autogenImage,
							Args: []string{
								"-g",
								"http://agntcy-agp:46357",
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
				},
			}
			// Create the pod
			fmt.Println("Creating pod...")
			createdPod, err := clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create MCP time server pod")

			// Register cleanup to run after all the spec is done
			ginkgo.DeferCleanup(func(ctx context.Context) {
				err := clientset.CoreV1().Pods(namespace).Delete(ctx, createdPod.Name, metav1.DeleteOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete pod")
			})

			// Wait for pod to be running
			err = waitForPodRunning(clientset, namespace, createdPod.Name, 300*time.Second)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdPod)
		})

		ginkgo.It("Create langchain agent Job", func() {
			var backOffLimit int32 = 2
			jobName := "langchain-agent"
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: namespace,
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  jobName,
									Image: langchainImage,
									Args: []string{
										"-g",
										"http://agntcy-agp:46357",
									},
									Env: []corev1.EnvVar{
										{
											Name:  "AZURE_OPENAI_ENDPOINT",
											Value: azure_openapi_endpoint,
										},
										{
											Name:  "AZURE_OPENAI_API_KEY",
											Value: azure_openapi_api_key,
										},
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
					BackoffLimit: &backOffLimit,
				},
			}

			// Create the job
			fmt.Println("Creating job...")
			createdJob, err := clientset.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create Llamaindext time agent job")

			// Register cleanup to run after this spec completes
			ginkgo.DeferCleanup(func(ctx context.Context) {
				deletePolicy := metav1.DeletePropagationBackground
				err := clientset.BatchV1().Jobs(namespace).Delete(ctx, createdJob.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete job")
			})

			// Wait for job to be succeded
			err = waitForJobCompletion(clientset, namespace, createdJob.Name, 300*time.Second)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdJob)
		})
	})
})

// 	ginkgo.Context("agent gateway", func() {
// 		ginkgo.It("agent gateway sanity test", func() {
// 			langchainAgentArgs := []string{
// 				"run",
// 				"python",
// 				"langchain_agent.py",
// 				"-m",
// 				"Budapest",
// 				"-g",
// 			}

// 			gwHost := "http://127.0.0.1:46357"
// 			if runtime.GOOS != "linux" {
// 				gwHost = "http://host.docker.internal:46357"
// 			}

// 			langchainAgentArgs = append(langchainAgentArgs, gwHost)

// 			envVars := map[string]string{
// 				"AZURE_OPENAI_API_KEY":  azure_openapi_api_key,
// 				"AZURE_OPENAI_ENDPOINT": azure_openapi_endpoint,
// 			}

// 			var err error

// 			switch os.Getenv("RUNNER_TYPE") {
// 			// NOTE: No binary release for agp yet
// 			// case "local":
// 			// 	runner, err = testutils.NewRunner(testutils.RunnerTypeLocal, testutils.WithEnvVars(envVars))
// 			default:
// 				runner, err = testutils.NewRunner(testutils.RunnerTypeDocker,
// 					testutils.WithDockerCmd("docker"),
// 					testutils.WithDockerImage(dockerImage),
// 					testutils.WithDockerArgs([]string{"run"}),
// 					testutils.WithEnvVars(envVars),
// 				)
// 			}

// 			gomega.Expect(err).NotTo(gomega.HaveOccurred())

// 			_, err = runner.Run("poetry", langchainAgentArgs...)
// 			if err != nil {
// 				exitErr, ok := err.(*exec.ExitError)
// 				if ok {
// 					err = fmt.Errorf("%s, stderr:%s", exitErr.String(), string(exitErr.Stderr))
// 				}
// 			}

// 			gomega.Expect(err).NotTo(gomega.HaveOccurred())
// 		})
// 	})
// })
