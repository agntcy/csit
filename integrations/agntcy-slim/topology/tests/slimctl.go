// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// controllerServiceName matches fullnameOverride in controller-values.yaml.
	controllerServiceName   = "slim-control"
	controllerAdminNS       = "admin"
	controllerNorthboundKey = "north"
)

// slimctlClient shells out to the slimctl binary against the control-plane
// northbound API.
type slimctlClient struct {
	bin        string
	server     string // e.g. http://1.2.3.4:50051
	configPath string // isolated config file passed via --config
}

// nodeEntry is one row of `slimctl controller node list`.
type nodeEntry struct {
	ID     string
	Group  string
	Status string
}

// linkEntry is one row of `slimctl controller link list`.
type linkEntry struct {
	LinkID     string
	Source     string
	DestNode   string
	DestEndpnt string
	Status     string
}

// newSlimctlClient builds a client. The northbound server URL is taken from the
// SLIMCTL_SERVER env var when set, otherwise it is discovered from the
// LoadBalancer service in the admin namespace.
func newSlimctlClient(clientset kubernetes.Interface, timeout time.Duration) (*slimctlClient, error) {
	bin := os.Getenv("SLIMCTL")
	if bin == "" {
		bin = "slimctl"
	}

	server := os.Getenv("SLIMCTL_SERVER")
	if server == "" {
		host, port, err := discoverNorthboundEndpoint(clientset, timeout)
		if err != nil {
			return nil, err
		}
		server = fmt.Sprintf("http://%s:%d", host, port)
	}
	// Explicit http:// makes slimctl use plaintext gRPC, matching the
	// controller's northbound tls.insecure: true.
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}

	// Write an isolated config file so slimctl does not pick up an
	// incompatible ~/.slimctl/config.yaml from the host.
	tmp, err := os.CreateTemp("", "slimctl-topology-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create slimctl config: %w", err)
	}
	if _, err := tmp.WriteString(fmt.Sprintf("endpoint: %q\n", server)); err != nil {
		_ = tmp.Close()
		return nil, fmt.Errorf("failed to write slimctl config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("failed to close slimctl config: %w", err)
	}

	return &slimctlClient{bin: bin, server: server, configPath: tmp.Name()}, nil
}

// discoverNorthboundEndpoint polls the LoadBalancer service until it has an
// external address, returning the address and the northbound port.
func discoverNorthboundEndpoint(clientset kubernetes.Interface, timeout time.Duration) (string, int32, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		svc, err := clientset.CoreV1().Services(controllerAdminNS).Get(context.TODO(), controllerServiceName, metav1.GetOptions{})
		if err != nil {
			lastErr = err
			time.Sleep(3 * time.Second)
			continue
		}

		port := northboundPort(svc)
		host := loadBalancerHost(svc)
		if host != "" {
			return host, port, nil
		}
		lastErr = fmt.Errorf("service %s/%s has no LoadBalancer ingress address yet", controllerAdminNS, controllerServiceName)
		time.Sleep(3 * time.Second)
	}
	return "", 0, fmt.Errorf("timed out discovering northbound endpoint: %w", lastErr)
}

func northboundPort(svc *corev1.Service) int32 {
	for _, p := range svc.Spec.Ports {
		if p.Name == controllerNorthboundKey || p.Port == 50051 {
			return p.Port
		}
	}
	// Fall back to the documented northbound port.
	return 50051
}

func loadBalancerHost(svc *corev1.Service) string {
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			return ing.IP
		}
		if ing.Hostname != "" {
			return ing.Hostname
		}
	}
	return ""
}

// run executes a slimctl controller subcommand and returns combined output.
// The command and its output are echoed to the Ginkgo report for debugging.
func (c *slimctlClient) run(args ...string) (string, error) {
	full := append([]string{"--config", c.configPath, "--server", c.server, "controller"}, args...)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.bin, full...)
	out, err := cmd.CombinedOutput()
	ginkgo.GinkgoWriter.Printf("$ %s %s\n%s\n", c.bin, strings.Join(full, " "), string(out))
	if err != nil {
		return string(out), fmt.Errorf("slimctl %s failed: %w\noutput:\n%s", strings.Join(full, " "), err, string(out))
	}
	return string(out), nil
}

// Nodes returns the registered nodes from `controller node list`.
func (c *slimctlClient) Nodes() ([]nodeEntry, error) {
	out, err := c.run("node", "list")
	if err != nil {
		return nil, err
	}
	rows := dataRowsAfterSeparator(out)
	entries := make([]nodeEntry, 0, len(rows))
	for _, row := range rows {
		fields := strings.Fields(row)
		if len(fields) < 3 {
			continue
		}
		entries = append(entries, nodeEntry{
			ID:     fields[0],
			Group:  fields[1],
			Status: fields[2],
		})
	}
	return entries, nil
}

// GroupNames returns the group names from `controller group list`.
func (c *slimctlClient) GroupNames() ([]string, error) {
	out, err := c.run("group", "list")
	if err != nil {
		return nil, err
	}
	rows := dataRowsAfterSeparator(out)
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		fields := strings.Fields(row)
		if len(fields) == 0 {
			continue
		}
		names = append(names, fields[0])
	}
	return names, nil
}

// Links returns the links from `controller link list`. When all is true,
// pending/failed/deleted links are included as well.
func (c *slimctlClient) Links(all bool) ([]linkEntry, error) {
	args := []string{"link", "list"}
	if all {
		args = append(args, "-a")
	}
	out, err := c.run(args...)
	if err != nil {
		return nil, err
	}
	rows := dataRowsAfterSeparator(out)
	entries := make([]linkEntry, 0, len(rows))
	for _, row := range rows {
		fields := strings.Fields(row)
		if len(fields) < 5 {
			continue
		}
		entries = append(entries, linkEntry{
			LinkID:     fields[0],
			Source:     fields[1],
			DestNode:   fields[2],
			DestEndpnt: fields[3],
			Status:     fields[len(fields)-1],
		})
	}
	return entries, nil
}

// AppliedLinkCount returns the number of links currently in APPLIED status.
func (c *slimctlClient) AppliedLinkCount() (int, error) {
	links, err := c.Links(false)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, l := range links {
		if strings.EqualFold(l.Status, "APPLIED") {
			count++
		}
	}
	return count, nil
}

// AddSegment creates a routing-domain segment.
func (c *slimctlClient) AddSegment(name string) (string, error) {
	return c.run("segment", "add", name)
}

// AddLink links two groups, optionally within a segment.
func (c *slimctlClient) AddLink(groupA, groupB, segment string) (string, error) {
	args := []string{"link", "add", groupA, groupB}
	if segment != "" {
		args = append(args, "-s", segment)
	}
	return c.run(args...)
}

// RemoveLink removes a link between two groups.
func (c *slimctlClient) RemoveLink(groupA, groupB, segment string) (string, error) {
	args := []string{"link", "remove", groupA, groupB}
	if segment != "" {
		args = append(args, "-s", segment)
	}
	return c.run(args...)
}

var separatorRe = regexp.MustCompile(`^-{3,}$`)

// dataRowsAfterSeparator returns the trimmed data rows of a slimctl table.
// slimctl prints a header row followed by a dashed separator line; every
// non-empty line after the separator is a data row.
func dataRowsAfterSeparator(output string) []string {
	lines := strings.Split(output, "\n")
	var rows []string
	seenSeparator := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !seenSeparator {
			if separatorRe.MatchString(trimmed) {
				seenSeparator = true
			}
			continue
		}
		if trimmed == "" {
			continue
		}
		rows = append(rows, trimmed)
	}
	return rows
}
