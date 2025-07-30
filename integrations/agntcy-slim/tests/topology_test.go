// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	"os"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"

	"github.com/agntcy/csit/integrations/agntcy-slim/tests/config"
	"github.com/agntcy/csit/integrations/testutils/k8shelper"
)

type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	CertFile           string `json:"cert_file"`
	KeyFile            string `json:"key_file"`
	CAFile             string `json:"ca_file"`
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
		slimController string
	)

	ginkgo.BeforeEach(func() {

		// Create Kubernetes client
		var err error
		clientset, err = k8shelper.CreateK8sClientSet()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a client")

		namespace = os.Getenv("NAMESPACE")
		slimctlPath = os.Getenv("SLIMCTL_PATH")
		topologyConfig = os.Getenv("TOPOLOGY_CONFIG")
		slimController = os.Getenv("SLIM_CONTROLLER_LOCAL_ENDPOINT")
		// Parse the topology configuration
		config, err := config.ParseTopology(topologyConfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to parse topology configuration")

		// expect topology.ValidateRoutes() to not return an error
		err = config.ValidateRoutes()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to validate routes")

		gomega.Expect(config).NotTo(gomega.BeNil(), "topology configuration should not be nil")
		topology = &config.Topology
	})

	ginkgo.Context("Slim topology test", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			log.Print(slimctlPath)
			// setup routes using the topology configuration

			// wait for SLIM instances to start
			time.Sleep(2000 * time.Millisecond)

			for serverName, server := range topology.Servers {
				for _, route := range server.Routes {
					channelName, destServerName := config.ParseRoute(route)

					log.Printf("Adding route on server %s for channel %s > %s", serverName, channelName, destServerName)

					// add route using slimctl
					gomega.Expect(exec.Command(slimctlPath,
						"route", "add", fmt.Sprintf("org/default/%s/0", channelName),
						"via", fmt.Sprintf("%s-conn-config.json", destServerName),
						"-s", slimController).Run()).To(gomega.Succeed())
				}
			}

		})

		ginkgo.It("Create SLIM client Job(s)", func() {

			for clientName, client := range topology.Clients {

				jobName := clientName
				imageName := client.Image
				envVars := map[string]string{}
				command := client.Cmd
				args := client.Args
				k8sHelper := k8shelper.NewK8sHelper(jobName, namespace, imageName, clientset).WithEnvVars(envVars)

				// expect client.ConnectedTo is not empty
				gomega.Expect(len(client.ConnectedTo)).NotTo(gomega.BeZero(), "client %s must be connected to at least one server", clientName)

				if client.SpireMtls {

					createdConfigMap, err := k8sHelper.CreateConfigMapFromFile("helper.conf", "../components/config/spire/helper.conf")
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdConfigMap)

					// Register cleanup to run after all the spec is done
					ginkgo.DeferCleanup(func(ctx context.Context) {
						//err := k8sHelper.CleanupConfigMap(ctx)
						//gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to delete config map")
					})

					cfg := ClientConfig{
						Endpoint: fmt.Sprintf("https://agntcy-%s:46357", client.ConnectedTo[0]),
						TLS: TLSConfig{
							InsecureSkipVerify: false,
							CertFile:           "/svids/tls.crt",
							KeyFile:            "/svids/tls.key",
							CAFile:             "/svids/svid_bundle.pem",
						},
					}
					cfgJSON, err := json.Marshal(cfg)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to marshal client config")

					args := append(args, "--config", string(cfgJSON))
					// Create a pod with the autogen agent with MTLS from SPIRE
					k8sHelper = k8sHelper.WithCommand([]string{"python", command}).WithArgs(args).WithSpireHelper()

				} else {
					endpoint := fmt.Sprintf("http://agntcy-%s:46357", client.ConnectedTo[0])
					cfg := ClientConfig{
						Endpoint: endpoint,
						TLS: TLSConfig{
							InsecureSkipVerify: true,
						},
					}
					_, err := json.Marshal(cfg)
					gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to marshal client config")

					//args = append(args, "--config", string(cfgJSON))
					args = append(args, "--slim", endpoint)
					k8sHelper = k8sHelper.WithCommand([]string{"python", command}).WithArgs(args)

				}

				_, err := k8sHelper.CreateJob()

				gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("failed to create %s job", clientName))

				// Register cleanup to run after this spec completes
				ginkgo.DeferCleanup(func(ctx context.Context) {
					//err := k8sHelper.CleanupJob(ctx)
					//gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("failed to delete job %s", clientName))
				})

				// Wait for job to be succeded
				//err = k8sHelper.WaitForJobCompletion(k8sTimeOutSeconds * time.Second)
				//gomega.Expect(err).NotTo(gomega.HaveOccurred(), createdJob)
			}
		})
	})
})
