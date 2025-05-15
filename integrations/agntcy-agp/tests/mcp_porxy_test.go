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

var _ = ginkgo.Describe("MCP over AGP test", func() {
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
		llamaindexTimeAgentImage = fmt.Sprintf("%s/agp/llamaindex-time-agent:%s", os.Getenv("IMAGE_REPO"), os.Getenv("LLAMAINDEX_TIME_AGENT_TAG"))
		mcpServerTimeImage = fmt.Sprintf("%s/agp/mcp-server-time:%s", os.Getenv("IMAGE_REPO"), os.Getenv("MCP_SERVER_TIME_TAG"))

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

	ginkgo.Context("AGP native MCP server", func() {
		// The MCP server is AGP-native and works on top of AGP using it as transport.
		// The client can address the MCP server as if it was a normal agent.
		ginkgo.It("Deploy MCP time server with AGP support", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mcp-servert-time",
					Namespace: namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "mcp-server-time",
							Image: mcpServerTimeImage,
							Args: []string{
								"--local-timezone",
								"America/New_York",
								"--config",
								`{"endpoint":"http://agntcy-agp:46357","tls":{"insecure":true}}`,
							},
							// Ports: []corev1.ContainerPort{
							// 	{
							// 		ContainerPort: 8000,
							// 	},
							// },
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
				},
			}
			// Create the pod
			fmt.Println("Creating pod...")
			createdPod, err := clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create MCP time server pod")

			// Wait for pod to be running
			err = waitForPodRunning(clientset, namespace, createdPod.Name, 120*time.Second)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdPod)
		})

		ginkgo.It("Create Llamaindex time agent Job", func() {
			var backOffLimit int32 = 2
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "llamaindex-time-agent",
					Namespace: namespace,
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "llamaindex-time-agent",
									Image: llamaindexTimeAgentImage,
									Args: []string{
										"--city",
										"New York",
										"--config",
										`{"endpoint":"http://agntcy-agp:46357","tls":{"insecure":true}}`,
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

			// Wait for job to be succeded
			err = waitForJobCompletion(clientset, namespace, createdJob.Name, 120*time.Second)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdJob)
		})
	})

	ginkgo.Context("MCP server via MCP proxy", func() {
		// The MCP server works on top of SSE and we can access it using the MCP proxy
		ginkgo.It("Deploy MCP time server without AGP support", func() {
			// TODO pbalogh-sa implement
		})
	})
})

func waitForPodRunning(c kubernetes.Interface, namespace, podName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	watch, err := c.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", podName),
	})
	if err != nil {
		return err
	}
	defer watch.Stop()

	fmt.Println("Waiting for pod to running...")

	for event := range watch.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}

		fmt.Printf("Pod %s status: %s\n", pod.Name, pod.Status.Phase)

		if pod.Status.Phase == corev1.PodRunning {
			return nil
		} else if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return fmt.Errorf("pod ran to completion")
		}
	}

	return fmt.Errorf("watch closed before pod became running")
}

func waitForJobCompletion(c kubernetes.Interface, namespace, jobName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	watch, err := c.BatchV1().Jobs(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", jobName),
	})
	if err != nil {
		return err
	}
	defer watch.Stop()

	fmt.Println("Waiting for job to complete...")

	for event := range watch.ResultChan() {
		job, ok := event.Object.(*batchv1.Job)
		if !ok {
			continue
		}

		fmt.Printf("Job %s status: Active=%d, Succeeded=%d, Failed=%d\n",
			job.Name, job.Status.Active, job.Status.Succeeded, job.Status.Failed)

		// Check job conditions for completion or failure
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
				fmt.Printf("Job %s completed successfully\n", job.Name)
				return nil
			} else if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
				return fmt.Errorf("job failed: %s", condition.Reason)
			}
		}
	}

	return fmt.Errorf("watch closed before job completed")
}
