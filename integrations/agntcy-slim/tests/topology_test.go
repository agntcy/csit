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

// TLS client configuration.
type TLSConfig struct {
	// CA source configuration
	CaSource *CaSource `json:"ca_source,omitempty"`
	// If true, load system CA certificates pool in addition to the certificates
	// configured in this struct.
	IncludeSystemCACertsPool *bool `json:"include_system_ca_certs_pool,omitempty"`
	// In gRPC and HTTP when set to true, this is used to disable the client transport security.
	// (optional, default false)
	Insecure *bool `json:"insecure,omitempty"`
	// InsecureSkipVerify will enable TLS but not verify the server certificate.
	InsecureSkipVerify *bool `json:"insecure_skip_verify,omitempty"`

	// TLS source configuration
	Source *TLSSource `json:"source,omitempty"`
	// The TLS version to use. If not set, the default is "tls1.3".
	// The value must be either "tls1.2" or "tls1.3".
	// (optional)
	TLSVersion *string `json:"tls_version,omitempty"`
}

// CA source configuration
type CaSource struct {
	Type string `json:"type"`
	// For type "file"
	Path *string `json:"path,omitempty"`
	// For type "pem"
	Data *string `json:"data,omitempty"`
	// For type "spire"
	JwtAudiences   *[]string `json:"jwt_audiences,omitempty"`
	SocketPath     *string   `json:"socket_path,omitempty"`
	TargetSpiffeID *string   `json:"target_spiffe_id,omitempty"`
	TrustDomains   *[]string `json:"trust_domains,omitempty"`
}

// TLS source configuration
type TLSSource struct {
	Type string `json:"type"`
	// For type "pem" or "file"
	Cert *string `json:"cert,omitempty"`
	Key  *string `json:"key,omitempty"`
	// For type "spire"
	JwtAudiences   *[]string `json:"jwt_audiences,omitempty"`
	SocketPath     *string   `json:"socket_path,omitempty"`
	TargetSpiffeID *string   `json:"target_spiffe_id,omitempty"`
	TrustDomains   *[]string `json:"trust_domains,omitempty"`
}

type ClientConfig struct {
	Endpoint string    `json:"endpoint"`
	TLS      TLSConfig `json:"tls"`
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
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

					err := k8sHelper.CreateServiceAccount()
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
							Source: &TLSSource{
								Type:       "spire",
								SocketPath: strPtr("unix:/tmp/spire-agent/public/api.sock"),
							},
							CaSource: &CaSource{
								Type:         "spire",
								SocketPath:   strPtr("unix:/tmp/spire-agent/public/api.sock"),
								TrustDomains: &[]string{"example.org"},
							},
						},
					}
					cfgJSON, err := json.Marshal(cfg)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to marshal client config")

					args := append(args, "--slim", string(cfgJSON))
					k8sHelper = k8sHelper.WithArgs(args).WithSpire()

				} else {
					endpoint := fmt.Sprintf("https://agntcy-%s-slim.%s.svc.cluster.local:46357", client.ConnectedTo[0], client.ConnectedTo[0])
					cfg := ClientConfig{
						Endpoint: endpoint,
						TLS: TLSConfig{
							Insecure: boolPtr(true),
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

				time.Sleep(30 * time.Second) // wait for pod to be ready

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
