// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/agntcy/csit/integrations/agntcy-slim/topology/tests/config"
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

const (
	slimClientConfigVolumeName = "slim-client-config"
	slimClientConfigMapKey     = "slim-config.json"
	slimClientConfigMountDir   = "/etc/slim"

	// p2pSharedSecret is a non-sensitive test fixture required by the
	// slim-bindings-p2p example CLI (not a credential to any real service).
	p2pSharedSecret = "secret1234secret1234secret123412"

	// Distinct per-sender payloads let a receiver's log reveal the source.
	msgFromAlice     = "hi-from-alice"
	msgFromBob       = "hi-from-bob"
	msgFromBobVerify = "hi-bob-restart"

	// Segments (routing domains) created at runtime with slimctl.
	segB  = "seg-b"
	segC  = "seg-c"
	segBC = "seg-bc"

	// Continuous sender pacing: each batch sends this many messages, then the
	// shell loop sleeps and starts another batch, so traffic continues across
	// node / control-plane restarts.
	p2pIterationsPerBatch = 20
	p2pBatchSleepSeconds  = 3
)

func slimClientConfigFilePath() string {
	return slimClientConfigMountDir + "/" + slimClientConfigMapKey
}

func slimClientConfigVolumes(configMapName string) ([]corev1.Volume, []corev1.VolumeMount) {
	return []corev1.Volume{{
			Name: slimClientConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
				},
			},
		}}, []corev1.VolumeMount{{
			Name:      slimClientConfigVolumeName,
			MountPath: slimClientConfigMountDir,
		}}
}

// clientPod captures the subset of k8shelper methods the scenarios need after a
// client pod has been deployed.
type clientPod interface {
	WaitForStringWithTimeout(searchString string, timeout time.Duration) (bool, string, error)
	CleanupPod(ctx context.Context) error
	CleanupConfigMap(ctx context.Context) error
}

// p2pClientSpec describes a single slim-bindings-p2p client. When Remote is set
// the client is a (continuous) sender; otherwise it is a passive receiver.
type p2pClientSpec struct {
	Name      string // pod + configmap name
	Cluster   string // target cluster whose slim node this client connects to
	LocalName string // this client's SLIM name (org/ns/<name>)
	Remote    string // destination SLIM name (senders only)
	Message   string // payload (senders only)
}

func clusterEndpoint(cluster string) string {
	return fmt.Sprintf("https://agntcy-%s-slim.%s.svc.cluster.local:46357", cluster, cluster)
}

func clusterDeploymentName(cluster string) string {
	return fmt.Sprintf("agntcy-%s-slim", cluster)
}

// deployP2PClient creates the config map + pod for a p2p client and waits for it
// to be running. Senders run under a shell retry loop so they keep producing
// traffic across restarts.
func deployP2PClient(clientset kubernetes.Interface, dynamicClient dynamic.Interface, namespace, image string, spec p2pClientSpec) clientPod {
	h := k8shelper.NewK8sHelper(spec.Name, namespace, image, clientset, dynamicClient).
		WithEnvVars(map[string]string{"PYTHONUNBUFFERED": "1"})

	cfg := ClientConfig{
		Endpoint: clusterEndpoint(spec.Cluster),
		TLS:      TLSConfig{Insecure: boolPtr(true)},
	}
	cfgJSON, err := json.Marshal(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to marshal client config")

	_, err = h.CreateConfigMap(slimClientConfigMapKey, string(cfgJSON))
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create slim config map for %s", spec.Name)

	vols, mounts := slimClientConfigVolumes(spec.Name)
	h = h.WithVolumes(vols).WithWithVolumeMounts(mounts)

	cliArgs := []string{
		"slim-bindings-p2p",
		"--local", spec.LocalName,
		"--shared-secret", p2pSharedSecret,
		"--slim-config", slimClientConfigFilePath(),
	}
	if spec.Remote != "" {
		cliArgs = append(cliArgs,
			"--remote", spec.Remote,
			"--message", spec.Message,
			"--iterations", strconv.Itoa(p2pIterationsPerBatch),
		)
		// Wrap in a shell loop so the sender keeps sending across restarts.
		script := fmt.Sprintf("while true; do %s; sleep %d; done",
			strings.Join(cliArgs, " "), p2pBatchSleepSeconds)
		h = h.WithCommand([]string{"/bin/sh", "-c"}).WithArgs([]string{script})
	} else {
		h = h.WithArgs(cliArgs)
	}

	_, err = h.CreatePod()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to create pod %s", spec.Name)

	err = h.WaitForPodRunning(k8sTimeOutSeconds * time.Second)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "pod %s did not become ready", spec.Name)

	return h
}

// linksBetween counts links whose endpoints connect clusterA and clusterB. Node
// IDs equal pod names, which contain the cluster name.
func linksBetween(links []linkEntry, clusterA, clusterB string) int {
	n := 0
	for _, l := range links {
		srcA := strings.Contains(l.Source, clusterA)
		dstA := strings.Contains(l.DestNode, clusterA)
		srcB := strings.Contains(l.Source, clusterB)
		dstB := strings.Contains(l.DestNode, clusterB)
		if (srcA && dstB) || (srcB && dstA) {
			n++
		}
	}
	return n
}

// nodeIDToPodName strips the control-plane group prefix from a node ID.
// slimctl reports node IDs as "<group>/<pod-name>" (e.g.
// "cluster-b/agntcy-cluster-b-slim-abc123"); the pod name is the part after
// the last "/". Kubernetes pod names never contain "/", so this is safe.
func nodeIDToPodName(nodeID string) string {
	if idx := strings.LastIndex(nodeID, "/"); idx >= 0 {
		return nodeID[idx+1:]
	}
	return nodeID
}

// connectedNodeSet returns the set of node IDs currently reported as Connected.
// Node/control-plane restarts leave stale "Unknown" node records behind, so
// callers must filter against this set to reason about the live topology.
func connectedNodeSet(nodes []nodeEntry) map[string]bool {
	set := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if strings.EqualFold(n.Status, "Connected") {
			set[n.ID] = true
		}
	}
	return set
}

// interClusterLinkEndpoints returns the full node IDs in targetCluster that
// terminate a link connecting clusterA and clusterB. The returned IDs are in
// slimctl "<group>/<pod-name>" form (use nodeIDToPodName for the pod name).
func interClusterLinkEndpoints(links []linkEntry, clusterA, clusterB, targetCluster string) []string {
	var out []string
	for _, l := range links {
		srcA := strings.Contains(l.Source, clusterA)
		dstA := strings.Contains(l.DestNode, clusterA)
		srcB := strings.Contains(l.Source, clusterB)
		dstB := strings.Contains(l.DestNode, clusterB)
		if !((srcA && dstB) || (srcB && dstA)) {
			continue
		}
		if strings.Contains(l.Source, targetCluster) {
			out = append(out, l.Source)
		}
		if strings.Contains(l.DestNode, targetCluster) {
			out = append(out, l.DestNode)
		}
	}
	return out
}

// interClusterLinkApplied reports whether at least one link connecting clusterA
// and clusterB is in APPLIED status.
func interClusterLinkApplied(links []linkEntry, clusterA, clusterB string) bool {
	for _, l := range links {
		srcA := strings.Contains(l.Source, clusterA)
		dstA := strings.Contains(l.DestNode, clusterA)
		srcB := strings.Contains(l.Source, clusterB)
		dstB := strings.Contains(l.DestNode, clusterB)
		if (srcA && dstB) || (srcB && dstA) {
			if strings.EqualFold(l.Status, "APPLIED") {
				return true
			}
		}
	}
	return false
}

// connectedLinksBetween counts links connecting clusterA and clusterB whose
// endpoints are both currently Connected. This ignores stale links left behind
// on Unknown nodes after a restart, unlike linksBetween.
func recordC2TopologyEvidence(scenario string, assertions []string) {
	row := c2EvidenceCase{
		RowID:      "c2-topology-routing",
		Scenario:   scenario,
		Mechanism:  "declarative-routes",
		UseCase:    "Multi-agent flow over fixed, named routes",
		Status:     "verified",
		Assertions: assertions,
	}
	gomega.Expect(upsertC2EvidenceCase(c2EvidenceReportPath(), row)).To(gomega.Succeed())
	logC2EvidenceSummary(row)
	ginkgo.AddReportEntry("C2 Evidence", row.Scenario+"="+row.Status)
}

func connectedLinksBetween(links []linkEntry, connected map[string]bool, clusterA, clusterB string) int {
	n := 0
	for _, l := range links {
		if !connected[l.Source] || !connected[l.DestNode] {
			continue
		}
		srcA := strings.Contains(l.Source, clusterA)
		dstA := strings.Contains(l.DestNode, clusterA)
		srcB := strings.Contains(l.Source, clusterB)
		dstB := strings.Contains(l.DestNode, clusterB)
		if (srcA && dstB) || (srcB && dstA) {
			n++
		}
	}
	return n
}

var _ = ginkgo.Describe("Agntcy slim topology test", func() {
	const (
		joinTimeout    = 3 * time.Minute
		msgTimeout     = 90 * time.Second
		absenceWindow  = 45 * time.Second
		linkTimeout    = 30 * time.Second
		restartTimeout = 5 * time.Minute
	)

	var (
		namespace     string
		clientImage   string
		clientset     kubernetes.Interface
		dynamicClient dynamic.Interface
		ctl           *slimctlClient

		bob, carol     clientPod
		createdClients []clientPod
	)

	ginkgo.Context("Slim topology test", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			var err error
			clientset, err = k8shelper.CreateK8sClientSet()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a client")

			dynamicClient, err = k8shelper.CreateDynamicK8sClient()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to create a dynamic client")

			namespace = os.Getenv("NAMESPACE")
			if namespace == "" {
				namespace = "default"
			}

			clientImage = os.Getenv("CLIENT_IMAGE")
			if clientImage == "" {
				clientImage = "ghcr.io/agntcy/slim/bindings-examples:local"
			}

			topologyConfig := os.Getenv("TOPOLOGY_CONFIG")
			parsed, err := config.ParseTopology(topologyConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to parse topology configuration")
			gomega.Expect(parsed).NotTo(gomega.BeNil(), "topology configuration should not be nil")

			ctl, err = newSlimctlClient()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to reach the control-plane northbound endpoint")
			ginkgo.GinkgoWriter.Printf("Using slimctl server %s\n", ctl.server)
		})

		ginkgo.AfterAll(func() {
			ctx := context.Background()
			for _, c := range createdClients {
				_ = c.CleanupPod(ctx)
				_ = c.CleanupConfigMap(ctx)
			}
			if ctl != nil {
				_, _ = ctl.RemoveLink("cluster-a", "cluster-b", segB)
				_, _ = ctl.RemoveLink("cluster-a", "cluster-c", segC)
				_, _ = ctl.RemoveLink("cluster-b", "cluster-c", segBC)
			}
		})

		ginkgo.When("the control plane and clusters are deployed", func() {
			ginkgo.It("then all clusters and nodes are joined", func() {
				ginkgo.By("waiting for all 5 nodes to report Connected via slimctl")
				gomega.Eventually(func(g gomega.Gomega) {
					nodes, err := ctl.Nodes()
					g.Expect(err).NotTo(gomega.HaveOccurred())
					g.Expect(nodes).To(gomega.HaveLen(5), "expected 1+2+2 nodes registered")
					for _, n := range nodes {
						g.Expect(n.Status).To(gomega.Equal("Connected"), "node %s not connected", n.ID)
					}
				}, joinTimeout, 5*time.Second).Should(gomega.Succeed())

				ginkgo.By("checking all three domains are registered")
				domains, err := ctl.DomainNames()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(domains).To(gomega.ContainElements("cluster-a", "cluster-b", "cluster-c"))
			})
		})

		ginkgo.When("cluster-a links to cluster-b and cluster-c in separate segments and the p2p clients are deployed", func() {
			ginkgo.It("then alice reaches bob and carol, but bob cannot reach carol", func() {
				ginkgo.By("creating isolated segments and links via slimctl")
				_, err := ctl.AddSegment(segB)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				_, err = ctl.AddSegment(segC)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				_, err = ctl.AddLink("cluster-a", "cluster-b", segB)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				_, err = ctl.AddLink("cluster-a", "cluster-c", segC)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				ginkgo.By("waiting for inter-cluster links to become APPLIED")
				gomega.Eventually(func(g gomega.Gomega) {
					links, err := ctl.Links(false)
					g.Expect(err).NotTo(gomega.HaveOccurred())
					g.Expect(interClusterLinkApplied(links, "cluster-a", "cluster-b")).To(gomega.BeTrue())
					g.Expect(interClusterLinkApplied(links, "cluster-a", "cluster-c")).To(gomega.BeTrue())
				}, linkTimeout, 5*time.Second).Should(gomega.Succeed())

				ginkgo.By("deploying bob/carol receivers, then alice senders and the bob->carol probe")
				bob = deployP2PClient(clientset, dynamicClient, namespace, clientImage, p2pClientSpec{
					Name: "bob", Cluster: "cluster-b", LocalName: "org/ns/bob",
				})
				carol = deployP2PClient(clientset, dynamicClient, namespace, clientImage, p2pClientSpec{
					Name: "carol", Cluster: "cluster-c", LocalName: "org/ns/carol",
				})
				aliceToBob := deployP2PClient(clientset, dynamicClient, namespace, clientImage, p2pClientSpec{
					Name: "alice-to-bob", Cluster: "cluster-a", LocalName: "org/ns/alice-to-bob",
					Remote: "org/ns/bob", Message: msgFromAlice,
				})
				aliceToCarol := deployP2PClient(clientset, dynamicClient, namespace, clientImage, p2pClientSpec{
					Name: "alice-to-carol", Cluster: "cluster-a", LocalName: "org/ns/alice-to-carol",
					Remote: "org/ns/carol", Message: msgFromAlice,
				})
				bobToCarol := deployP2PClient(clientset, dynamicClient, namespace, clientImage, p2pClientSpec{
					Name: "bob-to-carol", Cluster: "cluster-b", LocalName: "org/ns/bob-to-carol",
					Remote: "org/ns/carol", Message: msgFromBob,
				})
				createdClients = append(createdClients, bob, carol, aliceToBob, aliceToCarol, bobToCarol)

				ginkgo.By("asserting alice reaches bob")
				found, line, err := bob.WaitForStringWithTimeout("received: "+msgFromAlice, msgTimeout)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(found).To(gomega.BeTrue(), "bob never received alice's message")
				ginkgo.GinkgoWriter.Printf("bob received: %s\n", line)

				ginkgo.By("asserting alice reaches carol")
				found, line, err = carol.WaitForStringWithTimeout("received: "+msgFromAlice, msgTimeout)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(found).To(gomega.BeTrue(), "carol never received alice's message")
				ginkgo.GinkgoWriter.Printf("carol received: %s\n", line)

				ginkgo.By("asserting bob cannot reach carol (segment isolation)")
				found, _, err = carol.WaitForStringWithTimeout("received: "+msgFromBob, absenceWindow)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(found).To(gomega.BeFalse(), "carol unexpectedly received bob's message across isolated segments")

				ginkgo.By("asserting link topology via slimctl")
				links, err := ctl.Links(false)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(interClusterLinkApplied(links, "cluster-a", "cluster-b")).To(gomega.BeTrue(), "cluster-a<->cluster-b link not applied")
				gomega.Expect(interClusterLinkApplied(links, "cluster-a", "cluster-c")).To(gomega.BeTrue(), "cluster-a<->cluster-c link not applied")
				gomega.Expect(interClusterLinkApplied(links, "cluster-b", "cluster-c")).To(gomega.BeFalse(), "unexpected cluster-b<->cluster-c link")

				recordC2TopologyEvidence("isolated-routes", []string{
					fmt.Sprintf("alice delivered to bob (received: %s)", msgFromAlice),
					fmt.Sprintf("alice delivered to carol (received: %s)", msgFromAlice),
					fmt.Sprintf("bob blocked from carol across isolated segments (no received: %s)", msgFromBob),
					"cluster-a<->cluster-b link APPLIED",
					"cluster-a<->cluster-c link APPLIED",
					"no cluster-b<->cluster-c link while segments are isolated",
				})
			})
		})

		ginkgo.When("the topology is modified so cluster-b and cluster-c are linked", func() {
			ginkgo.It("then bob can now reach carol and the link is applied", func() {
				ginkgo.By("linking cluster-b and cluster-c via slimctl")
				_, err := ctl.AddSegment(segBC)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				_, err = ctl.AddLink("cluster-b", "cluster-c", segBC)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				ginkgo.By("waiting for the cluster-b<->cluster-c link to become APPLIED")
				gomega.Eventually(func(g gomega.Gomega) {
					links, err := ctl.Links(false)
					g.Expect(err).NotTo(gomega.HaveOccurred())
					g.Expect(interClusterLinkApplied(links, "cluster-b", "cluster-c")).To(gomega.BeTrue())
				}, linkTimeout, 5*time.Second).Should(gomega.Succeed())

				ginkgo.By("asserting carol now receives bob's messages")
				found, line, err := carol.WaitForStringWithTimeout("received: "+msgFromBob, msgTimeout)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(found).To(gomega.BeTrue(), "carol did not receive bob's message after linking")
				ginkgo.GinkgoWriter.Printf("carol received: %s\n", line)

				recordC2TopologyEvidence("linked-routes", []string{
					"cluster-b<->cluster-c link APPLIED after topology change",
					fmt.Sprintf("bob delivered to carol after link added (received: %s)", msgFromBob),
				})
			})
		})

		ginkgo.When("the gateway node handling the cluster-b<->cluster-c link is restarted", func() {
			ginkgo.It("then the link is restored and bob can still reach carol", func() {
				ctx := context.Background()

				ginkgo.By("identifying the Connected cluster-b node participating in the b<->c link")
				links, err := ctl.Links(false)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				nodes, err := ctl.Nodes()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				connected := connectedNodeSet(nodes)

				var gatewayID string
				for _, ep := range interClusterLinkEndpoints(links, "cluster-b", "cluster-c", "cluster-b") {
					if connected[ep] {
						gatewayID = ep
						break
					}
				}
				gomega.Expect(gatewayID).NotTo(gomega.BeEmpty(), "could not find a Connected cluster-b node for the b<->c link")
				gatewayPod := nodeIDToPodName(gatewayID)
				ginkgo.GinkgoWriter.Printf("restarting cluster-b gateway node %s (pod %s)\n", gatewayID, gatewayPod)

				ginkgo.By("deleting the gateway pod and waiting for the deployment to recover")
				err = k8shelper.DeletePodByName(ctx, clientset, "cluster-b", gatewayPod)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				err = k8shelper.WaitForDeploymentAvailable(ctx, clientset, "cluster-b", clusterDeploymentName("cluster-b"), restartTimeout)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				ginkgo.By("asserting the control plane fails the b<->c link over to a live cluster-b node")
				// The killed node lingers as an "Unknown" record, so require the
				// restored link to terminate on a Connected node other than the one
				// we just deleted (gateway failover picks a live sibling).
				gomega.Eventually(func(g gomega.Gomega) {
					links, err := ctl.Links(false)
					g.Expect(err).NotTo(gomega.HaveOccurred())
					nodes, err := ctl.Nodes()
					g.Expect(err).NotTo(gomega.HaveOccurred())
					connected := connectedNodeSet(nodes)

					restored := false
					for _, ep := range interClusterLinkEndpoints(links, "cluster-b", "cluster-c", "cluster-b") {
						if connected[ep] && ep != gatewayID {
							restored = true
							break
						}
					}
					g.Expect(restored).To(gomega.BeTrue(), "b<->c link not restored on a live cluster-b node")
				}, restartTimeout, 5*time.Second).Should(gomega.Succeed())

				ginkgo.By("proving post-recovery delivery with a fresh unique-message sender")
				verify := deployP2PClient(clientset, dynamicClient, namespace, clientImage, p2pClientSpec{
					Name: "bob-verify", Cluster: "cluster-b", LocalName: "org/ns/bob-verify",
					Remote: "org/ns/carol", Message: msgFromBobVerify,
				})
				ginkgo.DeferCleanup(func(ctx context.Context) {
					_ = verify.CleanupPod(ctx)
					_ = verify.CleanupConfigMap(ctx)
				})

				found, line, err := carol.WaitForStringWithTimeout("received: "+msgFromBobVerify, msgTimeout)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(found).To(gomega.BeTrue(), "carol did not receive bob's message after node restart")
				ginkgo.GinkgoWriter.Printf("carol received after restart: %s\n", line)

				recordC2TopologyEvidence("route-survives-restart", []string{
					"gateway node restart: b<->c link restored on a live cluster-b node",
					fmt.Sprintf("bob delivered to carol after gateway restart (received: %s)", msgFromBobVerify),
				})
			})
		})

		ginkgo.When("the control plane is restarted", func() {
			ginkgo.It("then the links remain intact", func() {
				ctx := context.Background()

				ginkgo.By("recording the applied link count before restart")
				before, err := ctl.AppliedLinkCount()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(before).To(gomega.BeNumerically(">", 0))

				ginkgo.By("rollout-restarting the control plane and waiting for it to recover")
				err = k8shelper.RestartDeployment(ctx, clientset, controllerAdminNS, controllerServiceName)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				err = k8shelper.WaitForDeploymentAvailable(ctx, clientset, controllerAdminNS, controllerServiceName, restartTimeout)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				ginkgo.By("rediscovering the northbound endpoint after restart")
				gomega.Eventually(func(g gomega.Gomega) {
					ctl, err = newSlimctlClient()
					g.Expect(err).NotTo(gomega.HaveOccurred())
					_, err = ctl.Nodes()
					g.Expect(err).NotTo(gomega.HaveOccurred())
				}, restartTimeout, 5*time.Second).Should(gomega.Succeed())

				ginkgo.By("asserting all three inter-cluster links persist on reconnected nodes")
				// After the restart the nodes re-register; persisted links must be
				// re-applied on Connected endpoints. Checking against the Connected
				// set ignores any stale "Unknown" nodes from earlier specs.
				gomega.Eventually(func(g gomega.Gomega) {
					links, err := ctl.Links(false)
					g.Expect(err).NotTo(gomega.HaveOccurred())
					nodes, err := ctl.Nodes()
					g.Expect(err).NotTo(gomega.HaveOccurred())
					connected := connectedNodeSet(nodes)
					g.Expect(connectedLinksBetween(links, connected, "cluster-a", "cluster-b")).To(gomega.BeNumerically(">", 0))
					g.Expect(connectedLinksBetween(links, connected, "cluster-a", "cluster-c")).To(gomega.BeNumerically(">", 0))
					g.Expect(connectedLinksBetween(links, connected, "cluster-b", "cluster-c")).To(gomega.BeNumerically(">", 0))
				}, restartTimeout, 5*time.Second).Should(gomega.Succeed())
			})
		})
	})
})
