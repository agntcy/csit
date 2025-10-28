// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"os"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/agntcy/csit/integrations/agntcy-slim/tests/config"
	"github.com/agntcy/csit/integrations/testutils/k8shelper"
)

type TLSConfig struct {
	Insecure           bool   `json:"insecure,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
	CAFile             string `json:"ca_file,omitempty"`
}

type ClientConfig struct {
	Endpoint string    `json:"endpoint"`
	TLS      TLSConfig `json:"tls"`
}

// ...

var _ = ginkgo.Describe("Agntcy slim topology test", func() {
	var (
		namespace      string
		slimctlPath    string
		topologyConfig string
		topology       *config.Topology
		clientset      kubernetes.Interface
		dynamicClient  dynamic.Interface
		//slimController string
	)

	ginkgo.BeforeEach(func() {

		// Create Kubernetes client
		var err error
		clientset, err = k8shelper.CreateK8sClientSet()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a client")

		dynamicClient, err = k8shelper.CreateDynamicK8sClient()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a dynamic client")

		namespace = os.Getenv("NAMESPACE")
		slimctlPath = os.Getenv("SLIMCTL_PATH")
		topologyConfig = os.Getenv("TOPOLOGY_CONFIG")
		//slimController = os.Getenv("SLIM_CONTROLLER_LOCAL_ENDPOINT")
		// Parse the topology configuration
		config, err := config.ParseTopology(topologyConfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to parse topology configuration")

		gomega.Expect(config).NotTo(gomega.BeNil(), "topology configuration should not be nil")
		topology = &config.Topology
	})

	ginkgo.Context("Slim topology test", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			log.Print(slimctlPath)
		})

		ginkgo.It("Create SLIM client Pods", func() {
			// alphanumerically order topology.Clients by key
			clientNames := make([]string, 0, len(topology.Clients))
			for name := range topology.Clients {
				clientNames = append(clientNames, name)
			}
			sort.Strings(clientNames)

			logWatchers := make(map[string]*k8shelper.LogWatcher)

			for _, clientName := range clientNames {
				client := topology.Clients[clientName]

				jobName := clientName
				imageName := client.Image
				envVars := map[string]string{
					"PYTHONUNBUFFERED": "1",
				}
				args := client.Args
				k8sHelper := k8shelper.NewK8sHelper(jobName, namespace, imageName, clientset, dynamicClient).WithEnvVars(envVars)

				// expect client.ConnectedTo is not empty
				gomega.Expect(len(client.ConnectedTo)).NotTo(gomega.BeZero(), "client %s must be connected to at least one server", clientName)

				if client.SpireMtls {
					createdConfigMap, err := k8sHelper.CreateConfigMapFromFile("helper.conf", "../components/config/spire/helper.conf")
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdConfigMap)
					// Register cleanup to run after all the spec is done
					ginkgo.DeferCleanup(func(ctx context.Context) {
						err := k8sHelper.CleanupConfigMap(ctx)
						gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete config map")
					})

					err = k8sHelper.CreateServiceAccount()
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create service account")
					// Register cleanup to run after all the spec is done
					ginkgo.DeferCleanup(func(ctx context.Context) {
						err := k8sHelper.CleanupServiceAccount()
						gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete service account")
					})

					err = k8sHelper.CreateClusterSPIFFEID()
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create spiffee ID")
					// Register cleanup to run after all the spec is done
					ginkgo.DeferCleanup(func(ctx context.Context) {
						err := k8sHelper.CleanupClusterSPIFFEID()
						gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete spiffee ID")
					})

					cfg := ClientConfig{
						Endpoint: fmt.Sprintf("https://agntcy-%s-slim.%s.svc.cluster.local:46357", client.ConnectedTo[0], client.ConnectedTo[0]),
						TLS: TLSConfig{
							InsecureSkipVerify: false,
							CertFile:           "/svids/tls.crt",
							KeyFile:            "/svids/tls.key",
							CAFile:             "/svids/svid_bundle.pem",
						},
					}
					cfgJSON, err := json.Marshal(cfg)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to marshal client config")

					args := append(args, "--slim", string(cfgJSON))
					k8sHelper = k8sHelper.WithArgs(args).WithSpireHelper()

				} else {
					endpoint := fmt.Sprintf("https://agntcy-%s-slim.%s.svc.cluster.local:46357", client.ConnectedTo[0], client.ConnectedTo[0])
					cfg := ClientConfig{
						Endpoint: endpoint,
						TLS: TLSConfig{
							Insecure: true,
						},
					}
					cfgJSON, err := json.Marshal(cfg)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to marshal client config")

					args = append(args, "--slim", string(cfgJSON))
					k8sHelper = k8sHelper.WithArgs(args)
				}

				createdPod, err := k8sHelper.CreatePod()
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("failed to create %s job", clientName))

				// Wait for pod to be running
				err = k8sHelper.WaitForPodRunning(k8sTimeOutSeconds * time.Second)
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdPod)

				time.Sleep(1000 * time.Millisecond) // wait for pod to be ready

				if client.AssertFor != "" {
					log.Printf("Starting log watcher for client %s with assertFor: %s", clientName, client.AssertFor)
					// Start watching logs for a specific assertString
					logWatcher, err := k8sHelper.WatchLogsForString(client.AssertFor)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to start log watcher")
					logWatchers[clientName] = logWatcher
					// Register cleanup for log watcher
					ginkgo.DeferCleanup(func() {
						logWatcher.Stop()
					})

				} else {
					log.Printf("No assertFor defined for client %s, skipping log watcher", clientName)
				}

				// Register cleanup to run after this spec completes
				ginkgo.DeferCleanup(func(ctx context.Context) {
					err := k8sHelper.CleanupPod(ctx)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("failed to delete pod %s", clientName))
				})

			}

			// Wait for all pods to show the expected log message
			for clientName, logWatcher := range logWatchers {
				ginkgo.By(fmt.Sprintf("Waiting for %s to show %s message", clientName, logWatcher.GetSearchString()))

				// Wait for the search string with a timeout
				done := make(chan bool, 1)
				var foundLine string
				var waitErr error

				go func() {
					foundLine, waitErr = logWatcher.Wait()
					done <- true
				}()

				select {
				case <-done:
					if waitErr != nil {
						// Print collected logs for debugging
						logs := logWatcher.GetLogs()
						fmt.Printf("Collected logs for %s:\n", clientName)
						for _, log := range logs {
							fmt.Printf("  %s\n", log)
						}
						gomega.Expect(waitErr).NotTo(gomega.HaveOccurred(), fmt.Sprintf("failed to find search string in %s logs", clientName))
					}
					fmt.Printf("Found expected message in %s: %s\n", clientName, foundLine)
				case <-time.After(30 * time.Second): // 30 second timeout
					logs := logWatcher.GetLogs()
					fmt.Printf("Timeout waiting for search string in %s. Collected logs:\n", clientName)
					for _, log := range logs {
						fmt.Printf("  %s\n", log)
					}
					gomega.Expect(false).To(gomega.BeTrue(), fmt.Sprintf("Timeout waiting for search string '%s' in %s logs", logWatcher.GetSearchString(), clientName))
				}
			}
		})
	})
})
