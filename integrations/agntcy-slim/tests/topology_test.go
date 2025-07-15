// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"

	"github.com/agntcy/csit/integrations/agntcy-slim/tests/config"
	"github.com/agntcy/csit/integrations/testutils/k8shelper"
)

var _ = ginkgo.Describe("Agntcy slim topology test", func() {
	var (
		namespace      string
		slimctlPath    string
		topologyConfig string
		topology       *config.Topology
		clientset      kubernetes.Interface
	)

	ginkgo.BeforeEach(func() {

		// Create Kubernetes client
		var err error
		clientset, err = k8shelper.CreateK8sClientSet()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a client")

		namespace = os.Getenv("NAMESPACE")
		slimctlPath = os.Getenv("SLIMCTL_PATH")
		topologyConfig = os.Getenv("TOPOLOGY_CONFIG")
		// Parse the topology configuration
		config, err := config.ParseTopology(topologyConfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to parse topology configuration")

		// expect topology.ValidateRoutes() to not return an error
		err = config.ValidateRoutes()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to validate routes")

		gomega.Expect(config).NotTo(gomega.BeNil(), "topology configuration should not be nil")
		topology = &config.Topology
	})

	ginkgo.Context("Slim sanity test", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {

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
						"-s", fmt.Sprintf("http://agntcy-%s:46358", serverName)).Run()).To(gomega.Succeed())
				}
			}

		})

		ginkgo.It("Create SLIM client Job(s)", func() {

			for clientName, client := range topology.Clients {

				jobName := clientName
				imageName := client.Image
				envVars := map[string]string{}
				command := client.Cmd
				cmdArgs := client.Args
				k8sHelper := k8shelper.NewK8sHelper(jobName, namespace, imageName, clientset).WithEnvVars(envVars)
				args := []string{command}
				args = append(args, cmdArgs...)

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
					args := append(args,
						fmt.Sprintf("--config",
							`{"endpoint": "https://agntcy-%s:46357", 
							"tls": {
								"insecure_skip_verify": false,
								"cert_file": "/svids/tls.crt",
								"key_file": "/svids/tls.key",
								"ca_file": "/svids/svid_bundle.pem"            
							}}`, client.ConnectedTo[0]))
					// Create a pod with the autogen agent with MTLS from SPIRE
					k8sHelper = k8sHelper.WithCommand([]string{"python"}).WithArgs(args).WithSpireHelper()

				} else {

					args = append(args, fmt.Sprintf("--config",
						`{"endpoint": "http://agntcy-%s:46357", 
						"tls": {
							"insecure": true
						}}`, client.ConnectedTo[0]))
					k8sHelper = k8sHelper.WithCommand([]string{"python"}).WithArgs(args)

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
			}
		})
	})
})
